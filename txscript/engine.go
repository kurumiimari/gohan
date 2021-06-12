// Copyright (c) 2013-2018 The btcsuite developers
// Copyright (c) 2015-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"bytes"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/kurumiimari/gohan/chain"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

// Engine is the virtual machine that executes scripts.
type Engine struct {
	scripts         [][]parsedOpcode
	scriptIdx       int
	scriptOff       int
	lastCodeSep     int
	dstack          stack // data stack
	astack          stack // alt stack
	tx              *chain.Transaction
	txIdx           int
	condStack       []int
	numOps          int
	flags           ScriptFlags
	sigCache        *SigCache
	hashCache       *TxSigHashes
	bip16           bool     // treat execution as pay-to-script-hash
	savedFirstStack [][]byte // stack from first script for bip16 scripts
	witnessVersion  int
	witnessProgram  []byte
	inputAmount     uint64
}

// hasFlag returns whether the script engine instance has the passed flag set.
func (vm *Engine) hasFlag(flag ScriptFlags) bool {
	return vm.flags&flag == flag
}

// isBranchExecuting returns whether or not the current conditional branch is
// actively executing.  For example, when the data stack has an OP_FALSE on it
// and an OP_IF is encountered, the branch is inactive until an OP_ELSE or
// OP_ENDIF is encountered.  It properly handles nested conditionals.
func (vm *Engine) isBranchExecuting() bool {
	if len(vm.condStack) == 0 {
		return true
	}
	return vm.condStack[len(vm.condStack)-1] == OpCondTrue
}

// executeOpcode peforms execution on the passed opcode.  It takes into account
// whether or not it is hidden by conditionals, but some rules still must be
// tested in this case.
func (vm *Engine) executeOpcode(pop *parsedOpcode) error {
	// Disabled opcodes are fail on program counter.
	if pop.isDisabled() {
		str := fmt.Sprintf("attempt to execute disabled opcode %s",
			pop.opcode.name)
		return scriptError(ErrDisabledOpcode, str)
	}

	// Always-illegal opcodes are fail on program counter.
	if pop.alwaysIllegal() {
		str := fmt.Sprintf("attempt to execute reserved opcode %s",
			pop.opcode.name)
		return scriptError(ErrReservedOpcode, str)
	}

	// Note that this includes OP_RESERVED which counts as a push operation.
	if pop.opcode.value > OP_16 {
		vm.numOps++
		if vm.numOps > MaxOpsPerScript {
			str := fmt.Sprintf("exceeded max operation limit of %d",
				MaxOpsPerScript)
			return scriptError(ErrTooManyOperations, str)
		}

	} else if len(pop.data) > MaxScriptElementSize {
		str := fmt.Sprintf("element size %d exceeds max allowed size %d",
			len(pop.data), MaxScriptElementSize)
		return scriptError(ErrElementTooBig, str)
	}

	// Nothing left to do when this is not a conditional opcode and it is
	// not in an executing branch.
	if !vm.isBranchExecuting() && !pop.isConditional() {
		return nil
	}

	// Ensure all executed data push opcodes use the minimal encoding when
	// the minimal data verification flag is set.
	if vm.dstack.verifyMinimalData && vm.isBranchExecuting() &&
		pop.opcode.value >= 0 && pop.opcode.value <= OP_PUSHDATA4 {

		if err := pop.checkMinimalDataPush(); err != nil {
			return err
		}
	}

	return pop.opcode.opfunc(pop, vm)
}

// disasm is a helper function to produce the output for DisasmPC and
// DisasmScript.  It produces the opcode prefixed by the program counter at the
// provided position in the script.  It does no error checking and leaves that
// to the caller to provide a valid offset.
func (vm *Engine) disasm(scriptIdx int, scriptOff int) string {
	return fmt.Sprintf("%02x:%04x: %s", scriptIdx, scriptOff,
		vm.scripts[scriptIdx][scriptOff].print(false))
}

// validPC returns an error if the current script position is valid for
// execution, nil otherwise.
func (vm *Engine) validPC() error {
	if vm.scriptIdx >= len(vm.scripts) {
		str := fmt.Sprintf("past input scripts %v:%v %v:xxxx",
			vm.scriptIdx, vm.scriptOff, len(vm.scripts))
		return scriptError(ErrInvalidProgramCounter, str)
	}
	if vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
		str := fmt.Sprintf("past input scripts %v:%v %v:%04d",
			vm.scriptIdx, vm.scriptOff, vm.scriptIdx,
			len(vm.scripts[vm.scriptIdx]))
		return scriptError(ErrInvalidProgramCounter, str)
	}
	return nil
}

