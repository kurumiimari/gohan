package chain

import (
	"github.com/kurumiimari/gohan/bio"
	"github.com/pkg/errors"
	"io"
)

type Script struct {
	raw []byte
}

func (c *Script) PushOp(op Opcode) *Script {
	c.raw = append(c.raw, byte(op))
	return c
}

func (c *Script) PushData(data []byte) *Script {
	c.raw = append(c.raw, byte(NewDataOpcode(len(data))))
	c.raw = append(c.raw, data...)
	return c
}

func (c *Script) WriteTo(w io.Writer) (int64, error) {
	g := bio.NewGuardWriter(w)
	bio.WriteByte(g, byte(len(c.raw)))
	bio.WriteRawBytes(g, c.raw)
	return g.N, errors.Wrap(g.Err, "error writing script")
}

func NewP2PKHScript(pkh []byte) *Script {
	code := new(Script).
		PushOp(OP_DUP).
		PushOp(OP_BLAKE160).
		PushData(pkh).
		PushOp(OP_EQUALVERIFY).
		PushOp(OP_CHECKSIG)

	return code
}
