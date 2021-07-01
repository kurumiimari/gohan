package wallet

import (
	"bytes"
	"github.com/bits-and-blooms/bloom/v3"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/txscript"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/pkg/errors"
	"os"
	"sync"
)

var (
	AddrLookahead = uint32(1000)
)

type BloomSaver func(dTx walletdb.Transactor, newFilter []byte) error

type AddressBloom struct {
	filter *bloom.BloomFilter
	mtx    sync.RWMutex
}

func NewAddressBloom() *AddressBloom {
	return &AddressBloom{
		filter: bloom.New(AddrBloomM, AddrBloomK),
	}
}

func NewAddressBloomFromBytes(b []byte) (*AddressBloom, error) {
	r := bytes.NewReader(b)
	filter := new(bloom.BloomFilter)
	if _, err := filter.ReadFrom(r); err != nil {
		return nil, errors.WithStack(err)
	}
	return &AddressBloom{
		filter: filter,
	}, nil
}

func (a *AddressBloom) Test(addr *chain.Address) bool {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.filter.Test(addr.Bytes())
}

func (a *AddressBloom) Update(addrs []*chain.Address) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	for _, addr := range addrs {
		a.filter.Add(addr.Bytes())
	}
}

func (a *AddressBloom) Bytes() []byte {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	buf := new(bytes.Buffer)
	if _, err := a.filter.WriteTo(buf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

type AddressManager struct {
	ring      Keyring
	bloom     *AddressBloom
	maker     AddressMaker
	accountID string
	branch    uint32
	currIdx   uint32
	lookIdx   int64
	lookSize  int64
	mtx       sync.RWMutex
}

type AddressManagerOption func(mgr *AddressManager)

type AddressMaker func(ring Keyring, branch, index uint32) *chain.Address

func P2PKHAddressMaker(ring Keyring, branch, index uint32) *chain.Address {
	return ring.Address(branch, index)
}

func HIP1AddressMaker(ring Keyring, branch, index uint32) *chain.Address {
	pub := ring.PublicKey(branch, index)
	script, err := txscript.NewHIP1LockingScript(pub.SerializeCompressed())
	if err != nil {
		panic(err)
	}
	return chain.NewAddressFromScript(script)
}

func NewAddressManager(
	ring Keyring,
	bloom *AddressBloom,
	accountID string,
	branch,
	currIdx uint32,
	lookIdx int64,
	opts ...AddressManagerOption,
) *AddressManager {
	mgr := &AddressManager{
		ring:      ring,
		bloom:     bloom,
		accountID: accountID,
		maker:     P2PKHAddressMaker,
		branch:    branch,
		currIdx:   currIdx,
		lookIdx:   lookIdx,
		lookSize:  int64(AddrLookahead),
	}
	for _, opt := range opts {
		opt(mgr)
	}
	return mgr
}

func WithLookSize(size uint32) AddressManagerOption {
	return func(mgr *AddressManager) {
		mgr.lookSize = int64(size)
	}
}

func WithAddressMaker(maker AddressMaker) AddressManagerOption {
	return func(mgr *AddressManager) {
		mgr.maker = maker
	}
}

func (a *AddressManager) Depth() uint32 {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	a.ensureInitialized()
	return a.currIdx
}

func (a *AddressManager) Lookahead() uint32 {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	a.ensureInitialized()
	return uint32(a.lookIdx)
}

func (a *AddressManager) Address() *chain.Address {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	a.ensureInitialized()
	return a.maker(a.ring, a.branch, a.currIdx)
}

func (a *AddressManager) NextAddress(dTx walletdb.Transactor) (*chain.Address, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.ensureInitialized()

	next := a.currIdx + 1
	addr := a.maker(a.ring, a.branch, next)
	if err := a.setAddressIdx(dTx, next); err != nil {
		return nil, err
	}
	a.currIdx = next
	return addr, nil
}

func (a *AddressManager) SetAddressIdx(dTx walletdb.Transactor, newIdx uint32) error {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.setAddressIdx(dTx, newIdx)
}

func (a *AddressManager) setAddressIdx(dTx walletdb.Transactor, newIdx uint32) error {
	if newIdx > a.currIdx {
		if err := walletdb.UpdateAddressIdx(dTx, a.accountID, a.branch, newIdx); err != nil {
			return err
		}
		a.currIdx = newIdx + 1
	}

	newLookTip := int64(newIdx) + a.lookSize

	if a.lookIdx > newLookTip {
		return nil
	}

	var addrs []*chain.Address
	for i := a.lookIdx + 1; i <= newLookTip; i++ {
		addr := a.maker(a.ring, a.branch, uint32(i))
		addrs = append(addrs, addr)
		if _, err := walletdb.CreateAddress(dTx, a.accountID, addr, a.branch, uint32(i)); err != nil {
			return err
		}
	}
	a.bloom.Update(addrs)
	if err := walletdb.UpdateAddressBloom(dTx, a.accountID, a.bloom.Bytes()); err != nil {
		return err
	}

	a.lookIdx = newLookTip
	return nil
}

func (a *AddressManager) ensureInitialized() {
	if a.lookIdx < 0 {
		panic("address manager is not initialized")
	}
}

func init() {
	if os.Getenv("IS_TEST") == "1" {
		AddrLookahead = 10
	}
}
