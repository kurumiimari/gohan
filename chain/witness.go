package chain

import (
	"encoding/hex"
	"encoding/json"
	"github.com/btcsuite/btcd/btcec"
	"github.com/kurumiimari/gohan/bio"
	"github.com/pkg/errors"
	"io"
)

type WitnessType string

const (
	WitnessTypeP2PKH = "p2pkh"
)

type Witness struct {
	Items [][]byte
}

func NewP2PKHWitness(sig *btcec.Signature, sigOpts SigOpt, pub *btcec.PublicKey) *Witness {
	sigB := make([]byte, 65)
	copy(sigB[:], SerializeSignature(sig))
	sigB[64] = byte(sigOpts)

	return &Witness{
		Items: [][]byte{
			sigB,
			pub.SerializeCompressed(),
		},
	}
}

func (wit *Witness) WriteTo(w io.Writer) (int64, error) {
	g := bio.NewGuardWriter(w)
	bio.WriteVarint(g, uint64(len(wit.Items)))
	for _, item := range wit.Items {
		bio.WriteVarBytes(g, item)
	}
	return g.N, errors.Wrap(g.Err, "error writing witness")
}

func (wit *Witness) ReadFrom(r io.Reader) (int64, error) {
	g := bio.NewGuardReader(r)
	count, _ := bio.ReadVarint(g)
	var items [][]byte
	for i := 0; i < int(count); i++ {
		item, _ := bio.ReadVarBytes(g)
		items = append(items, item)
	}
	if g.Err != nil {
		return g.N, errors.Wrap(g.Err, "error reading witness")
	}
	wit.Items = items
	return g.N, nil
}

func (wit *Witness) MarshalJSON() ([]byte, error) {
	var items []string
	for _, item := range wit.Items {
		items = append(items, hex.EncodeToString(item))
	}
	return json.Marshal(items)
}

func (wit *Witness) UnmarshalJSON(bytes []byte) error {
	tmp := make([]string, 0)
	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return err
	}

	wit.Items = make([][]byte, len(tmp))
	for i := 0; i < len(tmp); i++ {
		b, err := hex.DecodeString(tmp[i])
		if err != nil {
			return err
		}
		wit.Items[i] = b
	}
	return nil
}
