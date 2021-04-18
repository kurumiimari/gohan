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

func (k *KeyLocker) IsLocked() bool {
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

type PrivateKeyer interface {
	PrivateKey(path ...uint32) (*btcec.PrivateKey, error)
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
	index   uint32
}

func NewAccountKeyring(
	priv PrivateKeyer,
	pub chain.ExtendedKey,
	network *chain.Network,
	index uint32,
) *AccountKeyring {
	return &AccountKeyring{
		priv:    priv,
		pub:     pub,
		network: network,
		index:   index,
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
	return k.priv.PrivateKey(k.joinPaths(path)...)
}

func (k *AccountKeyring) XPub(path ...uint32) string {
	return k.PublicEK(path...).PublicString()
}

func (k *AccountKeyring) Address(path ...uint32) *chain.Address {
	return k.PublicEK(path...).Address()
}

func (k *AccountKeyring) joinPaths(path []uint32) []uint32 {
	var out []uint32
	out = append(out, []uint32{
		chain.HardenNode(chain.CoinPurpose),
		chain.HardenNode(k.network.KeyPrefix.CoinType),
		chain.HardenNode(k.index),
	}...)
	out = append(out, path...)
	return out
}
