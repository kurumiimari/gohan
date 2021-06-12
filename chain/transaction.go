package chain

import (
	"bytes"
	"encoding/hex"
	"github.com/kurumiimari/gohan/bio"
	"github.com/kurumiimari/gohan/gcrypto"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"io"
)

type Transaction struct {
	Version   uint32     `json:"version"`
	Inputs    []*Input   `json:"vin"`
	Outputs   []*Output  `json:"vout"`
	LockTime  uint32     `json:"locktime"`
	Witnesses []*Witness `json:"txinwitness"`
}

func (tx *Transaction) ID() gcrypto.Hash {
	h, _ := blake2b.New256(nil)
	if _, err := tx.writeTo(h, false); err != nil {
		panic(err)
	}
	return h.Sum(nil)
}

func (tx *Transaction) IDHex() string {
	return hex.EncodeToString(tx.ID())
}

func (tx *Transaction) WriteTo(w io.Writer) (int64, error) {
	return tx.writeTo(w, true)
}

func (tx *Transaction) writeTo(w io.Writer, includeWitnesses bool) (int64, error) {
	g := bio.NewGuardWriter(w)
	bio.WriteUint32LE(g, tx.Version)
	bio.WriteVarint(g, uint64(len(tx.Inputs)))
	for _, input := range tx.Inputs {
		input.WriteTo(g)
	}
	bio.WriteVarint(g, uint64(len(tx.Outputs)))
	for _, output := range tx.Outputs {
		output.WriteTo(g)
	}
	bio.WriteUint32LE(g, tx.LockTime)
	if includeWitnesses {
		for _, wit := range tx.Witnesses {
			wit.WriteTo(g)
		}
	}
	return g.N, errors.Wrap(g.Err, "error writing transaction")
}

func (tx *Transaction) ReadFrom(r io.Reader) (n int64, err error) {
	g := bio.NewGuardReader(r)
	version, _ := bio.ReadUint32LE(g)
	inCount, _ := bio.ReadVarint(g)
	var inputs []*Input
	for i := 0; i < int(inCount); i++ {
		input := new(Input)
		input.ReadFrom(g)
		inputs = append(inputs, input)
	}
	outCount, _ := bio.ReadVarint(g)
	var outputs []*Output
	for i := 0; i < int(outCount); i++ {
		output := new(Output)
		output.ReadFrom(g)
		outputs = append(outputs, output)
	}
	lockTime, _ := bio.ReadUint32LE(g)
	var witnesses []*Witness
	for i := 0; i < int(inCount); i++ {
		witness := new(Witness)
		witness.ReadFrom(g)
		witnesses = append(witnesses, witness)
	}
	if g.Err != nil {
		return g.N, errors.Wrap(g.Err, "error reading transaction")
	}
	tx.Version = version
	tx.Inputs = inputs
	tx.Outputs = outputs
	tx.LockTime = lockTime
	tx.Witnesses = witnesses
	return g.N, nil
}

func (tx *Transaction) Bytes() []byte {
	buf := new(bytes.Buffer)
	if _, err := tx.WriteTo(buf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
