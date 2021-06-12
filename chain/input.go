package chain

import (
	"github.com/kurumiimari/gohan/bio"
	"github.com/pkg/errors"
	"io"
)

type Input struct {
	Prevout  *Outpoint `json:"prevout"`
	Sequence uint32    `json:"sequence"`
}

func (i *Input) WriteTo(w io.Writer) (int64, error) {
	g := bio.NewGuardWriter(w)
	i.Prevout.WriteTo(g)
	bio.WriteUint32LE(g, i.Sequence)
	return g.N, errors.Wrap(g.Err, "error writing input")
}

func (i *Input) ReadFrom(r io.Reader) (int64, error) {
	g := bio.NewGuardReader(r)
	prevout := new(Outpoint)
	prevout.ReadFrom(g)
	seq, _ := bio.ReadUint32LE(g)
	if g.Err != nil {
		return g.N, errors.Wrap(g.Err, "error reading input")
	}
	i.Prevout = prevout
	i.Sequence = seq
	return g.N, nil
}
