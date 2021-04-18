package chain

import (
	"github.com/kurumiimari/gohan/bio"
	"github.com/pkg/errors"
	"io"
)

type Output struct {
	Value    uint64    `json:"value"`
	Address  *Address  `json:"address"`
	Covenant *Covenant `json:"covenant"`
}

func (o *Output) Size() int {
	return 8 + o.Covenant.Size()
}

func (o *Output) WriteTo(w io.Writer) (int64, error) {
	g := bio.NewGuardWriter(w)
	bio.WriteUint64LE(g, o.Value)
	o.Address.WriteTo(g)
	o.Covenant.WriteTo(g)
	return g.N, errors.Wrap(g.Err, "error writing output")
}

func (o *Output) ReadFrom(r io.Reader) (int64, error) {
	g := bio.NewGuardReader(r)
	address := new(Address)
	cov := new(Covenant)
	value, _ := bio.ReadUint64LE(g)
	address.ReadFrom(g)
	cov.ReadFrom(g)
	if g.Err != nil {
		return g.N, errors.Wrap(g.Err, "error reading output")
	}
	o.Value = value
	o.Covenant = cov
	o.Address = address
	return g.N, nil
}
