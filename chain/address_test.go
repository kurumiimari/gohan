package chain

import (
	"encoding/hex"
	"github.com/btcsuite/btcd/btcec"
	"github.com/stretchr/testify/require"
	"testing"
)

const addrHashHex = "6d5571fdbca1019cd0f0cd792d1b0bdfa7651c7e"
const addrBech32 = "hs1qd42hrldu5yqee58se4uj6xctm7nk28r70e84vx"
const addrPubkeyHex = "028d58b6d3c91a1871de5d6292cb435a416f99619b598eb296e9648de15a623fca"

func TestAddress_String(t *testing.T) {
	SetCurrNetwork(NetworkMain)
	hash, err := hex.DecodeString(addrHashHex)
	require.NoError(t, err)
	addr := NewAddressFromHash(hash)
	require.Equal(t, addrBech32, addr.String())
}

func TestNewAddressFromPubkey(t *testing.T) {
	SetCurrNetwork(NetworkMain)
	pubB, err := hex.DecodeString(addrPubkeyHex)
	require.NoError(t, err)
	pub, err := btcec.ParsePubKey(pubB, btcec.S256())
	require.NoError(t, err)
	addr := NewAddressFromPubkey(pub)
	require.NoError(t, err)
	require.Equal(t, "hs1qj74uc2cg3le8xslzq73fr78erz42xxnngxztc0", addr.String())
}

func TestNewAddressFromBech32(t *testing.T) {
	SetCurrNetwork(NetworkMain)
	addr, err := NewAddressFromBech32("hs1qj74uc2cg3le8xslzq73fr78erz42xxnngxztc0")
	require.NoError(t, err)
	require.Equal(t, "hs1qj74uc2cg3le8xslzq73fr78erz42xxnngxztc0", addr.String())
}
