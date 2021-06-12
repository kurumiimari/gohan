package txscript

import (
	"bytes"
	"github.com/kurumiimari/gohan/chain"
)

// NewP2PKHScript creates a new script to pay a transaction
// output to a 20-byte pubkey hash. It is expected that the input
// is a valid hash.
func NewP2PKHScript(pubKeyHash []byte) ([]byte, error) {
	return NewScriptBuilder().
		AddOp(OP_DUP).
		AddOp(OP_BLAKE160).
		AddData(pubKeyHash).
		AddOp(OP_EQUALVERIFY).
		AddOp(OP_CHECKSIG).
		Script()
}

func NewHIP1LockingScript(pubKey []byte) ([]byte, error) {
	return NewScriptBuilder().
		AddOp(OP_TYPE).
		AddInt64(int64(chain.CovenantTransfer)).
		AddOp(OP_EQUAL).
		AddOp(OP_IF).
		AddData(pubKey).
		AddOp(OP_CHECKSIG).
		AddOp(OP_ELSE).
		AddOp(OP_TYPE).
		AddInt64(int64(chain.CovenantFinalize)).
		AddOp(OP_EQUAL).
		AddOp(OP_ENDIF).
		Script()
}

func IsHIP1LockingScript(witness *chain.Witness) bool {
	script := witness.Items[len(witness.Items)-1]
	if len(script) != 44 {
		return false
	}
	pub := script[5:38]
	genScript, err := NewHIP1LockingScript(pub)
	return err == nil && bytes.Equal(script, genScript)
}
