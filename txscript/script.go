// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"bytes"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/kurumiimari/gohan/bio"
	"github.com/kurumiimari/gohan/chain"
	"golang.org/x/crypto/blake2b"
)

// SigHashType represents hash type bits at the end of a signature.
type SigHashType uint32

// Hash type bits from the end of a signature.
const (
	SigHashAll           SigHashType = 0x1
	SigHashNone          SigHashType = 0x2
	SigHashSingle        SigHashType = 0x3
	SigHashSingleReverse SigHashType = 0x4
	SigHashAnyOneCanPay  SigHashType = 0x80

	// sigHashMask defines the number of bits of the hash type which is used
	// to identify which outputs are signed.
	sigHashMask = 0x1f
)

// isSmallInt returns whether or not the opcode is considered a small integer,
// which is an OP_0, or OP_1 through OP_16.
func isSmallInt(op *opcode) bool {
	if op.value == OP_0 || (op.value >= OP_1 && op.value <= OP_16) {
		return true
	}
	return false
}

// isScriptHash returns true if the script passed is a pay-to-script-hash
// transaction, false otherwise.
func isScriptHash(pops []parsedOpcode) bool {
	return len(pops) == 3 &&
		pops[0].opcode.value == OP_HASH160 &&
		pops[1].opcode.value == OP_DATA_20 &&
		pops[2].opcode.value == OP_EQUAL
}

// isWitnessScriptHash returns true if the passed script is a
// pay-to-witness-script-hash transaction, false otherwise.
func isWitnessScriptHash(pops []parsedOpcode) bool {
	return len(pops) == 2 &&
		pops[0].opcode.value == OP_0 &&
		pops[1].opcode.value == OP_DATA_32
}

// isWitnessPubKeyHash returns true if the passed script is a
// pay-to-witness-pubkey-hash, and false otherwise.
func isWitnessPubKeyHash(pops []parsedOpcode) bool {
	return len(pops) == 2 &&
		pops[0].opcode.value == OP_0 &&
		pops[1].opcode.value == OP_DATA_20
}

// parseScriptTemplate is the same as parseScript but allows the passing of the
// template list for testing purposes.  When there are parse errors, it returns
// the list of parsed opcodes up to the point of failure along with the error.
func parseScriptTemplate(script []byte, opcodes *[256]opcode) ([]parsedOpcode, error) {
	retScript := make([]parsedOpcode, 0, len(script))
	var err error
	for i := 0; i < len(script); {
		instr := script[i]
		op := &opcodes[instr]
		pop := parsedOpcode{opcode: op}
		i, err = pop.checkParseableInScript(script, i)
		if err != nil {
			return retScript, err
		}

		retScript = append(retScript, pop)
	}

	return retScript, nil
}

// parseScript preparses the script in bytes into a list of parsedOpcodes while
// applying a number of sanity checks.
func parseScript(script []byte) ([]parsedOpcode, error) {
	return parseScriptTemplate(script, &opcodeArray)
}

// unparseScript reversed the action of parseScript and returns the
// parsedOpcodes as a list of bytes
func unparseScript(pops []parsedOpcode) ([]byte, error) {
	script := make([]byte, 0, len(pops))
	for _, pop := range pops {
		b, err := pop.bytes()
		if err != nil {
			return nil, err
		}
		script = append(script, b...)
	}
	return script, nil
}

// DisasmString formats a disassembled script for one line printing.  When the
// script fails to parse, the returned string will contain the disassembled
// script up to the point the failure occurred along with the string '[error]'
// appended.  In addition, the reason the script failed to parse is returned
// if the caller wants more information about the failure.
func DisasmString(buf []byte) (string, error) {
	var disbuf bytes.Buffer
	opcodes, err := parseScript(buf)
	for _, pop := range opcodes {
		disbuf.WriteString(pop.print(true))
		disbuf.WriteByte(' ')
	}
	if disbuf.Len() > 0 {
		disbuf.Truncate(disbuf.Len() - 1)
	}
	if err != nil {
		disbuf.WriteString("[error]")
	}
	return disbuf.String(), err
}

