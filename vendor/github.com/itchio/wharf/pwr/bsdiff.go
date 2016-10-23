package pwr

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"

	"github.com/itchio/wharf/wire"
)

func swap(a []int, i, j int) { a[i], a[j] = a[j], a[i] }

// Ternary-Split Quicksort, cf. http://www.larsson.dogma.net/ssrev-tr.pdf
func split(I, V []int, start, length, h int) {
	var i, j, k, x, jj, kk int

	if length < 16 {
		for k = start; k < start+length; k += j {
			j = 1
			x = V[I[k]+h]
			for i = 1; k+i < start+length; i++ {
				if V[I[k+i]+h] < x {
					x = V[I[k+i]+h]
					j = 0
				}
				if V[I[k+i]+h] == x {
					swap(I, k+i, k+j)
					j++
				}
			}
			for i = 0; i < j; i++ {
				V[I[k+i]] = k + j - 1
			}
			if j == 1 {
				I[k] = -1
			}
		}
		return
	}

	x = V[I[start+length/2]+h]
	jj = 0
	kk = 0
	for i = start; i < start+length; i++ {
		if V[I[i]+h] < x {
			jj++
		}
		if V[I[i]+h] == x {
			kk++
		}
	}
	jj += start
	kk += jj

	i = start
	j = 0
	k = 0
	for i < jj {
		if V[I[i]+h] < x {
			i++
		} else if V[I[i]+h] == x {
			swap(I, i, jj+j)
			j++
		} else {
			swap(I, i, kk+k)
			k++
		}
	}

	for jj+j < kk {
		if V[I[jj+j]+h] == x {
			j++
		} else {
			swap(I, jj+j, kk+k)
			k++
		}
	}

	if jj > start {
		split(I, V, start, jj-start, h)
	}

	for i = 0; i < kk-jj; i++ {
		V[I[jj+i]] = kk - 1
	}
	if jj == kk-1 {
		I[jj] = -1
	}

	if start+length > kk {
		split(I, V, kk, start+length-kk, h)
	}
}

// Faster Suffix Sorting, see: http://www.larsson.dogma.net/ssrev-tr.pdf
// Output `I` is a sorted suffix array.
func qsufsort(obuf []byte) []int {
	var buckets [256]int
	var i, h int
	I := make([]int, len(obuf)+1)
	V := make([]int, len(obuf)+1)

	for _, c := range obuf {
		buckets[c]++
	}
	for i = 1; i < 256; i++ {
		buckets[i] += buckets[i-1]
	}
	copy(buckets[1:], buckets[:])
	buckets[0] = 0

	for i, c := range obuf {
		buckets[c]++
		I[buckets[c]] = i
	}

	I[0] = len(obuf)
	for i, c := range obuf {
		V[i] = buckets[c]
	}

	V[len(obuf)] = 0
	for i = 1; i < 256; i++ {
		if buckets[i] == buckets[i-1]+1 {
			I[buckets[i]] = -1
		}
	}
	I[0] = -1

	for h = 1; I[0] != -(len(obuf) + 1); h += h {
		var n int
		for i = 0; i < len(obuf)+1; {
			if I[i] < 0 {
				n -= I[i]
				i -= I[i]
			} else {
				if n != 0 {
					I[i-n] = -n
				}
				n = V[I[i]] + 1 - i
				split(I, V, i, n, h)
				i += n
				n = 0
			}
		}
		if n != 0 {
			I[i-n] = -n
		}
	}

	for i = 0; i < len(obuf)+1; i++ {
		I[V[i]] = i
	}
	return I
}

// Returns the number of bytes common to a and b
func matchlen(a, b []byte) (i int) {
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return i
}

func search(I []int, obuf, nbuf []byte, st, en int) (pos, n int) {
	if en-st < 2 {
		x := matchlen(obuf[I[st]:], nbuf)
		y := matchlen(obuf[I[en]:], nbuf)

		if x > y {
			return I[st], x
		}
		return I[en], y
	}

	x := st + (en-st)/2
	if bytes.Compare(obuf[I[x]:], nbuf) < 0 {
		return search(I, obuf, nbuf, x, en)
	}
	return search(I, obuf, nbuf, st, x)
}

