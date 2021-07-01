package wallet

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	xPub = "xpub6BudTY3ybzN3FC6cLJiYprizHUWMC1UZyK4f7mB6r6DYJPnMT1tRgUtcWzcjBmCrQbJHb1JpbERigfL1a7X2JJsBZVbxXV5PgoDfoUzp43Y"
)

//func TestInterop(t *testing.T) {
//	bloom := bloom.New(1678, 23)
//	bloom.Add([]byte("test string"))
//	bloom.Add([]byte{0x01, 0x02, 0x03})
//	bloom.Add(make([]byte, 32, 32))
//	buf := new(bytes.Buffer)
//	bloom.WriteTo(buf)
//	fmt.Println(hex.EncodeToString(buf.Bytes()))
//
//	m := murmur3.New128()
//	m.Write([]byte("test string"))
//	a, b := m.Sum128()
//	m.Write([]byte{1})
//	c, d := m.Sum128()
//	bb := make([]byte, 16)
//	binary.BigEndian.PutUint64(bb, a)
//	binary.BigEndian.PutUint64(bb[8:], b)
//	bi := new(big.Int)
//	bi.SetBytes(bb)
//	fmt.Println(bi)
//	binary.BigEndian.PutUint64(bb, c)
//	binary.BigEndian.PutUint64(bb[8:], d)
//	bi.SetBytes(bb)
//	fmt.Println(bi)
//}

//func TestEstimateParameters(t *testing.T) {
//	m, k := bloom.EstimateParameters(100000, 0.000001)
//	fmt.Println(m)
//	fmt.Println(k)
//}

//func TestAddrBloom(t *testing.T) {
//	ek, err := chain.NewMasterExtendedKeyFromXPub(xPub, chain.NetworkMain)
//	require.NoError(t, err)
//
//	pool := NewAddrBloomFromAddrs(nil)
//
//	for i := 0; i < 10; i++ {
//		key := ek.Child(uint32(i))
//		addr := chain.NewAddressFromPubkey(key.PublicKey())
//		pool.Add(addr)
//	}
//
//	for i := 0; i < 10; i++ {
//		key := ek.Child(uint32(i))
//		addr := chain.NewAddressFromPubkey(key.PublicKey())
//		require.True(t, pool.Test(addr))
//	}
//
//	key := ek.Child(99).PublicKey()
//	require.False(t, pool.Test(chain.NewAddressFromPubkey(key)))
//}
//
//func TestAddrBloom_Marshaling(t *testing.T) {
//	ek, err := chain.NewMasterExtendedKeyFromXPub(xPub, chain.NetworkMain)
//	require.NoError(t, err)
//
//	pool1 := NewAddrBloomFromAddrs(nil)
//	for i := 0; i < 10; i++ {
//		key := ek.Child(uint32(i))
//		addr := chain.NewAddressFromPubkey(key.PublicKey())
//		pool1.Add(addr)
//	}
//
//	pool2, err := AddressBloomFromBytes(pool1.Bytes())
//	require.NoError(t, err)
//
//	for i := 0; i < 10; i++ {
//		key := ek.Child(uint32(i))
//		addr := chain.NewAddressFromPubkey(key.PublicKey())
//		require.True(t, pool2.Test(addr))
//	}
//}

func TestOutpointBloom(t *testing.T) {
	pool := NewOutpointBloomFromOutpoints(nil)

	var ops []*chain.Outpoint

	for i := 0; i < 10; i++ {
		op := &chain.Outpoint{
			Hash:  RandBytes(32),
			Index: uint32(i),
		}
		require.False(t, pool.Test(op))
		pool.Add(op)
		ops = append(ops, op)
	}

	for _, op := range ops {
		require.True(t, pool.Test(op))
	}
}