// removeOpcode will remove any opcode matching ``opcode'' from the opcode
// stream in pkscript
func removeOpcode(pkscript []parsedOpcode, opcode byte) []parsedOpcode {
	retScript := make([]parsedOpcode, 0, len(pkscript))
	for _, pop := range pkscript {
		if pop.opcode.value != opcode {
			retScript = append(retScript, pop)
		}
	}
	return retScript
}

// canonicalPush returns true if the object is either not a push instruction
// or the push instruction contained wherein is matches the canonical form
// or using the smallest instruction to do the job. False otherwise.
func canonicalPush(pop parsedOpcode) bool {
	opcode := pop.opcode.value
	data := pop.data
	dataLen := len(pop.data)
	if opcode > OP_16 {
		return true
	}

	if opcode < OP_PUSHDATA1 && opcode > OP_0 && (dataLen == 1 && data[0] <= 16) {
		return false
	}
	if opcode == OP_PUSHDATA1 && dataLen < OP_PUSHDATA1 {
		return false
	}
	if opcode == OP_PUSHDATA2 && dataLen <= 0xff {
		return false
	}
	if opcode == OP_PUSHDATA4 && dataLen <= 0xffff {
		return false
	}
	return true
}

// removeOpcodeByData will return the script minus any opcodes that would push
// the passed data to the stack.
func removeOpcodeByData(pkscript []parsedOpcode, data []byte) []parsedOpcode {
	retScript := make([]parsedOpcode, 0, len(pkscript))
	for _, pop := range pkscript {
		if !canonicalPush(pop) || !bytes.Contains(pop.data, data) {
			retScript = append(retScript, pop)
		}
	}
	return retScript

}

// calcHashPrevOuts calculates a single hash of all the previous outputs
// (txid:index) referenced within the passed transaction. This calculated hash
// can be re-used when validating all inputs spending segwit outputs, with a
// signature hash type of SigHashAll. This allows validation to re-use previous
// hashing computation, reducing the complexity of validating SigHashAll inputs
// from  O(N^2) to O(N).
func calcHashPrevOuts(tx *chain.Transaction) (out chainhash.Hash) {
	h, _ := blake2b.New256(nil)
	for _, input := range tx.Inputs {
		input.Prevout.WriteTo(h)
	}
	copy(out[:], h.Sum(nil))
	return out
}

// calcHashSequence computes an aggregated hash of each of the sequence numbers
// within the inputs of the passed transaction. This single hash can be re-used
// when validating all inputs spending segwit outputs, which include signatures
// using the SigHashAll sighash type. This allows validation to re-use previous
// hashing computation, reducing the complexity of validating SigHashAll inputs
// from O(N^2) to O(N).
func calcHashSequence(tx *chain.Transaction) (out chainhash.Hash) {
	h, _ := blake2b.New256(nil)
	for _, input := range tx.Inputs {
		bio.WriteUint32LE(h, input.Sequence)
	}
	copy(out[:], h.Sum(nil))
	return out
}

// calcHashOutputs computes a hash digest of all outputs created by the
// transaction encoded using the wire format. This single hash can be re-used
// when validating all inputs spending witness programs, which include
// signatures using the SigHashAll sighash type. This allows computation to be
// cached, reducing the total hashing complexity from O(N^2) to O(N).
func calcHashOutputs(tx *chain.Transaction) (out chainhash.Hash) {
	h, _ := blake2b.New256(nil)
	for _, o := range tx.Outputs {
		o.WriteTo(h)
	}
	copy(out[:], h.Sum(nil))
	return out
}

