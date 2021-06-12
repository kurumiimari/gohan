package chain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/kurumiimari/gohan/bio"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"io"
)

type Transaction struct {
	Version   uint32     `json:"version"`
	Inputs    []*Input   `json:"vin"`
	Outputs   []*Output  `json:"vout"`
	Locktime  uint32     `json:"locktime"`
	Witnesses []*Witness `json:"txinwitness"`
}

func (tx *Transaction) ID() []byte {
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
	bio.WriteUint32LE(g, tx.Locktime)
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
	tx.Locktime = lockTime
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

func (tx *Transaction) SignatureHash(coins []*Coin, index int, script *Script, sigOpts SigOpt) []byte {
	if len(coins) != len(tx.Inputs) {
		panic("mismatched coins and inputs")
	}
	if index > len(coins)-1 {
		panic(fmt.Sprintf("no coin for index %d", index))
	}

	coin := coins[index]
	input := tx.Inputs[index]
	if !coin.Prevout.Equal(input.Prevout) {
		panic("mis-matched coins and prevouts")
	}

	prevouts, _ := blake2b.New256(nil)
	sequences, _ := blake2b.New256(nil)
	outputs, _ := blake2b.New256(nil)

	if sigOpts&SighashAnyoneCanPay == 0 {
		for _, input := range tx.Inputs {
			input.Prevout.WriteTo(prevouts)
		}
	}

	rawSigopt := sigOpts & 0x1f

	if sigOpts&SighashAnyoneCanPay == 0 &&
		rawSigopt != SighashSingle &&
		rawSigopt != SighashSingleReverse &&
		rawSigopt != SighashNone {
		for _, input := range tx.Inputs {
			bio.WriteUint32LE(sequences, input.Sequence)
		}
	}

	if rawSigopt != SighashSingle &&
		rawSigopt != SighashSingleReverse &&
		rawSigopt != SighashNone {
		for _, o := range tx.Outputs {
			o.WriteTo(outputs)
		}
	} else if rawSigopt == SighashSingle {
		if index < len(tx.Outputs) {
			out := tx.Outputs[index]
			out.WriteTo(outputs)
		}
	} else if rawSigopt == SighashSingleReverse {
		if index < len(tx.Outputs) {
			out := tx.Outputs[len(tx.Outputs) - 1 - index]
			out.WriteTo(outputs)
		}
	}

	sighash, _ := blake2b.New256(nil)
	bio.WriteUint32LE(sighash, tx.Version)
	bio.WriteRawBytes(sighash, prevouts.Sum(nil))
	bio.WriteRawBytes(sighash, sequences.Sum(nil))
	bio.WriteRawBytes(sighash, input.Prevout.Hash)
	bio.WriteUint32LE(sighash, input.Prevout.Index)
	script.WriteTo(sighash)
	bio.WriteUint64LE(sighash, coin.Value)
	bio.WriteUint32LE(sighash, input.Sequence)
	bio.WriteRawBytes(sighash, outputs.Sum(nil))
	bio.WriteUint32LE(sighash, tx.Locktime)
	bio.WriteUint32LE(sighash, uint32(sigOpts))
	return sighash.Sum(nil)
}
