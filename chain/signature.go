package chain

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip32"
)

type DerivationSigner interface {
	Sign(deriv Derivation, b []byte) ([]byte, *bip32.Key, error)
}

func SignWithBip32Key(key *bip32.Key, b []byte) ([]byte, error) {
	if !key.IsPrivate {
		return nil, errors.New("cannot sign with a public key")
	}
	pk, _ := btcec.PrivKeyFromBytes(btcec.S256(), key.Key[:32])
	sig, err := pk.Sign(b)
	if err != nil {
		return nil, err
	}
	return SerializeRawSignature(sig), nil
}