// calcWitnessSignatureHash computes the sighash digest of a transaction's
// segwit input using the new, optimized digest calculation algorithm defined
// in BIP0143: https://github.com/bitcoin/bips/blob/master/bip-0143.mediawiki.
// This function makes use of pre-calculated sighash fragments stored within
// the passed HashCache to eliminate duplicate hashing computations when
// calculating the final digest, reducing the complexity from O(N^2) to O(N).
// Additionally, signatures now cover the input value of the referenced unspent
// output. This allows offline, or hardware wallets to compute the exact amount
// being spent, in addition to the final transaction fee. In the case the
// wallet if fed an invalid input amount, the real sighash will differ causing
// the produced signature to be invalid.
func calcWitnessSignatureHash(subScript []parsedOpcode, sigHashes *TxSigHashes,
	hashType SigHashType, tx *chain.Transaction, idx int, amt uint64) ([]byte, error) {

	// As a sanity check, ensure the passed input index for the transaction
	// is valid.
	if idx > len(tx.Inputs)-1 {
		return nil, fmt.Errorf("idx %d but %d txins", idx, len(tx.Inputs))
	}

	// We'll utilize this buffer throughout to incrementally calculate
	// the signature hash for this transaction.
	sigHash, _ := blake2b.New256(nil)

	// First write out, then encode the transaction's version number.
	bio.WriteUint32LE(sigHash, tx.Version)

	// Next write out the possibly pre-calculated hashes for the sequence
	// numbers of all inputs, and the hashes of the previous outs for all
	// outputs.
	var zeroHash chainhash.Hash

	// If anyone can pay isn't active, then we can use the cached
	// hashPrevOuts, otherwise we just write zeroes for the prev outs.
	if hashType&SigHashAnyOneCanPay == 0 {
		sigHash.Write(sigHashes.HashPrevOuts[:])
	} else {
		sigHash.Write(zeroHash[:])
	}

	// If the sighash isn't anyone can pay, single, single reverse, or none, the use the
	// cached hash sequences, otherwise write all zeroes for the
	// hashSequence.
	if hashType&SigHashAnyOneCanPay == 0 &&
		hashType&sigHashMask != SigHashSingle &&
		hashType&sigHashMask != SigHashSingleReverse &&
		hashType&sigHashMask != SigHashNone {
		sigHash.Write(sigHashes.HashSequence[:])
	} else {
		sigHash.Write(zeroHash[:])
	}

	txIn := tx.Inputs[idx]

	// Next, write the outpoint being spent.
	txIn.Prevout.Hash.WriteTo(sigHash)
	bio.WriteUint32LE(sigHash, txIn.Prevout.Index)

	rawScript, _ := unparseScript(subScript)
	sigHash.Write([]byte{byte(len(rawScript))})
	sigHash.Write(rawScript)

	// Next, add the input amount, and sequence number of the input being
	// signed.
	bio.WriteUint64LE(sigHash, amt)
	bio.WriteUint32LE(sigHash, txIn.Sequence)

	// If the current signature mode isn't single, or none, then we can
	// re-use the pre-generated hashoutputs sighash fragment. Otherwise,
	// we'll serialize and add only the target output index to the signature
	// pre-image.
	if hashType&SigHashSingle != SigHashSingle &&
		hashType&SigHashSingleReverse != SigHashSingleReverse &&
		hashType&SigHashNone != SigHashNone {
		sigHash.Write(sigHashes.HashOutputs[:])
	} else if hashType&sigHashMask == SigHashSingle {
		h, _ := blake2b.New256(nil)
		tx.Outputs[idx].WriteTo(h)
		sigHash.Write(h.Sum(nil))
	} else if hashType&sigHashMask == SigHashSingleReverse {
		h, _ := blake2b.New256(nil)
		tx.Outputs[len(tx.Outputs)-1-idx].WriteTo(h)
		sigHash.Write(h.Sum(nil))
	} else {
		sigHash.Write(zeroHash[:])
	}

	// Finally, write out the transaction's locktime, and the sig hash
	// type.
	bio.WriteUint32LE(sigHash, tx.LockTime)
	bio.WriteUint32LE(sigHash, uint32(hashType))
	return sigHash.Sum(nil), nil
}

// CalcWitnessSigHash computes the sighash digest for the specified input of
// the target transaction observing the desired sig hash type.
func CalcWitnessSigHash(script []byte, sigHashes *TxSigHashes, hType SigHashType,
	tx *chain.Transaction, idx int, amt uint64) ([]byte, error) {

	parsedScript, err := parseScript(script)
	if err != nil {
		return nil, fmt.Errorf("cannot parse output script: %v", err)
	}

	return calcWitnessSignatureHash(parsedScript, sigHashes, hType, tx, idx,
		amt)
}
