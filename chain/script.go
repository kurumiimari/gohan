package chain

import (
	"github.com/kurumiimari/gohan/bio"
	"github.com/pkg/errors"
	"io"
)

var (
	ErrScriptOversize = errors.New("script over max script size")
	ErrBadOpcode      = errors.New("bad opcode")
	ErrMinimalData    = errors.New("minimal data")
)

type Script struct {
	raw  []byte
	size int
}

func (c *Script) PushOp(op Opcode) *Script {
	c.raw = append(c.raw, byte(op))
	c.size += 1
	return c
}

func (c *Script) PushData(data []byte) *Script {
	c.raw = append(c.raw, byte(NewDataOpcode(len(data))))
	c.raw = append(c.raw, data...)
	c.size += len(data) + 1
	return c
}

func (c *Script) WriteTo(w io.Writer) (int64, error) {
	g := bio.NewGuardWriter(w)
	bio.WriteByte(g, byte(len(c.raw)))
	bio.WriteRawBytes(g, c.raw)
	return g.N, errors.Wrap(g.Err, "error writing script")
}

func (c *Script) Size() int {
	return c.size
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

type ScriptStack struct {
	Items [][]byte
}

func (s *ScriptStack) PushInt(i uint8) {
	s.Items = append(s.Items, []byte{i})
}

func ExecScript(script *Script, flags VerifyFlag, tx *Transaction, index int, value uint64) error {
	return nil
}
