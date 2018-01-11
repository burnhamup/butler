package probe

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/bsdiff"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

var args = struct {
	patch    *string
	fullpath *bool
	deep     *bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("probe", "(Advanced) Show statistics about a patch file").Hidden()
	args.patch = cmd.Arg("patch", "Path of the patch to analyze").Required().String()
	args.fullpath = cmd.Flag("fullpath", "Display full path names").Bool()
	args.deep = cmd.Flag("deep", "Analyze the top N changed files further").Bool()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.patch))
}

func Do(ctx *mansion.Context, patch string) error {
	topFileIndices, err := doPrimaryAnalysis(ctx, patch)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if *args.deep {
		err = doDeepAnalysis(ctx, patch, topFileIndices)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}

func doPrimaryAnalysis(ctx *mansion.Context, patch string) ([]patchStat, error) {
	patchReader, err := eos.Open(patch)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	defer patchReader.Close()

	stats, err := patchReader.Stat()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	comm.Opf("patch:  %s", humanize.IBytes(uint64(stats.Size())))

	cr := counter.NewReaderCallback(func(count int64) {
		comm.Progress(float64(count) / float64(stats.Size()))
	}, patchReader)

	rctx := wire.NewReadContext(cr)
	err = rctx.ExpectMagic(pwr.PatchMagic)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	header := &pwr.PatchHeader{}
	err = rctx.ReadMessage(header)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	rctx, err = pwr.DecompressWire(rctx, header.Compression)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	target := &tlc.Container{}
	err = rctx.ReadMessage(target)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	source := &tlc.Container{}
	err = rctx.ReadMessage(source)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	comm.Logf("  before: %s in %s", humanize.IBytes(uint64(target.Size)), target.Stats())
	comm.Logf("   after: %s in %s", humanize.IBytes(uint64(target.Size)), source.Stats())

	startTime := time.Now()

	comm.StartProgressWithTotalBytes(stats.Size())

	var patchStats []patchStat

	sh := &pwr.SyncHeader{}
	rop := &pwr.SyncOp{}
	bc := &bsdiff.Control{}

	var numBsdiff = 0
	var numRsync = 0
	for fileIndex, f := range source.Files {
		sh.Reset()
		err = rctx.ReadMessage(sh)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		stat := patchStat{
			fileIndex: int64(fileIndex),
			freshData: f.Size,
			algo:      sh.Type,
		}

		if sh.FileIndex != int64(fileIndex) {
			return nil, fmt.Errorf("malformed patch: expected file %d, got %d", fileIndex, sh.FileIndex)
		}

		switch sh.Type {
		case pwr.SyncHeader_RSYNC:
			{
				numRsync++
				readingOps := true
				var pos int64

				for readingOps {
					rop.Reset()

					err = rctx.ReadMessage(rop)
					if err != nil {
						return nil, errors.Wrap(err, 0)
					}

					switch rop.Type {
					case pwr.SyncOp_BLOCK_RANGE:
						fixedSize := (rop.BlockSpan - 1) * pwr.BlockSize
						lastIndex := rop.BlockIndex + (rop.BlockSpan - 1)
						lastSize := pwr.ComputeBlockSize(f.Size, lastIndex)
						totalSize := (fixedSize + lastSize)
						stat.freshData -= totalSize
						pos += totalSize
					case pwr.SyncOp_DATA:
						totalSize := int64(len(rop.Data))
						if ctx.Verbose {
							comm.Debugf("%s fresh data at %s (%d-%d)", humanize.IBytes(uint64(totalSize)), humanize.IBytes(uint64(pos)),
								pos, pos+totalSize)
						}
						pos += totalSize
					case pwr.SyncOp_HEY_YOU_DID_IT:
						readingOps = false
					}
				}
			}
		case pwr.SyncHeader_BSDIFF:
			{
				numBsdiff++
				readingOps := true

				bh := &pwr.BsdiffHeader{}
				err = rctx.ReadMessage(bh)
				if err != nil {
					return nil, errors.Wrap(err, 0)
				}

				for readingOps {
					bc.Reset()

					err = rctx.ReadMessage(bc)
					if err != nil {
						return nil, errors.Wrap(err, 0)
					}

					for _, b := range bc.Add {
						if b == 0 {
							stat.freshData--
						}
					}

					if bc.Eof {
						readingOps = false
					}
				}

				err = rctx.ReadMessage(rop)
				if err != nil {
					return nil, errors.Wrap(err, 0)
				}

				if rop.Type != pwr.SyncOp_HEY_YOU_DID_IT {
					msg := fmt.Sprintf("expected HEY_YOU_DID_IT, got %s", rop.Type)
					return nil, errors.New(msg)
				}
			}
		}

		patchStats = append(patchStats, stat)
	}

	comm.EndProgress()

	sort.Sort(byDecreasingFreshData(patchStats))

	var totalFresh int64
	for _, stat := range patchStats {
		totalFresh += stat.freshData
	}

	var freshThreshold = int64(0.9 * float64(totalFresh))
	var printedFresh int64

	duration := time.Since(startTime)

	perSec := humanize.IBytes(uint64(float64(stats.Size()) / duration.Seconds()))
	comm.Statf("Analyzed %s @ %s/s (%s total)", humanize.IBytes(uint64(stats.Size())), perSec, duration)
	comm.Statf("%d bsdiff series, %d rsync series", numBsdiff, numRsync)

	var numTouched = 0
	var numTotal = 0
	var naivePatchSize int64
	for _, stat := range patchStats {
		numTotal++
		if stat.freshData > 0 {
			numTouched++
			f := source.Files[stat.fileIndex]
			naivePatchSize += f.Size
		}
	}

	comm.Logf("")
	comm.Statf("Most of the fresh data is in the following files:")

	for i, stat := range patchStats {
		f := source.Files[stat.fileIndex]
		name := f.Path
		if !*args.fullpath {
			name = filepath.Base(name)
		}

		comm.Logf("  - %s / %s in %s (%.2f%% changed, %s)",
			humanize.IBytes(uint64(stat.freshData)),
			humanize.IBytes(uint64(f.Size)),
			name,
			float64(stat.freshData)/float64(f.Size)*100.0,
			stat.algo)

		printedFresh += stat.freshData

		if i >= 10 || printedFresh >= freshThreshold {
			break
		}
	}

	comm.Logf("")

	var kind = "simple"
	if numBsdiff > 0 {
		kind = "optimized"
	}
	comm.Statf("All in all, that's %s of fresh data in a %s %s patch",
		humanize.IBytes(uint64(totalFresh)),
		humanize.IBytes(uint64(stats.Size())),
		kind,
	)
	comm.Logf(" (%d/%d files are changed by this patch, they weigh a total of %s)", numTouched, numTotal, humanize.IBytes(uint64(naivePatchSize)))

	return patchStats, nil
}

type deepDiveContext struct {
	target *tlc.Container
	source *tlc.Container
	rctx   *wire.ReadContext
}

func doDeepAnalysis(ctx *mansion.Context, patch string, topFileIndices []int64) error {
	comm.Logf("")
	comm.Statf("Now deep-diving into top %d files", len(topFileIndices))

	topIndexMap := make(map[int64]bool)
	for _, i := range topFileIndices {
		topIndexMap[i] = true
	}

	patchReader, err := eos.Open(patch)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	defer patchReader.Close()

	stats, err := patchReader.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Opf("patch:  %s", humanize.IBytes(uint64(stats.Size())))

	cr := counter.NewReaderCallback(func(count int64) {
		comm.Progress(float64(count) / float64(stats.Size()))
	}, patchReader)

	rctx := wire.NewReadContext(cr)
	err = rctx.ExpectMagic(pwr.PatchMagic)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	header := &pwr.PatchHeader{}
	err = rctx.ReadMessage(header)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	rctx, err = pwr.DecompressWire(rctx, header.Compression)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	target := &tlc.Container{}
	err = rctx.ReadMessage(target)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	source := &tlc.Container{}
	err = rctx.ReadMessage(source)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	ddc := &deepDiveContext{
		target: target,
		source: source,
		rctx:   rctx,
	}

	sh := &pwr.SyncHeader{}

	for fileIndex := range source.Files {
		sh.Reset()
		err = rctx.ReadMessage(sh)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if sh.FileIndex != int64(fileIndex) {
			return fmt.Errorf("malformed patch: expected file %d, got %d", fileIndex, sh.FileIndex)
		}

		if topIndexMap[sh.FileIndex] {
			err = ddc.analyzeSeries(sh)
		} else {
			comm.Debugf("Skipping %d...", sh.FileIndex)
			err = ddc.skipSeries(sh)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}

	return nil
}

func (ddc *deepDiveContext) analyzeSeries(sh *pwr.SyncHeader) error {
	comm.Logf("")
	comm.Logf("=============================================")
	comm.Logf("")

	switch sh.Type {
	case pwr.SyncHeader_RSYNC:
		return ddc.analyzeRsync(sh)
	case pwr.SyncHeader_BSDIFF:
		return ddc.analyzeBsdiff(sh)
	default:
		return fmt.Errorf("don't know how to analyze series of type %d", sh.Type)
	}
}

func (ddc *deepDiveContext) analyzeRsync(sh *pwr.SyncHeader) error {
	f := ddc.source.Files[sh.FileIndex]
	comm.Logf("Analyzing rsync series for '%s'", f.Path)

	rctx := ddc.rctx
	readingOps := true

	rop := &pwr.SyncOp{}

	targetBlocks := make(map[int64]int64)

	for readingOps {
		rop.Reset()

		err := rctx.ReadMessage(rop)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		switch rop.Type {
		case pwr.SyncOp_BLOCK_RANGE:
			i := rop.FileIndex
			targetBlocks[i] = targetBlocks[i] + rop.BlockSpan
		case pwr.SyncOp_DATA:
			// TODO: something
		case pwr.SyncOp_HEY_YOU_DID_IT:
			readingOps = false
		}
	}

	if len(targetBlocks) > 0 {
		comm.Statf("Sourcing from '%d' blocks total: ", len(targetBlocks))
		for i, numBlocks := range targetBlocks {
			tf := ddc.target.Files[i]
			comm.Statf("Taking %d blocks from '%s'", numBlocks, tf.Path)
		}
	} else {
		comm.Statf("Entirely fresh data!")
	}

	return nil
}

func (ddc *deepDiveContext) analyzeBsdiff(sh *pwr.SyncHeader) error {
	f := ddc.source.Files[sh.FileIndex]
	comm.Logf("Analyzing bsdiff series for '%s'", f.Path)

	rctx := ddc.rctx
	readingOps := true

	bh := &pwr.BsdiffHeader{}
	err := rctx.ReadMessage(bh)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	tf := ddc.target.Files[bh.TargetIndex]
	comm.Logf("Diffed against target file '%s'", tf.Path)
	if tf.Path == f.Path {
		comm.Logf("Same path, can do in-place!")
	}

	bc := &bsdiff.Control{}

	var oldpos int64
	var newpos int64

	var pristine int64
	var bestUnchanged int64

	clearUnchanged := func() {
		if bestUnchanged > 1024*1024 {
			comm.Logf("%s contiguous unchanged block ending at from %s to %s",
				humanize.IBytes(uint64(bestUnchanged)),
				humanize.IBytes(uint64(newpos-bestUnchanged)),
				humanize.IBytes(uint64(newpos)),
			)
		}
		bestUnchanged = 0
	}

	for readingOps {
		bc.Reset()

		err = rctx.ReadMessage(bc)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if bc.Eof {
			readingOps = false
			break
		}

		if len(bc.Add) > 0 {
			if oldpos == newpos {
				var unchanged int64
				for _, b := range bc.Add {
					oldpos++
					newpos++
					if b == 0 {
						unchanged++
						bestUnchanged++
					} else {
						clearUnchanged()
					}
				}
				pristine += unchanged
			} else {
				oldpos += int64(len(bc.Add))
				newpos += int64(len(bc.Add))
			}
		}

		if len(bc.Copy) > 0 {
			clearUnchanged()
			newpos += int64(len(bc.Copy))
		}

		oldpos += bc.Seek
	}

	rop := &pwr.SyncOp{}

	err = rctx.ReadMessage(rop)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if rop.Type != pwr.SyncOp_HEY_YOU_DID_IT {
		msg := fmt.Sprintf("expected HEY_YOU_DID_IT, got %s", rop.Type)
		return errors.New(msg)
	}

	comm.Statf("%s / %s pristine after patch application", humanize.IBytes(uint64(pristine)), humanize.IBytes(uint64(tf.Size)))
	comm.Statf("File went from %s to %s", humanize.IBytes(uint64(tf.Size)), humanize.IBytes(uint64(f.Size)))

	return nil
}

func (ddc *deepDiveContext) skipSeries(sh *pwr.SyncHeader) error {
	rctx := ddc.rctx
	rop := &pwr.SyncOp{}
	bc := &bsdiff.Control{}

	switch sh.Type {
	case pwr.SyncHeader_RSYNC:
		{
			readingOps := true
			for readingOps {
				rop.Reset()

				err := rctx.ReadMessage(rop)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				if rop.Type == pwr.SyncOp_HEY_YOU_DID_IT {
					// yay, we did it!
					readingOps = false
				}
			}
		}
	case pwr.SyncHeader_BSDIFF:
		{
			bh := &pwr.BsdiffHeader{}
			err := rctx.ReadMessage(bh)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			readingOps := true
			for readingOps {
				bc.Reset()

				err := rctx.ReadMessage(bc)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				if bc.Eof {
					readingOps = false
				}
			}

			rop.Reset()
			err = rctx.ReadMessage(rop)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if rop.Type != pwr.SyncOp_HEY_YOU_DID_IT {
				// oh noes, we didn't do it
				return errors.New("missing HEY_YOU_DID_IT after bsdiff series")
			}
		}
	default:
		return fmt.Errorf("dunno how to skip series of type %d", sh.Type)
	}

	return nil
}

type patchStat struct {
	fileIndex int64
	freshData int64
	algo      pwr.SyncHeader_Type
}

type byDecreasingFreshData []patchStat

func (s byDecreasingFreshData) Len() int {
	return len(s)
}

func (s byDecreasingFreshData) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byDecreasingFreshData) Less(i, j int) bool {
	return s[j].freshData < s[i].freshData
}
