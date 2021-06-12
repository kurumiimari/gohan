package chain

import (
	"github.com/kurumiimari/gohan/bio"
	"golang.org/x/crypto/blake2b"
)

func CreateBlind(ek ExtendedKey, name string, address *Address, value uint64) []byte {
	return BlindFromNonce(value, GenerateNonce(ek, name, address, value))
}

func BlindFromNonce(value uint64, nonce []byte) []byte {
	h, _ := blake2b.New256(nil)
	bio.WriteUint64LE(h, value)
	bio.WriteRawBytes(h, nonce)
	return h.Sum(nil)
}

func GenerateNonce(ek ExtendedKey, name string, address *Address, value uint64) []byte {
	hi := value * (1 / 0x100000000) >> 0
	lo := value >> 0
	index := uint32((hi ^ lo) & 0x7fffffff)
	key := DeriveExtendedKey(ek, index).PublicKey()

	h, _ := blake2b.New256(nil)
	address.Hash.WriteTo(h)
	h.Write(key.SerializeCompressed())
	h.Write(HashName(name))

	return h.Sum(nil)
}
