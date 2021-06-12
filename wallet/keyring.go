package wallet

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/kurumiimari/gohan/chain"
	"github.com/pkg/errors"
	"sync"
)

var (
	ErrLocked          = errors.New("locked")
	ErrInvalidPassword = errors.New("invalid password")
)

type PrivateKeyer interface {
	PrivateKey(path ...uint32) (*btcec.PrivateKey, error)
}

type EKPrivateKeyer struct {
	ek chain.ExtendedKey
}

func NewEKPrivateKeyer(ek chain.ExtendedKey) PrivateKeyer {
	return &EKPrivateKeyer{ek: ek}
}

func (e EKPrivateKeyer) PrivateKey(path ...uint32) (*btcec.PrivateKey, error) {
	return chain.DeriveExtendedKey(e.ek, path...).PrivateKey()
}

type KeyLocker struct {
	box     SecretBox
	ek      chain.ExtendedKey
	mtx     sync.Mutex
	network *chain.Network
}

func NewKeyLocker(box SecretBox, network *chain.Network) *KeyLocker {
	return &KeyLocker{
		box:     box,
		network: network,
	}
}

func (k *KeyLocker) Unlock(password string) error {
	k.mtx.Lock()
	defer k.mtx.Unlock()
	priv, err := k.box.Decrypt(password)
	if err != nil {
		return ErrInvalidPassword
	}

	k.ek, err = chain.NewMasterExtendedKeyFromString(string(priv), k.network)
	if err != nil {
		panic(err)
	}
	return nil
}

func (k *KeyLocker) Lock() {
	k.mtx.Lock()
	defer k.mtx.Unlock()
	k.ek = nil
}

func (k *KeyLocker) Locked() bool {
	k.mtx.Lock()
	defer k.mtx.Unlock()
	return k.ek == nil
}

func (k *KeyLocker) PrivateKey(path ...uint32) (*btcec.PrivateKey, error) {
	k.mtx.Lock()
	defer k.mtx.Unlock()
	if k.ek == nil {
		return nil, ErrLocked
	}

	return chain.DeriveExtendedKey(k.ek, path...).PrivateKey()
}

type Keyring interface {
	PrivateKeyer
	IsPrivate() bool
	PublicEK(path ...uint32) chain.ExtendedKey
	Address(path ...uint32) *chain.Address
	PublicKey(path ...uint32) *btcec.PublicKey
	XPub(path ...uint32) string
}

type AccountKeyring struct {
	priv    PrivateKeyer
	pub     chain.ExtendedKey
	network *chain.Network
}

func NewAccountKeyring(
	priv PrivateKeyer,
	pub chain.ExtendedKey,
	network *chain.Network,
) *AccountKeyring {
	return &AccountKeyring{
		priv:    priv,
		pub:     pub,
		network: network,
	}
}

func (k *AccountKeyring) IsPrivate() bool {
	return k.priv != nil
}

func (k *AccountKeyring) PublicEK(path ...uint32) chain.ExtendedKey {
	return chain.DeriveExtendedKey(k.pub, path...)
}

func (k *AccountKeyring) PublicKey(path ...uint32) *btcec.PublicKey {
	return k.PublicEK(path...).PublicKey()
}

func (k *AccountKeyring) PrivateKey(path ...uint32) (*btcec.PrivateKey, error) {
	return k.priv.PrivateKey(path...)
}

func (k *AccountKeyring) XPub(path ...uint32) string {
	return k.PublicEK(path...).PublicString()
}

func (k *AccountKeyring) Address(path ...uint32) *chain.Address {
	return k.PublicEK(path...).Address()
}
