package bio

import "encoding/hex"

func MustDecodeHex(in string) []byte {
	out, err := hex.DecodeString(in)
	if err != nil {
		panic(err)
	}
	return out
}
