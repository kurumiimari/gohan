package chain

import "math"

type SigOpt uint8

const (
	SighashAll           SigOpt = 1
	SighashNone          SigOpt = 2
	SighashSingle        SigOpt = 3
	SighashSingleReverse SigOpt = 4
	SighashNoInput       SigOpt = 0x40
	SighashAnyoneCanPay  SigOpt = 0x80

	DefaultSequence = math.MaxUint32

	ReceiveBranch = uint32(0)
	ChangeBranch  = uint32(1)

	SignMessageMagic = "handshake signed message:\n"
)


var (
	ZeroHash = make([]byte, 32, 32)
)