// curPC returns either the current script and offset, or an error if the
// position isn't valid.
func (vm *Engine) curPC() (script int, off int, err error) {
	err = vm.validPC()
	if err != nil {
		return 0, 0, err
	}
	return vm.scriptIdx, vm.scriptOff, nil
}

// isWitnessVersionActive returns true if a witness program was extracted
// during the initialization of the Engine, and the program's version matches
// the specified version.
func (vm *Engine) isWitnessVersionActive(version uint) bool {
	return vm.witnessProgram != nil && uint(vm.witnessVersion) == version
}

// verifyWitnessProgram validates the stored witness program using the passed
// witness as input.
func (vm *Engine) verifyWitnessProgram(witness [][]byte) error {
	if vm.isWitnessVersionActive(0) {
		switch len(vm.witnessProgram) {
		case payToWitnessPubKeyHashDataSize:
			// The witness stack should consist of exactly two
			// items: the signature, and the pubkey.
			if len(witness) != 2 {
				err := fmt.Sprintf("should have exactly two "+
					"items in witness, instead have %v", len(witness))
				return scriptError(ErrWitnessProgramMismatch, err)
			}

			// Now we'll resume execution as if it were a regular
			// p2pkh transaction.
			pkScript, err := NewP2PKHScript(vm.witnessProgram)
			if err != nil {
				return err
			}
			pops, err := parseScript(pkScript)
			if err != nil {
				return err
			}

			// Set the stack to the provided witness stack, then
			// append the pkScript generated above as the next
			// script to execute.
			vm.scripts = append(vm.scripts, pops)
			vm.SetStack(witness)

		case payToWitnessScriptHashDataSize: // P2WSH
			// Additionally, The witness stack MUST NOT be empty at
			// this point.
			if len(witness) == 0 {
				return scriptError(ErrWitnessProgramEmpty, "witness "+
					"program empty passed empty witness")
			}

			// Obtain the witness script which should be the last
			// element in the passed stack. The size of the script
			// MUST NOT exceed the max script size.
			witnessScript := witness[len(witness)-1]
			if len(witnessScript) > MaxScriptSize {
				str := fmt.Sprintf("witnessScript size %d "+
					"is larger than max allowed size %d",
					len(witnessScript), MaxScriptSize)
				return scriptError(ErrScriptTooBig, str)
			}

			// Ensure that the serialized pkScript at the end of
			// the witness stack matches the witness program.
			witnessHash := sha3256(witnessScript)
			if !bytes.Equal(witnessHash, vm.witnessProgram) {
				return scriptError(ErrWitnessProgramMismatch,
					"witness program hash mismatch")
			}

			// With all the validity checks passed, parse the
			// script into individual op-codes so w can execute it
			// as the next script.
			pops, err := parseScript(witnessScript)
			if err != nil {
				return err
			}

			// The hash matched successfully, so use the witness as
			// the stack, and set the witnessScript to be the next
			// script executed.
			vm.scripts = append(vm.scripts, pops)
			vm.SetStack(witness[:len(witness)-1])

		default:
			errStr := fmt.Sprintf("length of witness program "+
				"must either be %v or %v bytes, instead is %v bytes",
				payToWitnessPubKeyHashDataSize,
				payToWitnessScriptHashDataSize,
				len(vm.witnessProgram))
			return scriptError(ErrWitnessProgramWrongLength, errStr)
		}
	} else if vm.hasFlag(ScriptVerifyDiscourageUpgradeableWitnessProgram) {
		errStr := fmt.Sprintf("new witness program versions "+
			"invalid: %v", vm.witnessProgram)
		return scriptError(ErrDiscourageUpgradableWitnessProgram, errStr)
	} else {
		errStr := fmt.Sprintf("invalid witness program version: %d", vm.witnessVersion)
		return scriptError(ErrInvalidWitness, errStr)
	}

	// All elements within the witness stack must not be greater
	// than the maximum bytes which are allowed to be pushed onto
	// the stack.
	for _, witElement := range vm.GetStack() {
		if len(witElement) > MaxScriptElementSize {
			str := fmt.Sprintf("element size %d exceeds "+
				"max allowed size %d", len(witElement),
				MaxScriptElementSize)
			return scriptError(ErrElementTooBig, str)
		}
	}

	return nil
}

// DisasmPC returns the string for the disassembly of the opcode that will be
// next to execute when Step() is called.
func (vm *Engine) DisasmPC() (string, error) {
	scriptIdx, scriptOff, err := vm.curPC()
	if err != nil {
		return "", err
	}
	return vm.disasm(scriptIdx, scriptOff), nil
}

