package wallet

import (
	"encoding/json"
	"fmt"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/client"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"
	"gopkg.in/tomb.v2"
	"runtime"
	"sync"
)

type Node struct {
	tmb     *tomb.Tomb
	network *chain.Network
	engine  *walletdb.Engine
	client  *client.NodeRPCClient
	bm      *BlockMonitor
	wallets map[string]*Wallet
	wMtx    sync.Mutex
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
		tmb:     tmb,
		network: network,
		engine:  engine,
		client:  client,
		bm:      bm,
		wallets: make(map[string]*Wallet),
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
	var wallets []*walletdb.Wallet
	err := s.engine.Transaction(func(tx walletdb.Transactor) error {
		w, err := walletdb.GetWallets(tx)
		if err != nil {
			return err
		}
		wallets = w
		return nil
	})
	if err != nil {
		return errors.WithStack(err)
	}

	s.wMtx.Lock()
	for _, w := range wallets {
		dec, err := UnmarshalSecretBox([]byte(w.Seed))
		if err != nil {
			panic(err)
		}

		s.wallets[w.ID] = NewWallet(
			s.tmb,
			s.network,
			s.engine,
			s.client,
			s.bm,
			NewKeyLocker(dec, s.network),
			w.ID,
		)
	}
	s.wMtx.Unlock()

	for name, w := range s.wallets {
		if err := w.Start(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("wallet %s failed to start", name))
		}
	}
	return nil
}

func (s *Node) ImportMnemonic(id, password, mnemonic string) (*walletdb.Wallet, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("invalid mnemonic")
	}

	ek := chain.NewMasterExtendedKeyFromMnemonic(mnemonic, "", s.network)
	wallet, err := s.create(id, password, ek)
	if err != nil {
		return nil, errors.Wrap(err, "error creating wallet")
	}
	return wallet, nil
}

func (s *Node) ImportXPub(id, password, xPubStr string) (*walletdb.Wallet, error) {
	ek, err := chain.NewMasterExtendedKeyFromXPub(xPubStr, s.network)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing xpub")
	}

	wallet, err := s.create(id, password, ek)
	if err != nil {
		return nil, errors.Wrap(err, "error creating wallet")
	}
	return wallet, nil
}

func (s *Node) CreateWallet(name, password string) (*walletdb.Wallet, string, error) {
	seed, mnemonic := chain.GenerateRandomSeed("")
	ek := chain.NewMasterExtendedKey(seed, s.network)
	wallet, err := s.create(name, password, ek)
	if err != nil {
		return nil, "", errors.Wrap(err, "error creating wallet")
	}
	return wallet, mnemonic, nil
}

func (s *Node) Wallet(name string) (*Wallet, error) {
	s.wMtx.Lock()
	defer s.wMtx.Unlock()
	wallet := s.wallets[name]
	if wallet == nil {
		return nil, errors.New("wallet not found")
	}
	return wallet, nil
}

func (s *Node) Wallets() []string {
	s.wMtx.Lock()
	defer s.wMtx.Unlock()
	var wallets []string
	for wName := range s.wallets {
		wallets = append(wallets, wName)
	}
	return wallets
}

func (s *Node) create(id, password string, ek chain.ExtendedKey) (*walletdb.Wallet, error) {
	s.wMtx.Lock()
	defer s.wMtx.Unlock()

	var err error
	var intBranch chain.ExtendedKey
	var extBranch chain.ExtendedKey
	var accountKey chain.ExtendedKey
	if ek.IsPrivate() {
		accountKey = chain.DeriveExtendedKey(
			ek,
			chain.HardenNode(chain.CoinPurpose),
			chain.HardenNode(s.network.KeyPrefix.CoinType),
			chain.HardenNode(0),
		)
		intBranch = chain.DeriveExtendedKey(
			accountKey,
			1,
		)
		extBranch = chain.DeriveExtendedKey(
			accountKey,
			0,
		)
	} else {
		accountKey = ek
		intBranch = chain.DeriveExtendedKey(
			ek,
			1,
		)
		extBranch = chain.DeriveExtendedKey(
			ek,
			0,
		)
	}

	addrPool := NewAddrBloomFromAddrs(nil)
	var recvAddrs []*chain.Address
	var changeAddrs []*chain.Address
	for i := 0; i < int(AddrLookahead); i++ {
		recvAddr := extBranch.Child(uint32(i)).Address()
		changeAddr := intBranch.Child(uint32(i)).Address()
		addrPool.Add(recvAddr)
		recvAddrs = append(recvAddrs, recvAddr)
		addrPool.Add(changeAddr)
		changeAddrs = append(changeAddrs, changeAddr)
	}

	dec, err := EncryptDefault([]byte(ek.PrivateString()), password)
	if err != nil {
		panic(err)
	}
	seed, err := json.Marshal(dec)
	if err != nil {
		panic(err)
	}

	var wallet *walletdb.Wallet
	err = s.engine.Transaction(func(tx walletdb.Transactor) error {
		wallet, err = walletdb.CreateWallet(
			tx,
			&walletdb.Wallet{
				ID:        id,
				Seed:      string(seed),
				WatchOnly: !ek.IsPrivate(),
			},
		)
		if err != nil {
			return errors.Wrap(err, "error saving wallet")
		}
		_, err = walletdb.CreateAccount(
			tx,
			"default",
			id,
			accountKey.PublicString(),
			addrPool.Bytes(),
			NewOutpointBloomFromOutpoints(nil).Bytes(),
		)
		if err != nil {
			return errors.Wrap(err, "error creating initial account")
		}
		accID := fmt.Sprintf("%s/default", id)
		for i := 0; i < len(recvAddrs); i++ {
			recv := recvAddrs[i].String(s.network)
			change := changeAddrs[i].String(s.network)
			if _, err := walletdb.CreateAddress(tx, recv, accID, 0, i); err != nil {
				return errors.Wrap(err, "error creating address")
			}
			if _, err := walletdb.CreateAddress(tx, change, accID, 1, i); err != nil {
				return errors.Wrap(err, "error creating address")
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "error creating wallet")
	}

	w := NewWallet(
		s.tmb,
		s.network,
		s.engine,
		s.client,
		s.bm,
		NewKeyLocker(dec, s.network),
		wallet.ID,
	)
	if err := w.Start(); err != nil {
		return nil, errors.Wrap(err, "error opening wallet")
	}
	s.wallets[wallet.ID] = w
	return wallet, nil
}
