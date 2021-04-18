package wallet

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/pkg/errors"
	"github.com/kurumiimari/gohan/walletdb"
	"os"
	"sync"
)

type addrUpdaterFunc func(walletdb.Transactor, string, int) error

var (
	AddrLookahead = uint32(1000)

	addrUpdaters = [2]addrUpdaterFunc{
		walletdb.UpdateAccountRecvIdx,
		walletdb.UpdateAccountChangeIdx,
	}
)

type AddrManager struct {
	network     *chain.Network
	ring        Keyring
	bloom       *AddressBloom
	accountID   string
	addrIndices [2]uint32
	lookTips    [2]uint32
	mtx         sync.RWMutex
}

func NewAddrManager(network *chain.Network, ring Keyring, bloom *AddressBloom, accountID string, addrIndices, lookTips [2]uint32) *AddrManager {
	return &AddrManager{
		network:     network,
		ring:        ring,
		bloom:       bloom,
		accountID:   accountID,
		addrIndices: addrIndices,
		lookTips:    lookTips,
	}
}

func (a *AddrManager) HasAddress(addr *chain.Address) bool {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.bloom.Test(addr)
}

func (a *AddrManager) ChangeLookahead() uint32 {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.lookTips[chain.ChangeBranch]
}

func (a *AddrManager) RecvLookahead() uint32 {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.lookTips[chain.ReceiveBranch]
}

func (a *AddrManager) ChangeDepth() uint32 {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.addrIndices[chain.ChangeBranch]
}

func (a *AddrManager) RecvDepth() uint32 {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.addrIndices[chain.ReceiveBranch]
}

func (a *AddrManager) ChangeAddress() *chain.Address {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.ring.Address(chain.ChangeBranch, a.addrIndices[chain.ChangeBranch])
}

func (a *AddrManager) RecvAddress() *chain.Address {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.ring.Address(chain.ReceiveBranch, a.addrIndices[chain.ReceiveBranch])
}

func (a *AddrManager) GenChangeAddress(q walletdb.Transactor) (*chain.Address, error) {
	return a.GenAddress(q, chain.ChangeBranch)
}

func (a *AddrManager) GenRecvAddress(q walletdb.Transactor) (*chain.Address, error) {
	return a.GenAddress(q, chain.ReceiveBranch)
}

func (a *AddrManager) GenAddress(q walletdb.Transactor, branch uint32) (*chain.Address, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	nextIdx := a.addrIndices[branch] + 1
	address := a.ring.Address(branch, nextIdx)
	updater := addrUpdaters[branch]
	if err := updater(q, a.accountID, int(nextIdx)); err != nil {
		return nil, errors.Wrap(err, "error updating account change index")
	}
	if err := a.incLookahead(q, branch, nextIdx); err != nil {
		return nil, errors.Wrap(err, "error incrementing lookahead")
	}

	a.addrIndices[branch] = nextIdx
	return address, nil
}

func (a *AddrManager) IncLookahead(q walletdb.Transactor, branch uint32, newIdx uint32) error {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.incLookahead(q, branch, newIdx)
}

func (a *AddrManager) incLookahead(q walletdb.Transactor, branch uint32, nextIdx uint32) error {
	if nextIdx+AddrLookahead-1 < a.lookTips[branch] {
		return nil
	}

	bloomCopy := a.bloom.Copy()
	for i := a.lookTips[branch] + 1; i < nextIdx+AddrLookahead; i++ {
		next := a.ring.Address(branch, i)
		_, err := walletdb.CreateAddress(
			q,
			next.String(a.network),
			a.accountID,
			int(branch),
			int(i),
		)
		if err != nil {
			return errors.Wrap(err, "error writing new lookahead address")
		}
		bloomCopy.Add(next)
	}

	if err := walletdb.UpdateAddressBloom(q, a.accountID, bloomCopy.Bytes()); err != nil {
		return errors.Wrap(err, "error updating bloom filter")
	}

	if a.addrIndices[branch] < nextIdx {
		var updater func(walletdb.Transactor, string, int) error
		if branch == chain.ReceiveBranch {
			updater = walletdb.UpdateAccountRecvIdx
		} else {
			updater = walletdb.UpdateAccountChangeIdx
		}
		if err := updater(q, a.accountID, int(nextIdx)); err != nil {
			return errors.Wrap(err, "error updating index")
		}

		a.addrIndices[branch] = nextIdx
	}

	a.bloom = bloomCopy
	a.lookTips[branch] = nextIdx + AddrLookahead - 1
	return nil
}

func init() {
	if os.Getenv("IS_TEST") == "1" {
		AddrLookahead = 10
	}
}
