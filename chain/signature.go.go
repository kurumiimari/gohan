package chain

import (
	"github.com/btcsuite/btcd/btcec"
	"math/big"
)

func SerializeRawSignature(sig *btcec.Signature) []byte {
	// handle low 'S' malleability
	// see btcec
	sigS := sig.S
	curve := btcec.S256()
	if sigS.Cmp(new(big.Int).Rsh(curve.N, 1)) == 1 {
		sigS = new(big.Int).Sub(curve.N, sigS)
	}

	rb := sig.R.Bytes()
	sb := sigS.Bytes()
	b := make([]byte, 64)
	copy(b[32-len(rb):], rb)
	copy(b[64-len(sb):], sb)
	return b
}
