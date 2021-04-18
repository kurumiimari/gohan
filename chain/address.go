package chain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/bech32"
	"github.com/kurumiimari/gohan/bio"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"io"
)

type Address struct {
	Version uint8
	Hash    []byte
}

func NewAddress(version uint8, hash []byte) *Address {
	return &Address{
		Version: version,
		Hash:    hash,
	}
}

func NewAddressFromHash(hash []byte) *Address {
	return &Address{
		Version: 0,
		Hash:    hash,
	}
}

func NewAddressFromPubkey(key *btcec.PublicKey) *Address {
	h, _ := blake2b.New(20, nil)
	h.Write(key.SerializeCompressed())
	hash := h.Sum(nil)
	return NewAddressFromHash(hash)
}

func NewAddressFromBech32(bech string) (*Address, error) {
	_, data, err := bech32.Decode(bech)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding bech32")
	}
	version := data[0]
	hash, err := bech32.ConvertBits(data[1:], 5, 8, false)
	if err != nil {
		return nil, errors.Wrap(err, "error converting bits")
	}
	return &Address{
		Version: version,
		Hash:    hash,
	}, nil
}

func MustAddressFromBech32(bech string) *Address {
	addr, err := NewAddressFromBech32(bech)
	if err != nil {
		panic(err)
	}
	return addr
}

func (a *Address) Size() int {
	return 2 + len(a.Hash)
}

func (a *Address) IsPubkeyHash() bool {
	return a.Version == 0 && len(a.Hash) == 20
}

func (a *Address) IsScriptHash() bool {
	return a.Version == 0 && len(a.Hash) == 32
}

func (a *Address) String(network *Network) string {
	data, err := bech32.ConvertBits(a.Hash, 8, 5, true)
	if err != nil {
		panic(err)
	}
	bech, err := bech32.Encode(network.AddressHRP, append([]byte{a.Version}, data...))
	if err != nil {
		panic(err)
	}
	return bech
}

func (a *Address) Equal(b *Address) bool {
	return a.Version == b.Version && bytes.Equal(a.Hash, b.Hash)
}

func (a *Address) WriteTo(w io.Writer) (int64, error) {
	g := bio.NewGuardWriter(w)
	bio.WriteByte(g, a.Version)
	bio.WriteByte(g, uint8(len(a.Hash)))
	bio.WriteRawBytes(g, a.Hash)
	return g.N, errors.Wrap(g.Err, "error writing address")
}

func (a *Address) ReadFrom(r io.Reader) (int64, error) {
	g := bio.NewGuardReader(r)
	version, _ := bio.ReadByte(g)
	hashLen, _ := bio.ReadByte(g)
	hash, _ := bio.ReadFixedBytes(r, int(hashLen))
	if g.Err != nil {
		return g.N, errors.Wrap(g.Err, "error reading address")
	}
	a.Version = version
	a.Hash = hash
	return g.N, nil
}

func (a *Address) MarshalJSON() ([]byte, error) {
	addrJson := struct {
		Version uint8  `json:"version"`
		Hash    string `json:"hash"`
	}{
		Version: a.Version,
		Hash:    hex.EncodeToString(a.Hash),
	}
	return json.Marshal(addrJson)
}
