package bio

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type GuardWriter struct {
	w   io.Writer
	N   int64
	Err error
}

func NewGuardWriter(w io.Writer) *GuardWriter {
	return &GuardWriter{
		w: w,
	}
}

func (g *GuardWriter) Write(b []byte) (int, error) {
	if g.Err != nil {
		return 0, g.Err
	}

	n, err := g.w.Write(b)
	g.N += int64(n)
	if err != nil {
		g.Err = err
	}
	return n, err
}

func WriteByte(w io.Writer, b byte) (int, error) {
	return w.Write([]byte{b})
}

func WriteRawBytes(w io.Writer, b []byte) (int, error) {
	return w.Write(b)
}

func WriteFixedBytes(w io.Writer, b []byte, bLen int) (int, error) {
	if len(b) != bLen {
		panic(fmt.Sprintf("buffer len is %d but should be %d", len(b), bLen))
	}
	return WriteRawBytes(w, b)
}

func WriteVarBytes(w io.Writer, b []byte) (int, error) {
	var total int
	n, err := WriteVarint(w, uint64(len(b)))
	total += n
	if err != nil {
		return total, err
	}
	n, err = WriteRawBytes(w, b)
	total += n
	return total, err
}

func WriteVarint(w io.Writer, n uint64) (int, error) {
	var buf []byte
	if n <= 0xfc {
		buf = []byte{uint8(n)}
	} else if n <= math.MaxUint16 {
		buf = make([]byte, 3)
		buf[0] = 0xfd
		binary.LittleEndian.PutUint16(buf[1:], uint16(n))
	} else if n <= math.MaxUint32 {
		buf = make([]byte, 5)
		buf[0] = 0xfe
		binary.LittleEndian.PutUint32(buf[1:], uint32(n))
	} else {
		buf = make([]byte, 9)
		buf[0] = 0xff
		binary.LittleEndian.PutUint64(buf[1:], n)
	}
	return WriteRawBytes(w, buf)
}

func WriteUint16BE(w io.Writer, n uint16) (int, error) {
	return WriteRawBytes(w, Uint16BE(n))
}

func WriteUint32LE(w io.Writer, n uint32) (int, error) {
	return WriteRawBytes(w, Uint32LE(n))
}

func WriteUint64LE(w io.Writer, n uint64) (int, error) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, n)
	return WriteRawBytes(w, b)
}

func Uint16BE(n uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, n)
	return b
}

func Uint32LE(n uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	return b
}
