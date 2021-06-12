package gcrypto

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
	"io"
	"reflect"
)

type Hash []byte

func (h Hash) IsZero() bool {
	for _, b := range h {
		if b != 0x00 {
			return false
		}
	}
	return true
}

func (h Hash) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(h)
	return int64(n), err
}

func (h Hash) String() string {
	return hex.EncodeToString(h)
}

func (h Hash) MarshalJSON() ([]byte, error) {
	if len(h) == 0 {
		return json.Marshal(nil)
	}
	return json.Marshal(h.String())
}

func (h *Hash) UnmarshalJSON(b []byte) error {
	var hexStr string
	if err := json.Unmarshal(b, &hexStr); err != nil {
		return errors.WithStack(err)
	}
	buf, err := hex.DecodeString(hexStr)
	if err != nil {
		return errors.WithStack(err)
	}
	*h = buf
	return nil
}

func (h Hash) Value() (driver.Value, error) {
	if len(h) == 0 {
		return nil, nil
	}
	return hex.EncodeToString(h), nil
}

func (h *Hash) Scan(src interface{}) error {
	switch t := src.(type) {
	case nil:
		*h = nil
	case string:
		buf, err := hex.DecodeString(t)
		if err != nil {
			return errors.WithStack(err)
		}
		*h = buf
	default:
		return errors.Errorf("cannot scan %v into hash", reflect.TypeOf(src))
	}

	return nil
}

func (h Hash) Equal(other Hash) bool {
	return bytes.Equal(h, other)
}

func Blake160(in []byte) Hash {
	buf, _ := blake2b.New(20, nil)
	buf.Write(in)
	return buf.Sum(nil)
}

func Blake256(in []byte) Hash {
	buf := blake2b.Sum256(in)
	return buf[:]
}

func SHA3256(in []byte) Hash {
	buf := sha3.Sum256(in)
	return buf[:]
}