// BSDiff computes the difference between old and new, according to the bsdiff
// algorithm, and writes the result to patch.
func BSDiff(old, new io.Reader, patch *wire.WriteContext) error {
	obuf, err := ioutil.ReadAll(old)
	if err != nil {
		return err
	}

	nbuf, err := ioutil.ReadAll(new)
	if err != nil {
		return err
	}

	var lenf int
	I := qsufsort(obuf)
	db := make([]byte, len(nbuf))
	eb := make([]byte, len(nbuf))
	var dblen, eblen int

	bsdc := &BsdiffControl{}

	// Compute the differences, writing ctrl as we go
	var scan, pos, length int
	var lastscan, lastpos, lastoffset int
	for scan < len(nbuf) {
		var oldscore int
		scan += length
		for scsc := scan; scan < len(nbuf); scan++ {
			pos, length = search(I, obuf, nbuf[scan:], 0, len(obuf))

			for ; scsc < scan+length; scsc++ {
				if scsc+lastoffset < len(obuf) &&
					obuf[scsc+lastoffset] == nbuf[scsc] {
					oldscore++
				}
			}

			if (length == oldscore && length != 0) || length > oldscore+8 {
				break
			}

			if scan+lastoffset < len(obuf) && obuf[scan+lastoffset] == nbuf[scan] {
				oldscore--
			}
		}

		if length != oldscore || scan == len(nbuf) {
			var s, Sf int
			lenf = 0
			for i := 0; lastscan+i < scan && lastpos+i < len(obuf); {
				if obuf[lastpos+i] == nbuf[lastscan+i] {
					s++
				}
				i++
				if s*2-i > Sf*2-lenf {
					Sf = s
					lenf = i
				}
			}

			lenb := 0
			if scan < len(nbuf) {
				var s, Sb int
				for i := 1; (scan >= lastscan+i) && (pos >= i); i++ {
					if obuf[pos-i] == nbuf[scan-i] {
						s++
					}
					if s*2-i > Sb*2-lenb {
						Sb = s
						lenb = i
					}
				}
			}

			if lastscan+lenf > scan-lenb {
				overlap := (lastscan + lenf) - (scan - lenb)
				s := 0
				Ss := 0
				lens := 0
				for i := 0; i < overlap; i++ {
					if nbuf[lastscan+lenf-overlap+i] == obuf[lastpos+lenf-overlap+i] {
						s++
					}
					if nbuf[scan-lenb+i] == obuf[pos-lenb+i] {
						s--
					}
					if s > Ss {
						Ss = s
						lens = i + 1
					}
				}

				lenf += lens - overlap
				lenb -= lens
			}

			for i := 0; i < lenf; i++ {
				db[dblen+i] = nbuf[lastscan+i] - obuf[lastpos+i]
			}
			for i := 0; i < (scan-lenb)-(lastscan+lenf); i++ {
				eb[eblen+i] = nbuf[lastscan+lenf+i]
			}

			dblen += lenf
			eblen += (scan - lenb) - (lastscan + lenf)

			bsdc.Add = int64(lenf)
			bsdc.Copy = int64((scan - lenb) - (lastscan + lenf))
			bsdc.Seek = int64((pos - lenb) - (lastpos + lenf))

			err := patch.WriteMessage(bsdc)
			if err != nil {
				return err
			}

			lastscan = scan - lenb
			lastpos = pos - lenb
			lastoffset = pos - scan
		}
	}

	// Write sentinel control message
	bsdc.Reset()
	bsdc.Add = -1
	err = patch.WriteMessage(bsdc)
	if err != nil {
		return err
	}

	// Write diff data
	bsdd := &BsdiffDiff{
		Data: db[:dblen],
	}

	err = patch.WriteMessage(bsdd)
	if err != nil {
		return err
	}

	// Write extra data
	bsed := &BsdiffExtra{
		Data: eb[:eblen],
	}

	err = patch.WriteMessage(bsed)
	if err != nil {
		return err
	}

	return nil
}

// ErrCorrupt indicates that a patch is corrupted, most often that it would produce a longer file
// than specified
var ErrCorrupt = errors.New("corrupt patch")

// BSPatch applies patch to old, according to the bspatch algorithm,
// and writes the result to new.
func BSPatch(old io.Reader, new io.Writer, newSize int64, patch *wire.ReadContext) error {
	var ctrlOps []BsdiffControl

	ctrlOp := &BsdiffControl{}

	for {
		ctrlOp.Reset()
		err := patch.ReadMessage(ctrlOp)
		if err != nil {
			return err
		}

		if ctrlOp.Add == -1 {
			break
		}

		ctrlOps = append(ctrlOps, BsdiffControl{
			Add:  ctrlOp.Add,
			Seek: ctrlOp.Seek,
			Copy: ctrlOp.Copy,
		})
	}

	diffMessage := &BsdiffDiff{}
	err := patch.ReadMessage(diffMessage)
	if err != nil {
		return err
	}
	diff := diffMessage.Data

	var diffOffset int64

	extraMessage := &BsdiffExtra{}
	err = patch.ReadMessage(extraMessage)
	if err != nil {
		return err
	}
	extra := extraMessage.Data

	var extraOffset int64

	obuf, err := ioutil.ReadAll(old)
	if err != nil {
		return err
	}

	nbuf := make([]byte, newSize)

	var oldpos, newpos int64

	for _, ctrl := range ctrlOps {
		// Sanity-check
		if newpos+ctrl.Add > newSize {
			return ErrCorrupt
		}

		// Read diff string
		copy(nbuf[newpos:newpos+ctrl.Add], diff[diffOffset:diffOffset+ctrl.Add])
		diffOffset += ctrl.Add

		// Add old data to diff string
		for i := int64(0); i < ctrl.Add; i++ {
			nbuf[newpos+i] += obuf[oldpos+i]
		}

		// Adjust pointers
		newpos += ctrl.Add
		oldpos += ctrl.Add

		// Sanity-check
		if newpos+ctrl.Copy > newSize {
			return ErrCorrupt
		}

		// Read extra string
		copy(nbuf[newpos:newpos+ctrl.Copy], extra[extraOffset:extraOffset+ctrl.Copy])
		extraOffset += ctrl.Copy

		// Adjust pointers
		newpos += ctrl.Copy
		oldpos += ctrl.Seek
	}

	// Write the new file
	for len(nbuf) > 0 {
		n, err := new.Write(nbuf)
		if err != nil {
			return err
		}
		nbuf = nbuf[n:]
	}

	return nil
}
