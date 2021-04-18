package wallet

import "crypto/rand"

func RandBytes(bLen int) []byte {
	out := make([]byte, bLen)
	if _, err := rand.Read(out); err != nil {
		panic(err)
	}
	return out
}
