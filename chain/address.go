package chain

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/bech32"
	"github.com/kurumiimari/gohan/bio"
	"github.com/kurumiimari/gohan/gcrypto"
	"github.com/pkg/errors"
	"io"
)

type Address struct {
	Version uint8
	Hash    gcrypto.Hash
}

func NewAddress(version uint8, hash gcrypto.Hash) *Address {
	return &Address{
		Version: version,
		Hash:    hash,
	}
}

func NewAddressFromHash(hash gcrypto.Hash) *Address {
	return &Address{
		Version: 0,
		Hash:    hash,
	}
}

func NewAddressFromPubkey(key *btcec.PublicKey) *Address {
	return NewAddressFromHash(gcrypto.Blake160(key.SerializeCompressed()))
}

func NewAddressFromScript(script []byte) *Address {
	return NewAddressFromHash(gcrypto.SHA3256(script))
}

func NewAddressFromBech32(bech string) (*Address, error) {
	_, data, err := bech32.Decode(bech)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	version := data[0]
	converted, err := bech32.ConvertBits(data[1:], 5, 8, false)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(converted) != 20 && len(converted) != 32 {
		return nil, errors.New("invalid hash length")
	}

	return &Address{
		Version: version,
		Hash:    converted,
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

func (a *Address) String() string {
	data, err := bech32.ConvertBits(a.Hash, 8, 5, true)
	if err != nil {
		panic(err)
	}
	bech, err := bech32.Encode(GetCurrNetwork().AddressHRP, append([]byte{a.Version}, data...))
	if err != nil {
		panic(err)
	}
	return bech
}

func (a *Address) Equal(b *Address) bool {
	return a.Version == b.Version && a.Hash.Equal(b.Hash)
}

func (a *Address) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(a.Bytes())
	return int64(n), err
}

func (a *Address) ReadFrom(r io.Reader) (int64, error) {
	g := bio.NewGuardReader(r)
	version, _ := bio.ReadByte(g)
	hashLen, _ := bio.ReadByte(g)
	hashData, _ := bio.ReadFixedBytes(r, int(hashLen))
	if g.Err != nil {
		return g.N, errors.Wrap(g.Err, "error reading address")
	}
	hash := gcrypto.Hash(hashData)
	a.Version = version
	a.Hash = hash
	return g.N, nil
}

func (a *Address) Bytes() []byte {
	buf := make([]byte, 2+len(a.Hash))
	buf[0] = a.Version
	buf[1] = uint8(len(a.Hash))
	copy(buf[2:], a.Hash)
	return buf
}

func (a *Address) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}

	return a.String(), nil
}

func (a *Address) Scan(src interface{}) error {
	switch s := src.(type) {
	case nil:
		return nil
	case string:
		addr, err := NewAddressFromBech32(s)
		if err != nil {
			return err
		}
		*a = *addr
		return nil
	default:
		return errors.New("cannot scan an address from types other than string")
	}
}

func (a *Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *Address) UnmarshalJSON(b []byte) error {
	var bech string
	if err := json.Unmarshal(b, &bech); err != nil {
		return errors.WithStack(err)
	}
	addr, err := NewAddressFromBech32(bech)
	if err != nil {
		return errors.WithStack(err)
	}
	*a = *addr
	return nil
}
