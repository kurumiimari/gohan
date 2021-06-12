package chain

import "math"

type SigOpt uint8

type VerifyFlag uint64

const (
	DefaultSequence = math.MaxUint32

	ReceiveBranch = uint32(0)
	ChangeBranch  = uint32(1)

	SignMessageMagic = "handshake signed message:\n"
)

var (
	ZeroHash = make([]byte, 32, 32)
)
