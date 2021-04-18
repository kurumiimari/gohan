package chain

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/tyler-smith/go-bip39"
)

type ExtendedKey interface {
	IsPrivate() bool
	Child(i uint32) ExtendedKey
	Address() *Address
	PublicKey() *btcec.PublicKey
	PrivateKey() (*btcec.PrivateKey, error)
	PublicString() string
	PrivateString() string
	Neuter() ExtendedKey
}

type MasterExtendedKey struct {
	ek      *hdkeychain.ExtendedKey
	network *Network
}

func NewMasterExtendedKey(seed []byte, network *Network) *MasterExtendedKey {
	ek, err := hdkeychain.NewMaster(seed, network.ChainParams())

	// HSD crashes on key derivation errors,
	// so we'll do the same thing.
	if err != nil {
		panic(err)
	}

	return &MasterExtendedKey{
		ek:      ek,
		network: network,
	}
}

func NewMasterExtendedKeyFromString(priv string, network *Network) (*MasterExtendedKey, error) {
	ek, err := hdkeychain.NewKeyFromString(priv)
	if err != nil {
		return nil, err
	}

	return &MasterExtendedKey{
		ek:      ek,
		network: network,
	}, nil
}

func NewMasterExtendedKeyFromMnemonic(mnemonic string, password string, network *Network) *MasterExtendedKey {
	seed := bip39.NewSeed(mnemonic, password)
	return NewMasterExtendedKey(seed, network)
}

func NewMasterExtendedKeyFromXPub(xPub string, network *Network) (*MasterExtendedKey, error) {
	ek, err := hdkeychain.NewKeyFromString(xPub)
	if err != nil {
		return nil, err
	}

	return &MasterExtendedKey{
		ek:      ek,
		network: network,
	}, nil
}

func (m *MasterExtendedKey) IsPrivate() bool {
	return m.ek.IsPrivate()
}

func (m *MasterExtendedKey) Child(i uint32) ExtendedKey {
	ek, err := m.ek.Child(i)
	if err != nil {
		panic(err)
	}

	return &MasterExtendedKey{
		ek:      ek,
		network: m.network,
	}
}

func (m *MasterExtendedKey) Address() *Address {
	return NewAddressFromPubkey(m.PublicKey())
}

func (m *MasterExtendedKey) PublicKey() *btcec.PublicKey {
	pub, err := m.ek.ECPubKey()
	if err != nil {
		panic(err)
	}
	return pub
}

func (m *MasterExtendedKey) PrivateKey() (*btcec.PrivateKey, error) {
	return m.ek.ECPrivKey()
}

func (m *MasterExtendedKey) PublicString() string {
	pub, err := m.ek.Neuter()
	if err != nil {
		panic(err)
	}
	return pub.String()
}

func (m *MasterExtendedKey) PrivateString() string {
	return m.ek.String()
}

func (m *MasterExtendedKey) Neuter() ExtendedKey {
	ek, err := m.ek.Neuter()
	if err != nil {
		panic(err)
	}
	return &MasterExtendedKey{
		ek:      ek,
		network: m.network,
	}
}
