package wallet

import (
	"fmt"
	"github.com/kurumiimari/gohan/bio"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/txscript"
	"github.com/pkg/errors"
)

type WitnessFactory func(coins []*chain.Coin, tx *chain.Transaction) (*chain.Witness, error)

type TxBuilder struct {
	Coins     []*chain.Coin
	Outputs   []*chain.Output
	Witnesses []*chain.Witness
	Version   uint32
	Locktime  uint32
}

func (b *TxBuilder) AddCoin(coin *chain.Coin) {
	b.Coins = append(b.Coins, coin)
}

func (b *TxBuilder) AddOutput(output *chain.Output) {
	b.Outputs = append(b.Outputs, output)
}

func (b *TxBuilder) Sign(ring Keyring) error {
	tx := b.Build()
	for i, coin := range b.Coins {
		if i < len(b.Witnesses) {
			continue
		}

		key, err := ring.PrivateKey(coin.Derivation...)
		if err != nil {
			return err
		}
		wit, err := txscript.P2PKHWitnessSignature(tx, i, coin.Value, key)
		if err != nil {
			return err
		}
		b.Witnesses = append(b.Witnesses, wit)
	}
	return nil
}

func (b *TxBuilder) Build() *chain.Transaction {
	tx := &chain.Transaction{
		Version:   b.Version,
		Inputs:    make([]*chain.Input, len(b.Coins)),
		Witnesses: make([]*chain.Witness, len(b.Witnesses)),
		Outputs:   b.Outputs,
		LockTime:  b.Locktime,
	}
	for i, coin := range b.Coins {
		tx.Inputs[i] = &chain.Input{
			Prevout: &chain.Outpoint{
				Hash:  coin.Prevout.Hash,
				Index: coin.Prevout.Index,
			},
			Sequence: chain.DefaultSequence,
		}
	}
	for i, wit := range b.Witnesses {
		tx.Witnesses[i] = wit
	}
	return tx
}

func (b *TxBuilder) EstimateSize() int {
	var est int
	est += 4
	est += bio.SizeVarint(len(b.Coins))
	est += 4
	est += len(b.Coins) * 150
	est += bio.SizeVarint(len(b.Outputs))
	for _, out := range b.Outputs {
		est += out.Size()
	}
	return est
}

func (b *TxBuilder) Fund(fundingCoins []*chain.Coin, changeAddress *chain.Address, feeRate uint64) error {
	if feeRate == 0 {
		panic("fee rate is zero")
	}

	size := uint64(b.EstimateSize())
	fee := size * feeRate

	var totalIn uint64
	usedCoins := make(map[string]bool)
	for _, coin := range b.Coins {
		totalIn += coin.Value
		usedCoins[fmt.Sprintf("%x%d", coin.Prevout.Hash, coin.Prevout.Index)] = true
	}

	var totalOut uint64
	for _, out := range b.Outputs {
		totalOut += out.Value
	}

	totalWithFee := totalOut + fee
	if totalIn >= totalWithFee {
		if totalIn == totalWithFee {
			return nil
		}

		changeOut := &chain.Output{
			Value:    0,
			Address:  changeAddress,
			Covenant: chain.EmptyCovenant,
		}
		addlFee := uint64(changeOut.Size()) * feeRate
		newTotalWithFee := totalWithFee + addlFee
		if totalIn > newTotalWithFee {
			changeOut.Value = totalIn - newTotalWithFee
			b.Outputs = append(b.Outputs, changeOut)
		}

		return nil
	}

	newCoins := make([]*chain.Coin, len(b.Coins))
	copy(newCoins, b.Coins)

	for _, coin := range fundingCoins {
		if usedCoins[fmt.Sprintf("%x%d", coin.Prevout.Hash, coin.Prevout.Index)] {
			continue
		}
		totalIn += coin.Value
		totalWithFee += 150 * feeRate
		newCoins = append(newCoins, coin)

		if totalIn >= totalWithFee {
			change := totalIn - totalWithFee
			if change == 0 {
				b.Coins = newCoins
				return nil
			}

			changeOut := &chain.Output{
				Value:    0,
				Address:  changeAddress,
				Covenant: chain.EmptyCovenant,
			}
			addlFee := uint64(changeOut.Size()) * feeRate
			newTotalWithFee := totalWithFee + addlFee
			if totalIn < newTotalWithFee {
				continue
			}

			changeOut.Value = change - addlFee
			b.Coins = newCoins
			b.Outputs = append(b.Outputs, changeOut)
			return nil
		}
	}

	return errors.New("insufficient funds")
}