// DisasmScript returns the disassembly string for the script at the requested
// offset index.  Index 0 is the signature script and 1 is the public key
// script.
func (vm *Engine) DisasmScript(idx int) (string, error) {
	if idx >= len(vm.scripts) {
		str := fmt.Sprintf("script index %d >= total scripts %d", idx,
			len(vm.scripts))
		return "", scriptError(ErrInvalidIndex, str)
	}

	var disstr string
	for i := range vm.scripts[idx] {
		disstr = disstr + vm.disasm(idx, i) + "\n"
	}
	return disstr, nil
}

// CheckErrorCondition returns nil if the running script has ended and was
// successful, leaving a a true boolean on the stack.  An error otherwise,
// including if the script has not finished.
func (vm *Engine) CheckErrorCondition(finalScript bool) error {
	// Check execution is actually done.  When pc is past the end of script
	// array there are no more scripts to run.
	if vm.scriptIdx < len(vm.scripts) {
		return scriptError(ErrScriptUnfinished,
			"error check when script unfinished")
	}

	// If we're in version zero witness execution mode, and this was the
	// final script, then the stack MUST be clean in order to maintain
	// compatibility with BIP16.
	if finalScript && vm.dstack.Depth() != 1 {
		return scriptError(ErrEvalFalse, "witness program must "+
			"have clean stack")
	}

	if vm.dstack.Depth() < 1 {
		return scriptError(ErrEmptyStack,
			"stack empty at end of script execution")
	}

	v, err := vm.dstack.PopBool()
	if err != nil {
		return err
	}
	if !v {
		// Log interesting data.
		log.Tracef("%v", newLogClosure(func() string {
			dis0, _ := vm.DisasmScript(0)
			dis1, _ := vm.DisasmScript(1)
			return fmt.Sprintf("scripts failed: script0: %s\n"+
				"script1: %s", dis0, dis1)
		}))
		return scriptError(ErrEvalFalse,
			"false stack entry at end of script execution")
	}
	return nil
}

// Step will execute the next instruction and move the program counter to the
// next opcode in the script, or the next script if the current has ended.  Step
// will return true in the case that the last opcode was successfully executed.
//
// The result of calling Step or any other method is undefined if an error is
// returned.
func (vm *Engine) Step() (done bool, err error) {
	// Verify that it is pointing to a valid script address.
	err = vm.validPC()
	if err != nil {
		return true, err
	}
	opcode := &vm.scripts[vm.scriptIdx][vm.scriptOff]
	vm.scriptOff++

	// Execute the opcode while taking into account several things such as
	// disabled opcodes, illegal opcodes, maximum allowed operations per
	// script, maximum script element sizes, and conditionals.
	err = vm.executeOpcode(opcode)
	if err != nil {
		return true, err
	}

	// The number of elements in the combination of the data and alt stacks
	// must not exceed the maximum number of stack elements allowed.
	combinedStackSize := vm.dstack.Depth() + vm.astack.Depth()
	if combinedStackSize > MaxStackSize {
		str := fmt.Sprintf("combined stack size %d > max allowed %d",
			combinedStackSize, MaxStackSize)
		return false, scriptError(ErrStackOverflow, str)
	}

	// Prepare for next instruction.
	if vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
		// Illegal to have an `if' that straddles two scripts.
		if err == nil && len(vm.condStack) != 0 {
			return false, scriptError(ErrUnbalancedConditional,
				"end of script reached in conditional execution")
		}

		// Alt stack doesn't persist.
		_ = vm.astack.DropN(vm.astack.Depth())

		vm.numOps = 0 // number of ops is per script.
		vm.scriptOff = 0
		vm.scriptIdx++

		// there are zero length scripts in the wild
		if vm.scriptIdx < len(vm.scripts) && vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
			vm.scriptIdx++
		}
		vm.lastCodeSep = 0
		if vm.scriptIdx >= len(vm.scripts) {
			return true, nil
		}
	}
	return false, nil
}

// Execute will execute all scripts in the script engine and return either nil
// for successful validation or an error if one occurred.
func (vm *Engine) Execute() (err error) {
	done := false
	for !done {
		log.Tracef("%v", newLogClosure(func() string {
			dis, err := vm.DisasmPC()
			if err != nil {
				return fmt.Sprintf("stepping (%v)", err)
			}
			return fmt.Sprintf("stepping %v", dis)
		}))

		done, err = vm.Step()
		if err != nil {
			return err
		}
		log.Tracef("%v", newLogClosure(func() string {
			var dstr, astr string

			// if we're tracing, dump the stacks.
			if vm.dstack.Depth() != 0 {
				dstr = "Stack:\n" + vm.dstack.String()
			}
			if vm.astack.Depth() != 0 {
				astr = "AltStack:\n" + vm.astack.String()
			}

			return dstr + astr
		}))
	}

	return vm.CheckErrorCondition(true)
}

