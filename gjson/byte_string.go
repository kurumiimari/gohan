package gjson

import (
	"encoding/hex"
	"encoding/json"
	"github.com/pkg/errors"
)

type ByteString []byte

func (b ByteString) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(b))
}

func (b *ByteString) UnmarshalJSON(buf []byte) error {
	var h string
	if err := json.Unmarshal(buf, &h); err != nil {
		return errors.WithStack(err)
	}
	bs, err := hex.DecodeString(h)
	if err != nil {
		return errors.WithStack(err)
	}
	*b = bs
	return nil
}
