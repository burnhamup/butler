package elfprops

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/elefant"
	"github.com/itchio/wharf/eos"
)

var args = struct {
	path *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("elfprops", "(Advanced) Gives information about an ELF binary").Hidden()
	args.path = cmd.Arg("path", "The ELF binary to analyze").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(*args.path))
}

func Do(path string) error {
	f, err := eos.Open(path)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer f.Close()

	props, err := elefant.Probe(f, nil)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Result(props)

	return nil
}
