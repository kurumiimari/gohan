package chain

import "github.com/tyler-smith/go-bip39"

func GenerateRandomSeed(password string) ([]byte, string) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		panic(err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		panic(err)
	}

	return bip39.NewSeed(mnemonic, password), mnemonic
}
