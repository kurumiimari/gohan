package chain

import (
	"fmt"
	"github.com/kurumiimari/gohan/bio"
	"github.com/kurumiimari/gohan/gcrypto"
	"github.com/pkg/errors"
	"io"
)

type Outpoint struct {
	Hash  gcrypto.Hash `json:"hash"`
	Index uint32       `json:"index"`
}

func (o *Outpoint) WriteTo(w io.Writer) (int64, error) {
	g := bio.NewGuardWriter(w)
	o.Hash.WriteTo(g)
	bio.WriteUint32LE(g, o.Index)
	return g.N, errors.Wrap(g.Err, "error writing outpoint")
}

func (o *Outpoint) ReadFrom(r io.Reader) (int64, error) {
	g := bio.NewGuardReader(r)
	hash, _ := bio.ReadFixedBytes(g, 32)
	index, _ := bio.ReadUint32LE(g)
	if g.Err != nil {
		return g.N, errors.Wrap(g.Err, "error reading outpoint")
	}
	o.Hash = hash
	o.Index = index
	return g.N, nil
}

func (o *Outpoint) Equal(other *Outpoint) bool {
	return o.Hash.Equal(other.Hash) &&
		o.Index == other.Index
}

func (o *Outpoint) String() string {
	return fmt.Sprintf("%s/%d", o.Hash, o.Index)
}
