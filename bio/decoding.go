package bio

import (
	"encoding/binary"
	"encoding/hex"
	"io"
)

type GuardReader struct {
	r   io.Reader
	N   int64
	Err error
}

func NewGuardReader(r io.Reader) *GuardReader {
	return &GuardReader{
		r: r,
	}
}

func (g *GuardReader) Read(b []byte) (int, error) {
	if g.Err != nil {
		return 0, g.Err
	}

	n, err := g.r.Read(b)
	g.N += int64(n)
	if err != nil {
		g.Err = err
	}
	return n, err
}

func ReadByte(r io.Reader) (byte, error) {
	b, err := ReadFixedBytes(r, 1)
	if err != nil {
		return 0, err
	}
	return b[0], err
}

func ReadFixedBytes(r io.Reader, byteLen int) ([]byte, error) {
	b := make([]byte, byteLen)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, err
	}
	return b, nil
}

func ReadVarBytes(r io.Reader) ([]byte, error) {
	l, err := ReadVarint(r)
	if err != nil {
		return nil, err
	}
	return ReadFixedBytes(r, int(l))
}

func ReadVarint(r io.Reader) (uint64, error) {
	sigil, err := ReadByte(r)
	if err != nil {
		return 0, nil
	}
	if sigil < 0xfd {
		return uint64(sigil), nil
	}
	if sigil == 0xfd {
		num := make([]byte, 2)
		if _, err := io.ReadFull(r, num); err != nil {
			return 0, err
		}
		return uint64(binary.LittleEndian.Uint16(num)), nil
	}
	if sigil == 0xfe {
		num := make([]byte, 4)
		if _, err := io.ReadFull(r, num); err != nil {
			return 0, err
		}
		return uint64(binary.LittleEndian.Uint32(num)), nil
	}
	num := make([]byte, 8)
	if _, err := io.ReadFull(r, num); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(num), nil
}

func ReadUint32LE(r io.Reader) (uint32, error) {
	b, err := ReadFixedBytes(r, 4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func ReadUint64LE(r io.Reader) (uint64, error) {
	b, err := ReadFixedBytes(r, 8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

func ReadUint16BE(r io.Reader) (uint16, error) {
	b, err := ReadFixedBytes(r, 2)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

func DecodeHexArray(in []string) ([][]byte, error) {
	items := make([][]byte, len(in))
	for i, item := range in {
		b, err := hex.DecodeString(item)
		if err != nil {
			return nil, err
		}
		items[i] = b
	}
	return items, nil
}

func MustDecodeHexArray(in []string) [][]byte {
	out, err := DecodeHexArray(in)
	if err != nil {
		panic(err)
	}
	return out
}
