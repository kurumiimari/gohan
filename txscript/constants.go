package txscript

// ScriptFlags is a bitmask defining additional operations or tests that will be
// done when executing a script pair.
type ScriptFlags uint32

const (
	ScriptVerifyNone ScriptFlags = 1 << iota

	// ScriptVerifyMinimalData defines that signatures must use the smallest
	// push operator. This is both rules 3 and 4 of BIP0062.
	ScriptVerifyMinimalData

	// ScriptDiscourageUpgradableNops defines whether to verify that
	// NOP1 through NOP10 are reserved for future soft-fork upgrades.  This
	// flag must not be used for consensus critical code nor applied to
	// blocks as this flag is only for stricter standard transaction
	// checks.  This flag is only applied when the above opcodes are
	// executed.
	ScriptDiscourageUpgradableNops

	// ScriptVerifyDiscourageUpgradeableWitnessProgram makes witness
	// program with versions 2-16 non-standard.
	ScriptVerifyDiscourageUpgradeableWitnessProgram

	// ScriptVerifyMinimalIf makes a script with an OP_IF/OP_NOTIF whose
	// operand is anything other than empty vector or [0x01] non-standard.
	ScriptVerifyMinimalIf

	// ScriptVerifyNullFail defines that signatures must be empty if
	// a CHECKSIG or CHECKMULTISIG operation fails.
	ScriptVerifyNullFail
)

const (
	// StandardVerifyFlags are the script flags which are used when
	// executing transaction scripts to enforce additional checks which
	// are required for the script to be considered standard.
	StandardVerifyFlags = 0 |
		ScriptVerifyMinimalData |
		ScriptVerifyMinimalIf |
		ScriptVerifyNullFail |
		ScriptDiscourageUpgradableNops |
		ScriptVerifyDiscourageUpgradeableWitnessProgram
)

const (
	// MaxStackSize is the maximum combined height of stack and alt stack
	// during execution.
	MaxStackSize = 1000

	// MaxScriptSize is the maximum allowed length of a raw script.
	MaxScriptSize = 10000

	// payToWitnessPubKeyHashDataSize is the size of the witness program's
	// data push for a pay-to-witness-pub-key-hash output.
	payToWitnessPubKeyHashDataSize = 20

	// payToWitnessScriptHashDataSize is the size of the witness program's
	// data push for a pay-to-witness-script-hash output.
	payToWitnessScriptHashDataSize = 32
)

// These are the constants specified for maximums in individual scripts.
const (
	MaxOpsPerScript       = 201 // Max number of non-push operations.
	MaxPubKeysPerMultiSig = 20  // Multisig can't have more sigs than this.
	MaxScriptElementSize  = 520 // Max bytes pushable to the stack.
)

const (
	// LockTimeThreshold is the number below which a lock time is
	// interpreted to be a block number.  Since an average of one block
	// is generated per 10 minutes, this allows blocks for about 9,512
	// years.
	LockTimeThreshold = 5e8 // Tue Nov 5 00:53:20 1985 UTC
)
