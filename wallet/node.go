package wallet

import (
	"encoding/json"
	"fmt"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/client"
	"github.com/kurumiimari/gohan/shakedex"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"
	"gopkg.in/tomb.v2"
	"runtime"
	"sync"
)

type Node struct {
	tmb      *tomb.Tomb
	network  *chain.Network
	engine   *walletdb.Engine
	client   *client.NodeRPCClient
	bm       *BlockMonitor
	accounts map[string]*Account
	wMtx     sync.Mutex
}

type NodeStatus struct {
	Status   string `json:"status"`
	Height   int    `json:"height"`
	MemUsage uint64 `json:"mem_usage"`
	Version  string `json:"version"`
}

func NewNode(
	tmb *tomb.Tomb,
	network *chain.Network,
	engine *walletdb.Engine,
	client *client.NodeRPCClient,
	bm *BlockMonitor,
) *Node {
	return &Node{
		tmb:      tmb,
		network:  network,
		engine:   engine,
		client:   client,
		bm:       bm,
		accounts: make(map[string]*Account),
	}
}

func (s *Node) Status() *NodeStatus {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &NodeStatus{
		Status:   "OK",
		Height:   s.bm.LastHeight(),
		MemUsage: memStats.HeapAlloc,
	}
}

func (s *Node) PollBlock() error {
	return s.bm.Poll()
}

func (s *Node) Start() error {
	var accounts []*walletdb.AccountOpts
	err := s.engine.Transaction(func(tx walletdb.Transactor) error {
		a, err := walletdb.GetAllAccounts(tx)
		if err != nil {
			return err
		}
		accounts = a
		return nil
	})
	if err != nil {
		return errors.WithStack(err)
	}

	s.wMtx.Lock()
	for _, acc := range accounts {
		s.accounts[acc.ID], err = NewAccount(
			s.tmb,
			s.network,
			s.engine,
			s.client,
			s.bm,
			acc,
		)
		if err != nil {
			return err
		}
	}
	s.wMtx.Unlock()

	for name, a := range s.accounts {
		if err := a.Start(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("account %s failed to start", name))
		}
	}
	return nil
}

func (s *Node) ImportMnemonic(id, password, mnemonic string, index uint32) (*Account, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("invalid mnemonic")
	}

	ek := chain.NewMasterExtendedKeyFromMnemonic(mnemonic, "", s.network)
	wallet, err := s.create(id, password, ek, index)
	if err != nil {
		return nil, errors.Wrap(err, "error creating wallet")
	}
	return wallet, nil
}

func (s *Node) ImportXPub(id, password, xPubStr string, index uint32) (*Account, error) {
	ek, err := chain.NewMasterExtendedKeyFromXPub(xPubStr, s.network)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing xpub")
	}

	wallet, err := s.create(id, password, ek, index)
	if err != nil {
		return nil, errors.Wrap(err, "error creating wallet")
	}
	return wallet, nil
}

func (s *Node) CreateWallet(name, password string, index uint32) (*Account, string, error) {
	seed, mnemonic := chain.GenerateRandomSeed("")
	ek := chain.NewMasterExtendedKey(seed, s.network)
	wallet, err := s.create(name, password, ek, index)
	if err != nil {
		return nil, "", errors.Wrap(err, "error creating wallet")
	}
	return wallet, mnemonic, nil
}

func (s *Node) Account(id string) (*Account, error) {
	s.wMtx.Lock()
	defer s.wMtx.Unlock()
	account := s.accounts[id]
	if account == nil {
		return nil, errors.New("account not found")
	}
	return account, nil
}

func (s *Node) Accounts() []string {
	s.wMtx.Lock()
	defer s.wMtx.Unlock()
	var accounts []string
	for id := range s.accounts {
		accounts = append(accounts, id)
	}
	return accounts
}

func (s *Node) create(id, password string, ek chain.ExtendedKey, index uint32) (*Account, error) {
	s.wMtx.Lock()
	defer s.wMtx.Unlock()

	if err := ValidateAccountID(id); err != nil {
		return nil, errors.Wrap(err, "invalid account ID")
	}

	var err error
	var accountKey chain.ExtendedKey
	if ek.IsPrivate() {
		accountKey = chain.DeriveExtendedKey(
			ek,
			chain.HardenNode(chain.CoinPurpose),
			chain.HardenNode(s.network.KeyPrefix.CoinType),
			chain.HardenNode(index),
		)
	} else {
		accountKey = ek
	}

	bloom := NewAddressBloom()
	ring := NewAccountKeyring(nil, accountKey, s.network)
	dec, err := EncryptDefault([]byte(accountKey.PrivateString()), password)
	if err != nil {
		panic(err)
	}
	seed, err := json.Marshal(dec)
	if err != nil {
		panic(err)
	}

	opts := &walletdb.AccountOpts{
		ID:            id,
		Seed:          string(seed),
		WatchOnly:     !ek.IsPrivate(),
		Idx:           index,
		XPub:          accountKey.Neuter(),
		OutpointBloom: NewOutpointBloomFromOutpoints(nil).Bytes(),
	}

	err = s.engine.Transaction(func(tx walletdb.Transactor) error {
		for i := uint32(0); i <= AddrLookahead; i++ {
			recv := ring.Address(chain.ReceiveBranch, i)
			change := ring.Address(chain.ChangeBranch, i)
			if _, err := walletdb.CreateAddress(tx, opts.ID, recv, chain.ReceiveBranch, i); err != nil {
				return err
			}
			if _, err := walletdb.CreateAddress(tx, opts.ID, change, chain.ChangeBranch, i); err != nil {
				return err
			}
			bloom.Update([]*chain.Address{
				recv,
				change,
			})
		}

		for i := uint32(0); i <= 10; i++ {
			dutch := HIP1AddressMaker(ring, shakedex.AddressBranch, i)
			if _, err := walletdb.CreateAddress(tx, opts.ID, dutch, shakedex.AddressBranch, i); err != nil {
				return err
			}
			bloom.Update([]*chain.Address{dutch})
		}

		opts.LookaheadTips, err = walletdb.GetLookaheadTips(tx, opts.ID)
		if err != nil {
			return err
		}
		opts.AddressBloom = bloom.Bytes()
		return walletdb.CreateAccount(
			tx,
			opts,
		)
	})
	if err != nil {
		return nil, errors.Wrap(err, "error creating account")
	}

	acc, err := NewAccount(
		s.tmb,
		s.network,
		s.engine,
		s.client,
		s.bm,
		opts,
	)
	if err != nil {
		return nil, err
	}
	if err := acc.Start(); err != nil {
		return nil, errors.Wrap(err, "error opening wallet")
	}
	s.accounts[id] = acc
	return acc, nil
}
