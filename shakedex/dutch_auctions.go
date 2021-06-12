package shakedex

import "github.com/kurumiimari/gohan/chain"

const (
	V1DutchAuctionMagic = "SHAKEDEX_PROOF:1.0.0"
)

func NewDutchAuctionScript(pk []byte) *chain.Script {
	return new(chain.Script).
		PushOp(chain.OP_TYPE).
		PushOp(chain.OP_9).
		PushOp(chain.OP_EQUAL).
		PushOp(chain.OP_IF).
		PushData(pk).
		PushOp(chain.OP_CHECKSIG).
		PushOp(chain.OP_ELSE).
		PushOp(chain.OP_TYPE).
		PushOp(chain.OP_10).
		PushOp(chain.OP_EQUAL).
		PushOp(chain.OP_ENDIF)
}

type DutchAuction struct {
	Name             string
	LockingTxHash    string
	LockingOutputIdx int
	PublicKey        []byte
	PaymentAddr      *chain.Address
	Data             []byte
	FeeAddr          *chain.Address
}

type DutchAuctionPresign struct {
	Price     uint64
	LockTime  int
	Fee       uint64
	Signature []byte
}

