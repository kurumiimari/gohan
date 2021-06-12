// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"fmt"
	"github.com/kurumiimari/gohan/chain"

	"github.com/btcsuite/btcd/btcec"
)

// RawTxInWitnessSignature returns the serialized ECDSA signature for the input
// idx of the given transaction, with the hashType appended to it.
func RawTxInWitnessSignature(tx *chain.Transaction, sigHashes *TxSigHashes, idx int,
	amt uint64, subScript []byte, hashType SigHashType,
	key *btcec.PrivateKey) ([]byte, error) {

	parsedScript, err := parseScript(subScript)
	if err != nil {
		return nil, fmt.Errorf("cannot parse output script: %v", err)
	}

	hash, err := calcWitnessSignatureHash(parsedScript, sigHashes, hashType, tx,
		idx, amt)
	if err != nil {
		return nil, err
	}

	signature, err := key.Sign(hash)
	if err != nil {
		return nil, fmt.Errorf("cannot sign tx input: %s", err)
	}

	return append(chain.SerializeSignature(signature), byte(hashType)), nil
}

func WitnessSignature(tx *chain.Transaction, sigHashes *TxSigHashes, idx int, amt uint64,
	subscript []byte, hashType SigHashType, privKey *btcec.PrivateKey) ([]byte, []byte, error) {

	sig, err := RawTxInWitnessSignature(tx, sigHashes, idx, amt, subscript,
		hashType, privKey)
	if err != nil {
		return nil, nil, err
	}

	pk := (*btcec.PublicKey)(&privKey.PublicKey)

	return sig, pk.SerializeCompressed(), nil
}

func P2PKHWitnessSignature(tx *chain.Transaction, idx int, amt uint64, privKey *btcec.PrivateKey) (*chain.Witness, error) {
	script, err := NewP2PKHScript(blake160(privKey.PubKey().SerializeCompressed()))
	if err != nil {
		return nil, err
	}
	sig, pub, err := WitnessSignature(tx, NewTxSigHashes(tx), idx, amt, script, SigHashAll, privKey)
	if err != nil {
		return nil, err
	}

	return &chain.Witness{
		Items: [][]byte{
			sig,
			pub,
		},
	}, nil
}

func HIP1PresignWitnessSignature(tx *chain.Transaction, idx int, amt uint64, privKey *btcec.PrivateKey) (*chain.Witness, error) {
	script, err := NewHIP1LockingScript(privKey.PubKey().SerializeCompressed())
	if err != nil {
		return nil, err
	}
	sig, _, err := WitnessSignature(tx, NewTxSigHashes(tx), idx, amt, script, SigHashAnyOneCanPay|SigHashSingleReverse, privKey)

	return &chain.Witness{
		Items: [][]byte{
			sig,
			script,
		},
	}, nil
}

func HIP1CancelWitnessSignature(tx *chain.Transaction, idx int, amt uint64, privKey *btcec.PrivateKey) (*chain.Witness, error) {
	script, err := NewHIP1LockingScript(privKey.PubKey().SerializeCompressed())
	if err != nil {
		return nil, err
	}
	sig, _, err := WitnessSignature(tx, NewTxSigHashes(tx), idx, amt, script, SigHashAnyOneCanPay|SigHashSingle, privKey)

	return &chain.Witness{
		Items: [][]byte{
			sig,
			script,
		},
	}, nil
}
