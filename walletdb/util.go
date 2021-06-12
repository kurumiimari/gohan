package walletdb

import (
	"encoding/hex"
	"github.com/kurumiimari/gohan/chain"
)

func scanOutpoint(hashHex string, idx uint32) *chain.Outpoint {
	if hashHex == "" {
		return nil
	}

	hash, err := hex.DecodeString(hashHex)
	if err != nil {
		panic(err)
	}
	return &chain.Outpoint{
		Hash:  hash,
		Index: idx,
	}
}
