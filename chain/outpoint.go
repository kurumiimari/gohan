package chain

import (
	"encoding/hex"
	"encoding/json"
	"github.com/kurumiimari/gohan/bio"
	"github.com/pkg/errors"
	"io"
)

type Outpoint struct {
	Hash  []byte
	Index uint32
}

func (o *Outpoint) WriteTo(w io.Writer) (int64, error) {
	g := bio.NewGuardWriter(w)
	bio.WriteFixedBytes(g, o.Hash, 32)
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

func (o *Outpoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Hash  string `json:"hash"`
		Index uint32 `json:"index"`
	}{
		Hash:  hex.EncodeToString(o.Hash),
		Index: o.Index,
	})
}

func (o *Outpoint) UnmarshalJSON(bytes []byte) error {
	tmp := struct {
		Hash  string `json:"hash"`
		Index uint32 `json:"index"`
	}{}

	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return err
	}

	hash, err := hex.DecodeString(tmp.Hash)
	if err != nil {
		return err
	}
	o.Hash = hash
	o.Index = tmp.Index
	return nil
}