// subScript returns the script since the last OP_CODESEPARATOR.
func (vm *Engine) subScript() []parsedOpcode {
	return vm.scripts[vm.scriptIdx][vm.lastCodeSep:]
}

// checkHashTypeEncoding returns whether or not the passed hashtype adheres to
// the strict encoding requirements if enabled.
func (vm *Engine) checkHashTypeEncoding(hashType SigHashType) error {
	sigHashType := hashType & ^SigHashAnyOneCanPay
	if sigHashType < SigHashAll || sigHashType > SigHashSingleReverse {
		str := fmt.Sprintf("invalid hash type 0x%x", hashType)
		return scriptError(ErrInvalidSigHashType, str)
	}
	return nil
}

// checkPubKeyEncoding returns whether or not the passed public key adheres to
// the strict encoding requirements if enabled.
func (vm *Engine) checkPubKeyEncoding(pubKey []byte) error {
	if !btcec.IsCompressedPubKey(pubKey) {
		str := "only compressed keys are accepted"
		return scriptError(ErrWitnessPubKeyType, str)
	}

	if len(pubKey) == 33 && (pubKey[0] == 0x02 || pubKey[0] == 0x03) {
		// Compressed
		return nil
	}
	if len(pubKey) == 65 && pubKey[0] == 0x04 {
		// Uncompressed
		return nil
	}

	return scriptError(ErrPubKeyType, "unsupported public key type")
}

// getStack returns the contents of stack as a byte array bottom up
func getStack(stack *stack) [][]byte {
	array := make([][]byte, stack.Depth())
	for i := range array {
		// PeekByteArry can't fail due to overflow, already checked
		array[len(array)-i-1], _ = stack.PeekByteArray(int32(i))
	}
	return array
}

// setStack sets the stack to the contents of the array where the last item in
// the array is the top item in the stack.
func setStack(stack *stack, data [][]byte) {
	// This can not error. Only errors are for invalid arguments.
	_ = stack.DropN(stack.Depth())

	for i := range data {
		stack.PushByteArray(data[i])
	}
}

// GetStack returns the contents of the primary stack as an array. where the
// last item in the array is the top of the stack.
func (vm *Engine) GetStack() [][]byte {
	return getStack(&vm.dstack)
}

// SetStack sets the contents of the primary stack to the contents of the
// provided array where the last item in the array will be the top of the stack.
func (vm *Engine) SetStack(data [][]byte) {
	setStack(&vm.dstack, data)
}

// GetAltStack returns the contents of the alternate stack as an array where the
// last item in the array is the top of the stack.
func (vm *Engine) GetAltStack() [][]byte {
	return getStack(&vm.astack)
}

// SetAltStack sets the contents of the alternate stack to the contents of the
// provided array where the last item in the array will be the top of the stack.
func (vm *Engine) SetAltStack(data [][]byte) {
	setStack(&vm.astack, data)
}

// NewEngine returns a new script engine for the provided public key script,
// transaction, and input index.  The flags modify the behavior of the script
// engine according to the description provided by each flag.
func NewEngine(
	tx *chain.Transaction,
	txIdx int,
	flags ScriptFlags,
	sigCache *SigCache,
	hashCache *TxSigHashes,
	address *chain.Address,
	inputAmount uint64,
) (*Engine, error) {
	// The provided transaction input index must refer to a valid input.
	if txIdx < 0 || txIdx >= len(tx.Inputs) {
		str := fmt.Sprintf("transaction input index %d is negative or "+
			">= %d", txIdx, len(tx.Inputs))
		return nil, scriptError(ErrInvalidIndex, str)
	}

	vm := Engine{flags: flags, sigCache: sigCache, hashCache: hashCache,
		inputAmount: inputAmount}

	vm.witnessVersion, vm.witnessProgram = int(address.Version), address.Hash

	if err := vm.verifyWitnessProgram(tx.Witnesses[txIdx].Items); err != nil {
		return nil, err
	}

	if vm.hasFlag(ScriptVerifyMinimalData) {
		vm.dstack.verifyMinimalData = true
		vm.astack.verifyMinimalData = true
	}

	vm.tx = tx
	vm.txIdx = txIdx
	return &vm, nil
}

func EngineStandardVerify(tx *chain.Transaction, txIdx int, address *chain.Address, amount uint64) error {
	engine, err := NewEngine(
		tx,
		txIdx,
		StandardVerifyFlags,
		NewSigCache(10),
		NewTxSigHashes(tx),
		address,
		amount,
	)
	if err != nil {
		return err
	}

	return engine.Execute()
}

func blake160(b []byte) []byte {
	h, _ := blake2b.New(20, nil)
	h.Write(b)
	return h.Sum(nil)
}

func sha3256(b []byte) []byte {
	out := sha3.Sum256(b)
	return out[:]
}
