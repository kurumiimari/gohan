package wallet

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/kurumiimari/gohan/bio"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/client"
	"github.com/kurumiimari/gohan/log"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip32"
	"golang.org/x/crypto/blake2b"
	"gopkg.in/tomb.v2"
	"strings"
	"sync"
	"time"
)

const (
	BlockFetchConcurrency = 5
)

var accLogger = log.ModuleLogger("account")

type UnspentBid struct {
	Name            string `json:"name"`
	BlockHeight     int    `json:"block_height"`
	Lockup          uint64 `json:"lockup"`
	BidValue        uint64 `json:"bid_value"`
	TxHash          string `json:"tx_hash"`
	OutIdx          int    `json:"out_idx"`
	Revealable      bool   `json:"revealable"`
	RevealableBlock int    `json:"revealable_block"`
}

type UnspentReveal struct {
	Name            string `json:"name"`
	BlockHeight     int    `json:"block_height"`
	Value           uint64 `json:"value"`
	TxHash          string `json:"tx_hash"`
	OutIdx          int    `json:"out_idx"`
	Redeemable      bool   `json:"redeemable"`
	RedeemableBlock int    `json:"redeemable_block"`
}

type Account struct {
	tmb           *tomb.Tomb
	network       *chain.Network
	engine        *walletdb.Engine
	client        *client.NodeRPCClient
	bm            *BlockMonitor
	ring          Keyring
	addrManager   *AddrManager
	id            string
	name          string
	idx           uint32
	rescanHeight  int
	xPub          *bip32.Key
	outpointBloom *OutpointBloom
	mtx           sync.RWMutex
	lgr           log.Logger
}

func NewAccount(
	tmb *tomb.Tomb,
	network *chain.Network,
	engine *walletdb.Engine,
	client *client.NodeRPCClient,
	bm *BlockMonitor,
	ring Keyring,
	dbAcc *walletdb.Account,
) *Account {
	return &Account{
		tmb:     tmb,
		network: network,
		engine:  engine,
		client:  client,
		bm:      bm,
		ring:    ring,
		addrManager: NewAddrManager(
			network,
			ring,
			MustAddressBloomFromBytes(dbAcc.AddressBloom),
			dbAcc.ID,
			[2]uint32{dbAcc.ReceivingIdx, dbAcc.ChangeIdx},
			dbAcc.LookaheadTips,
		),
		id:            dbAcc.ID,
		name:          dbAcc.Name,
		idx:           dbAcc.Idx,
		rescanHeight:  dbAcc.RescanHeight,
		outpointBloom: MustOutpointBloomFromBytes(dbAcc.OutpointBloom),
		lgr:           accLogger.Child("id", fmt.Sprintf("%s/%s", dbAcc.WalletID, dbAcc.Name)),
	}
}

func (a *Account) Start() error {
	a.tmb.Go(func() error {
		blockC := a.bm.Subscribe()

		if err := a.bm.poll(); err != nil {
			a.lgr.Error("error on initial poll", "err", err)
		}

		for {
			select {
			case <-a.tmb.Dying():
				return nil
			case notif := <-blockC:
				if err := a.lockedRescan(notif); err != nil {
					a.lgr.Error("error indexing block", "err", err)
				}
			}
		}
	})

	return nil
}

func (a *Account) Name() string {
	return a.name
}

func (a *Account) Index() uint32 {
	return a.idx
}

func (a *Account) Locked() bool {
	_, err := a.ring.PrivateKey(0, 0)
	return errors.Is(err, ErrLocked)
}

func (a *Account) RescanHeight() int {
	return a.rescanHeight
}

func (a *Account) Balances() (*walletdb.Balances, error) {
	var balances *walletdb.Balances
	err := a.engine.Transaction(func(tx walletdb.Transactor) error {
		bals, err := walletdb.GetBalances(tx, a.id, a.network, a.rescanHeight)
		balances = bals
		return err
	})
	return balances, err
}

func (a *Account) AddressDepth() (uint32, uint32) {
	return a.addrManager.RecvDepth(), a.addrManager.ChangeDepth()
}

func (a *Account) LookaheadDepth() (uint32, uint32) {
	return a.addrManager.RecvLookahead(), a.addrManager.ChangeLookahead()
}

func (a *Account) ReceiveAddress() *chain.Address {
	return a.addrManager.RecvAddress()
}

func (a *Account) ChangeAddress() *chain.Address {
	return a.addrManager.ChangeAddress()
}

func (a *Account) XPub() string {
	return a.ring.XPub()
}

func (a *Account) GenerateReceiveAddress() (*chain.Address, uint32, error) {
	var addr *chain.Address
	err := a.engine.Transaction(func(tx walletdb.Transactor) error {
		genAddr, err := a.addrManager.GenRecvAddress(tx)
		if err != nil {
			return err
		}

		addr = genAddr
		return nil
	})
	return addr, a.addrManager.RecvDepth(), err
}

func (a *Account) GenerateChangeAddress() (*chain.Address, uint32, error) {
	var addr *chain.Address
	err := a.engine.Transaction(func(tx walletdb.Transactor) error {
		genAddr, err := a.addrManager.GenChangeAddress(tx)
		if err != nil {
			return err
		}

		addr = genAddr
		return nil
	})
	return addr, a.addrManager.ChangeDepth(), err
}

func (a *Account) Coins() ([]*walletdb.Coin, error) {
	var coins []*walletdb.Coin
	err := a.engine.Transaction(func(q walletdb.Transactor) error {
		c, err := walletdb.GetUnspentCoins(q, a.id)
		if err != nil {
			return err
		}
		coins = c
		return nil
	})
	return coins, err
}

func (a *Account) Names(count, offset int) ([]*walletdb.Name, error) {
	var names []*walletdb.Name
	err := a.engine.Transaction(func(tx walletdb.Transactor) error {
		n, err := walletdb.GetNames(tx, a.id, count, offset)
		if err != nil {
			return err
		}
		names = n
		return nil
	})
	return names, err
}

func (a *Account) History(name string, count, offset int) ([]*walletdb.RichNameHistoryEntry, error) {
	var history []*walletdb.RichNameHistoryEntry
	err := a.engine.Transaction(func(tx walletdb.Transactor) error {
		hist, err := walletdb.GetNameHistory(tx, a.network, a.id, name, count, offset)
		if err != nil {
			return err
		}
		history = hist
		return nil
	})
	return history, err
}

func (a *Account) Transactions(count int, offset int) ([]*walletdb.RichTransaction, error) {
	var txs []*walletdb.RichTransaction
	err := a.engine.Transaction(func(q walletdb.Transactor) error {
		t, err := walletdb.ListTransactions(q, a.id, a.network, count, offset)
		if err != nil {
			return err
		}
		txs = t
		return nil
	})
	return txs, err
}

func (a *Account) Send(value uint64, feeRate uint64, address *chain.Address) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	return a.txTransactor(func(dTx walletdb.Transactor) (*chain.Transaction, error) {
		return a.send(dTx, address, value, feeRate)
	})
}

func (a *Account) Open(name string, feeRate uint64) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	if !chain.IsNameValid(name) {
		return nil, errors.New("invalid name")
	}

	if chain.IsNameReserved(a.network, a.rescanHeight, name) {
		return nil, errors.New("name is reserved")
	}

	if !chain.HasRollout(a.network, a.rescanHeight, name) {
		return nil, errors.New("name not rolled out yet")
	}

	state, err := a.client.GetNameInfo(name)
	if err != nil {
		return nil, err
	}
	if state.Info != nil {
		return nil, errors.New("name is not openable")
	}

	return a.txTransactor(func(dTx walletdb.Transactor) (*chain.Transaction, error) {
		// TODO: double open

		recvAddr := a.addrManager.RecvAddress()

		txb := new(TxBuilder)
		txb.AddOutput(&chain.Output{
			Value:   0,
			Address: recvAddr,
			Covenant: &chain.Covenant{
				Type: chain.CovenantOpen,
				Items: [][]byte{
					chain.HashName(name),
					bio.Uint32LE(0),
					[]byte(name),
				},
			},
		})

		tx, err := a.fundTx(dTx, txb, feeRate)
		if err != nil {
			return nil, err
		}
		if err := a.sendTx(dTx, tx); err != nil {
			return nil, err
		}
		return tx, nil
	})
}

func (a *Account) Bid(name string, feeRate, value, lockup uint64) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	if !chain.IsNameValid(name) {
		return nil, errors.New("invalid name")
	}

	if chain.IsNameReserved(a.network, a.rescanHeight, name) {
		return nil, errors.New("name is reserved")
	}

	if !chain.HasRollout(a.network, a.rescanHeight, name) {
		return nil, errors.New("name not rolled out yet")
	}

	if value > lockup {
		return nil, errors.New("value exceeds lockup")
	}

	state, err := a.requireNameState(name, "BIDDING")
	if err != nil {
		return nil, err
	}

	recvAddr := a.addrManager.RecvAddress()
	blind := chain.CreateBlind(a.ring.PublicEK(), name, recvAddr, value)
	txb := new(TxBuilder)
	txb.AddOutput(&chain.Output{
		Value:   lockup,
		Address: recvAddr,
		Covenant: &chain.Covenant{
			Type: chain.CovenantBid,
			Items: [][]byte{
				chain.HashName(name),
				bio.Uint32LE(uint32(state.Info.Height)),
				[]byte(name),
				blind,
			},
		},
	})

	var tx *chain.Transaction
	return a.txTransactor(func(dTx walletdb.Transactor) (*chain.Transaction, error) {
		tx, err = a.fundTx(dTx, txb, feeRate)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		entry := &walletdb.NameHistory{
			AccountID: a.id,
			Name:      name,
			Type:      walletdb.NameActionBid,
			TxHash:    tx.IDHex(),
			OutIdx:    0,
			Value:     lockup,
			BidValue:  value,
		}
		if err := walletdb.UpdateNameHistory(dTx, entry); err != nil {
			return nil, err
		}
		if err := a.sendTx(dTx, tx); err != nil {
			return nil, err
		}
		return tx, nil
	})
}

func (a *Account) Reveal(name string, feeRate uint64) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	if !chain.IsNameValid(name) {
		return nil, errors.New("invalid name")
	}

	state, err := a.requireNameState(name, "REVEAL")
	if err != nil {
		return nil, err
	}

	return a.txTransactor(func(dTx walletdb.Transactor) (*chain.Transaction, error) {
		bids, err := walletdb.GetRevealableBids(dTx, a.id, name, a.network, state.Info.Stats.RevealPeriodStart)
		if err != nil {
			return nil, err
		}
		if len(bids) == 0 {
			return nil, errors.New("no bids to reveal")
		}

		return a.sendReveals(dTx, bids, name, state.Info.Height, feeRate)
	})
}

func (a *Account) Redeem(name string, feeRate uint64) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	if !chain.IsNameValid(name) {
		return nil, errors.New("invalid name")
	}

	state, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	winner := state.Info.Owner

	return a.txTransactor(func(dTx walletdb.Transactor) (*chain.Transaction, error) {
		localReveals, err := walletdb.GetRedeemableReveals(dTx, a.id, name, a.network, state.Info.Stats.RenewalPeriodStart)
		if err != nil {
			return nil, err
		}
		if len(localReveals) == 0 {
			return nil, errors.New("no reveals to redeem")
		}

		losingReveals := make([]*walletdb.RedeemableReveal, 0)
		for _, rev := range localReveals {
			if rev.TxHash == winner.Hash && rev.OutIdx == winner.Index {
				continue
			}
			losingReveals = append(losingReveals, rev)
		}

		if len(losingReveals) == 0 {
			return nil, errors.New("no losing reveals")
		}

		return a.sendRedeems(dTx, losingReveals, name, state.Info.Height, feeRate)
	})
}

func (a *Account) Update(name string, resource *chain.Resource, feeRate uint64) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	var tx *chain.Transaction
	var err error
	err = a.engine.Transaction(func(q walletdb.Transactor) error {
		tx, err = a.sendUpdate(q, name, resource, feeRate)
		return err
	})
	return tx, err
}

func (a *Account) Transfer(name string, address *chain.Address, feeRate uint64) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	state, err := a.client.GetNameInfo(name)
	if err != nil {
		return nil, errors.Wrap(err, "error getting id info")
	}
	if state.Info == nil {
		return nil, errors.New("name must be opened")
	}
	if state.Info.State != "CLOSED" {
		return nil, errors.New("auction is not closed")
	}

	var tx *chain.Transaction
	err = a.engine.Transaction(func(q walletdb.Transactor) error {
		owner := state.Info.Owner
		ownerCoin, err := walletdb.GetCoinByOutpoint(q, a.id, owner.Hash, owner.Index)
		if err == sql.ErrNoRows {
			return errors.New("name is not owned by this wallet")
		}
		if err != nil {
			return errors.Wrap(err, "error getting transfer coin")
		}

		coin := ConvertDBCoin(ownerCoin)
		cov := coin.Covenant
		if cov.Type != chain.CovenantRegister &&
			cov.Type != chain.CovenantUpdate &&
			cov.Type != chain.CovenantRenew &&
			cov.Type != chain.CovenantFinalize {
			return errors.New("name must be registered")
		}

		txb := new(TxBuilder)
		txb.AddCoin(coin)
		txb.AddOutput(&chain.Output{
			Value:   coin.Value,
			Address: coin.Address,
			Covenant: &chain.Covenant{
				Type: chain.CovenantTransfer,
				Items: [][]byte{
					chain.HashName(name),
					bio.Uint32LE(uint32(state.Info.Height)),
					{address.Version},
					address.Hash,
				},
			},
		})

		tx, err := a.fundTx(q, txb, feeRate)
		if err != nil {
			return errors.Wrap(err, "error funding transaction")
		}

		return a.sendTx(q, tx)
	})
	return tx, err
}

func (a *Account) Finalize(name string, feeRate uint64) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	state, err := a.client.GetNameInfo(name)
	if err != nil {
		return nil, errors.Wrap(err, "error getting id info")
	}
	if state.Info == nil {
		return nil, errors.New("name must be opened")
	}
	if state.Info.State != "CLOSED" {
		return nil, errors.New("auction is not closed")
	}

	var tx *chain.Transaction
	err = a.engine.Transaction(func(q walletdb.Transactor) error {
		owner := state.Info.Owner
		ownerCoin, err := walletdb.GetCoinByOutpoint(q, a.id, owner.Hash, owner.Index)
		if err == sql.ErrNoRows {
			return errors.New("name is not owned by this wallet")
		}
		if err != nil {
			return errors.Wrap(err, "error getting transfer coin")
		}

		coin := ConvertDBCoin(ownerCoin)
		if coin.Covenant.Type != chain.CovenantTransfer {
			return errors.New("name is not being transferred")
		}

		if a.rescanHeight < coin.Height+a.network.TransferLockup {
			return errors.New("transfer is still locked up")
		}

		txb := new(TxBuilder)
		addr := chain.NewAddress(
			coin.Covenant.Items[2][0],
			coin.Covenant.Items[3],
		)

		var flags uint8
		if state.Info.Weak {
			flags |= 1
		}

		renewalBlockRaw, err := a.client.GetRenewalBlock(a.network, a.rescanHeight)
		if err != nil {
			return errors.Wrap(err, "error getting renewal block")
		}

		renewalBlock := new(chain.Block)
		if _, err := renewalBlock.ReadFrom(bytes.NewReader(renewalBlockRaw)); err != nil {
			panic(err)
		}

		txb.AddCoin(coin)
		txb.AddOutput(&chain.Output{
			Value:   coin.Value,
			Address: addr,
			Covenant: &chain.Covenant{
				Type: chain.CovenantFinalize,
				Items: [][]byte{
					chain.HashName(name),
					bio.Uint32LE(uint32(state.Info.Height)),
					[]byte(name),
					{flags},
					bio.Uint32LE(uint32(state.Info.Claimed)),
					bio.Uint32LE(uint32(state.Info.Renewals)),
					renewalBlock.Hash(),
				},
			},
		})

		tx, err := a.fundTx(q, txb, feeRate)
		if err != nil {
			return errors.Wrap(err, "error funding transaction")
		}

		return a.sendTx(q, tx)
	})
	return tx, err
}

func (a *Account) Renew(name string, feeRate uint64) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	return a.txTransactor(func(q walletdb.Transactor) (*chain.Transaction, error) {
		return a.sendRenewal(q, name, feeRate)
	})
}

func (a *Account) Revoke(name string, feeRate uint64) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	state, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	return a.txTransactor(func(q walletdb.Transactor) (*chain.Transaction, error) {
		owner := state.Info.Owner
		ownerCoin, err := walletdb.GetCoinByOutpoint(q, a.id, owner.Hash, owner.Index)
		if err == sql.ErrNoRows {
			return nil, errors.New("you do not own this name")
		}
		if err != nil {
			return nil, err
		}

		coin := ConvertDBCoin(ownerCoin)
		covType := coin.Covenant.Type
		if covType != chain.CovenantRegister &&
			covType != chain.CovenantUpdate &&
			covType != chain.CovenantRenew &&
			covType != chain.CovenantTransfer &&
			covType != chain.CovenantFinalize {
			return nil, errors.New("name must be registered")
		}

		txb := new(TxBuilder)
		txb.AddCoin(coin)
		txb.AddOutput(&chain.Output{
			Value:   coin.Value,
			Address: coin.Address,
			Covenant: &chain.Covenant{
				Type: chain.CovenantRevoke,
				Items: [][]byte{
					chain.HashName(name),
					bio.Uint32LE(uint32(state.Info.Height)),
				},
			},
		})

		tx, err := a.fundTx(q, txb, feeRate)
		if err != nil {
			return nil, err
		}
		if err := a.sendTx(q, tx); err != nil {
			return nil, err
		}
		return tx, nil
	})
}

func (a *Account) Zap() error {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	return a.engine.Transaction(func(q walletdb.Transactor) error {
		return walletdb.Zap(q, a.id)
	})
}

func (a *Account) Rescan(height int) error {
	if a.bm.LastHeight() < height {
		return errors.New("cannot rescan beyond the chain head")
	}

	if height < 0 {
		return errors.New("cannot rescan to a negative height")
	}

	a.tmb.Go(func() error {
		a.mtx.Lock()
		defer a.mtx.Unlock()
		if err := a.rollback(height); err != nil {
			a.lgr.Error("error rolling back", "err", err)
			return nil
		}
		if err := a.rescan(a.bm.LastHeight()); err != nil {
			a.lgr.Error("error rescanning", "err", err)
		}
		return nil
	})

	return nil
}

func (a *Account) SignMessage(addr *chain.Address, msg []byte) (*btcec.Signature, error) {
	var dbAddr *walletdb.Address
	err := a.engine.Transaction(func(tx walletdb.Transactor) error {
		dba, err := walletdb.GetAddress(tx, a.id, addr.String(a.network))
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("address not found")
		}
		if err != nil {
			return err
		}
		dbAddr = dba
		return nil
	})
	if err != nil {
		return nil, err
	}

	key, err := a.ring.PrivateKey(uint32(dbAddr.Branch), uint32(dbAddr.Idx))
	if err != nil {
		return nil, err
	}

	h, _ := blake2b.New256(nil)
	h.Write([]byte(chain.SignMessageMagic))
	h.Write(msg)
	return key.Sign(h.Sum(nil))
}

func (a *Account) SignMessageWithName(name string, msg []byte) (*btcec.Signature, error) {
	info, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	var dbCoin *walletdb.Coin
	err = a.engine.Transaction(func(tx walletdb.Transactor) error {
		owner := info.Info.Owner
		c, err := walletdb.GetCoinByOutpoint(tx, a.id, owner.Hash, owner.Index)
		if err != nil {
			return err
		}
		dbCoin = c
		return err
	})
	if err != nil {
		return nil, err
	}

	return a.SignMessage(chain.MustAddressFromBech32(dbCoin.Address), msg)
}

func (a *Account) UnspentBids(count, offset int) ([]*UnspentBid, error) {
	var unspents []*walletdb.UnspentBid
	err := a.engine.Transaction(func(tx walletdb.Transactor) error {
		usbs, err := walletdb.GetUnspentBids(tx, a.id, count, offset)
		if err != nil {
			return err
		}
		unspents = usbs
		return nil
	})

	if len(unspents) == 0 {
		return make([]*UnspentBid, 0), nil
	}

	names := make([]string, len(unspents))
	for i, unspent := range unspents {
		names[i] = unspent.Name
	}

	infos, err := a.client.BatchGetNameInfo(names)
	if err != nil {
		return nil, err
	}

	var filtered []*UnspentBid
	for i, info := range infos {
		if info.Error != nil {
			a.lgr.Warning("error getting revealable bid name info", "err", info.Error)
			continue
		}

		dbUnspent := unspents[i]
		revealable := info.Info.Info.State == "REVEAL"
		// TODO: handle names that come up for auction again
		revealableBlock := -1
		if info.Info.Info.State == "BIDDING" {
			revealableBlock = info.Info.Info.Stats.BidPeriodEnd
		}

		filtered = append(filtered, &UnspentBid{
			Name:            dbUnspent.Name,
			BlockHeight:     dbUnspent.BlockHeight,
			Lockup:          dbUnspent.Lockup,
			BidValue:        dbUnspent.BidValue,
			TxHash:          dbUnspent.TxHash,
			OutIdx:          dbUnspent.OutIdx,
			Revealable:      revealable,
			RevealableBlock: revealableBlock,
		})
	}

	return filtered, err
}

func (a *Account) UnspentReveals(count, offset int) ([]*UnspentReveal, error) {
	var unspents []*walletdb.UnspentReveal
	err := a.engine.Transaction(func(tx walletdb.Transactor) error {
		usbs, err := walletdb.GetUnspentReveals(tx, a.id, count, offset)
		if err != nil {
			return err
		}
		unspents = usbs
		return nil
	})

	if len(unspents) == 0 {
		return make([]*UnspentReveal, 0), nil
	}

	names := make([]string, len(unspents))
	for i, unspent := range unspents {
		names[i] = unspent.Name
	}

	infos, err := a.client.BatchGetNameInfo(names)
	if err != nil {
		return nil, err
	}

	var filtered []*UnspentReveal
	for i, info := range infos {
		if info.Error != nil {
			a.lgr.Warning("error getting redeemable bid name info", "err", info.Error)
			continue
		}

		dbUnspent := unspents[i]
		redeemable := info.Info.Info.State == "CLOSED" && dbUnspent.Value > 0
		redeemableBlock := -1
		if info.Info.Info.State == "REVEAL" {
			redeemableBlock = info.Info.Info.Stats.RevealPeriodEnd
		}

		// TODO: handle names that come up for auction again

		filtered = append(filtered, &UnspentReveal{
			Name:            dbUnspent.Name,
			BlockHeight:     dbUnspent.BlockHeight,
			Value:           dbUnspent.Value,
			TxHash:          dbUnspent.TxHash,
			OutIdx:          dbUnspent.OutIdx,
			Redeemable:      redeemable,
			RedeemableBlock: redeemableBlock,
		})
	}

	return filtered, err
}

func (a *Account) rollback(height int) error {
	a.lgr.Info("rolling back account", "height", height)

	err := a.engine.Transaction(func(tx walletdb.Transactor) error {
		return walletdb.Rollback(tx, a.id, height)
	})
	if err != nil {
		return err
	}

	a.rescanHeight = height
	return nil
}

func (a *Account) lockedRescan(notif *BlockNotification) error {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	if a.rescanHeight > notif.CommonTip && notif.ChainTip > notif.CommonTip {
		if err := a.rollback(notif.CommonTip); err != nil {
			return err
		}
	}

	return a.rescan(notif.ChainTip)
}

func (a *Account) rescan(chainHeight int) error {
	rescanHeight := a.rescanHeight
	if chainHeight == rescanHeight {
		a.lgr.Debug("index up-to-date, skipping scan", "height", rescanHeight)
		return nil
	}

	a.lgr.Info("scanning account", "height", rescanHeight, "chain_height", chainHeight)

	var j int
	for i := rescanHeight + 1; i <= chainHeight; i += BlockFetchConcurrency {
		count := BlockFetchConcurrency
		if i+count > chainHeight {
			count = chainHeight - i + 1
		}

		blocks, err := GetRawBlocksConcurrently(a.client, i, count)
		if err != nil {
			return errors.Wrap(err, "error fetching block")
		}

		for resIdx, block := range blocks {
			if err := a.scanBlock(resIdx+i, block); err != nil {
				return errors.Wrap(err, "error processing block")
			}
		}

		j += BlockFetchConcurrency
		if j%(BlockFetchConcurrency*20) == 0 {
			a.lgr.Info(
				"rescan in progress",
				"height",
				i,
				"chain_height",
				chainHeight,
			)
		}
	}

	err := a.engine.Transaction(func(tx walletdb.Transactor) error {
		return walletdb.UpdateRescanHeight(tx, a.id, a.rescanHeight)
	})
	if err != nil {
		return err
	}

	a.lgr.Info("scan complete", "height", chainHeight)
	return nil
}

func (a *Account) scanBlock(height int, block *chain.Block) error {
	// see if we spend/receive any coins in this block to avoid
	// hitting the disk for every block.
	var shouldScan bool
	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			if a.outpointBloom.Test(input.Prevout) {
				shouldScan = true
				goto check
			}
		}

		for _, out := range tx.Outputs {
			if a.addrManager.HasAddress(out.Address) {
				shouldScan = true
				goto check
			}
		}
	}

check:
	if !shouldScan {
		// record rescan height every 50 blocks if they're empty
		if height%50 == 0 {
			err := a.engine.Transaction(func(tx walletdb.Transactor) error {
				return walletdb.UpdateRescanHeight(tx, a.id, height)
			})
			if err != nil {
				return err
			}
		}

		a.rescanHeight = height
		return nil
	}

	var spends int
	var coins int
	err := a.engine.Transaction(func(dTx walletdb.Transactor) error {
		for txIdx, tx := range block.Transactions {
			var indexed bool
			coinbase := bytes.Equal(tx.Inputs[0].Prevout.Hash, chain.ZeroHash)

			for inIdx, input := range tx.Inputs {
				if !a.outpointBloom.Test(input.Prevout) {
					continue
				}

				if !indexed {
					dbTx := &walletdb.Transaction{
						Hash:        tx.IDHex(),
						Idx:         txIdx,
						BlockHeight: height,
						BlockHash:   hex.EncodeToString(block.Hash()),
						Raw:         tx.Bytes(),
						Time:        int(block.Time),
					}
					if _, err := walletdb.UpsertTransaction(dTx, a.id, dbTx); err != nil {
						return err
					}
					indexed = true
				}

				if err := a.scanInput(dTx, tx, txIdx, height, inIdx); err != nil {
					return err
				}
				spends++
			}

			for outIdx, out := range tx.Outputs {
				if !a.addrManager.HasAddress(out.Address) {
					continue
				}

				if !indexed {
					dbTx := &walletdb.Transaction{
						Hash:        tx.IDHex(),
						Idx:         txIdx,
						BlockHeight: height,
						BlockHash:   hex.EncodeToString(block.Hash()),
						Raw:         tx.Bytes(),
						Time:        int(block.Time),
					}
					if _, err := walletdb.UpsertTransaction(dTx, a.id, dbTx); err != nil {
						return err
					}
					indexed = true
				}

				if err := a.scanOutput(dTx, tx, height, txIdx, outIdx, coinbase); err != nil {
					return err
				}
				coins++
			}
		}

		return walletdb.UpdateRescanHeight(dTx, a.id, height)
	})
	if err != nil {
		return err
	}

	if spends > 0 || coins > 0 {
		a.lgr.Info(
			"added transactions to wallet db",
			"height", height,
			"spends", spends,
			"coins", coins,
		)
	}

	a.rescanHeight = height
	return nil
}

func (a *Account) scanInput(q walletdb.Transactor, tx *chain.Transaction, height, txIdx, inIdx int) error {
	prevout := tx.Inputs[inIdx].Prevout
	coin, err := walletdb.GetCoinByOutpoint(q, a.id, hex.EncodeToString(prevout.Hash), int(prevout.Index))
	if errors.Is(err, sql.ErrNoRows) {
		a.lgr.Warning(
			"bloom filter false positive",
			"account_name", a.name,
			"prevout", fmt.Sprintf("%x/%d", prevout.Hash, prevout.Index),
		)
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "error getting coin from DB")
	}

	if coin.SpendingTxHash != "" {
		a.lgr.Debug(
			"coin already spent",
			"height", height,
			"tx_idx", txIdx,
			"input_idx", inIdx,
			"addr", coin.Address,
			"outpoint_tx_hash", hex.EncodeToString(prevout.Hash),
			"outpoint_idx", prevout.Index,
		)
		return nil
	}

	a.lgr.Debug(
		"found spend",
		"height", height,
		"tx_idx", txIdx,
		"input_idx", inIdx,
		"addr", coin.Address,
		"outpoint_tx_hash", hex.EncodeToString(prevout.Hash),
		"outpoint_idx", prevout.Index,
	)

	err = walletdb.UpdateCoinSpent(
		q,
		coin.ID,
		tx.IDHex(),
	)
	if err != nil {
		return errors.Wrap(err, "error updating coin spent")
	}

	pCoin := ConvertDBCoin(coin)
	if pCoin.Covenant.Type != chain.CovenantTransfer {
		return nil
	}

	nameHash := pCoin.Covenant.Items[0]
	finalizeIdx := -1
	for i, out := range tx.Outputs {
		if out.Covenant.Type == chain.CovenantFinalize {
			finalizeIdx = i
			break
		}
	}

	if finalizeIdx == -1 {
		return nil
	}

	if err := walletdb.UpsertNameHash(q, a.id, nameHash, walletdb.NameStatusTransferred); err != nil {
		return err
	}

	return walletdb.UpdateNameHistory(q, &walletdb.NameHistory{
		AccountID:    a.id,
		Type:         walletdb.NameActionFinalizeOut,
		NameHash:     nameHash,
		TxHash:       tx.IDHex(),
		OutIdx:       finalizeIdx,
		Value:        tx.Outputs[finalizeIdx].Value,
		ParentTxHash: hex.EncodeToString(prevout.Hash),
		ParentOutIdx: prevout.Index,
	})
}

func (a *Account) scanOutput(q walletdb.Transactor, tx *chain.Transaction, height, txIdx, outIdx int, coinbase bool) error {
	output := tx.Outputs[outIdx]
	addr := output.Address
	addrBech := addr.String(a.network)
	lookAddr, err := a.checkAddrInDB(q, addr)
	if err != nil {
		return errors.Wrap(err, "error getting addr from DB")
	}

	if lookAddr == nil {
		a.lgr.Warning(
			"bloom filter false positive",
			"account_name", a.name,
			"addr", addrBech,
		)
		return nil
	}

	_, err = walletdb.GetCoinByOutpoint(q, a.id, tx.IDHex(), outIdx)
	path := chain.Derivation{uint32(lookAddr.Branch), uint32(lookAddr.Idx)}
	if err == nil {
		a.lgr.Debug(
			"coin already exists",
			"height", height,
			"tx_idx", txIdx,
			"out_idx", outIdx,
			"addr", addrBech,
			"path", path,
		)
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return errors.Wrap(err, "error checking coin existence")
	}

	a.lgr.Debug(
		"found coin",
		"height", height,
		"tx_idx", txIdx,
		"out_idx", outIdx,
		"addr", addrBech,
		"path", path,
	)

	if err := a.addrManager.IncLookahead(q, uint32(lookAddr.Branch), uint32(lookAddr.Idx)+1); err != nil {
		return err
	}

	outpointBloomCopy := a.outpointBloom.Copy()
	outpointBloomCopy.Add(&chain.Outpoint{
		Hash:  tx.ID(),
		Index: uint32(outIdx),
	})

	err = walletdb.CreateCoin(q, &walletdb.CreateCoinOpts{
		AccountID:     a.id,
		TxHash:        tx.IDHex(),
		OutIdx:        outIdx,
		Value:         output.Value,
		Address:       addr.String(a.network),
		Coinbase:      coinbase,
		CovenantType:  output.Covenant.Type.String(),
		CovenantItems: output.Covenant.Items,
	})

	if err != nil {
		return err
	}

	if err = walletdb.UpdateOutpointBloom(q, a.id, outpointBloomCopy.Bytes()); err != nil {
		return errors.Wrap(err, "error updating account bloom filters")
	}

	if err := a.scanCovenant(q, tx, output, outIdx); err != nil {
		return errors.Wrap(err, "error scanning covenants")
	}

	a.outpointBloom = outpointBloomCopy
	return nil
}

func (a *Account) scanCovenant(q walletdb.Transactor, tx *chain.Transaction, out *chain.Output, outIdx int) error {
	entry := &walletdb.NameHistory{
		AccountID: a.id,
		TxHash:    tx.IDHex(),
		OutIdx:    outIdx,
		Value:     out.Value,
	}

	switch out.Covenant.Type {
	case chain.CovenantOpen:
		entry.Name = string(out.Covenant.Items[2])
		entry.Type = walletdb.NameActionOpen
		if err := walletdb.UpsertName(q, a.id, entry.Name, walletdb.NameStatusUnowned); err != nil {
			return err
		}
	case chain.CovenantBid:
		entry.Name = string(out.Covenant.Items[2])
		entry.Type = walletdb.NameActionBid
		if err := walletdb.UpsertName(q, a.id, entry.Name, walletdb.NameStatusUnowned); err != nil {
			return err
		}
	case chain.CovenantReveal:
		input := tx.Inputs[outIdx]
		entry.NameHash = out.Covenant.Items[0]
		entry.Type = walletdb.NameActionReveal
		entry.ParentTxHash = hex.EncodeToString(input.Prevout.Hash)
		entry.ParentOutIdx = input.Prevout.Index
	case chain.CovenantRedeem:
		input := tx.Inputs[outIdx]
		entry.NameHash = out.Covenant.Items[0]
		entry.Type = walletdb.NameActionRedeem
		entry.ParentTxHash = hex.EncodeToString(input.Prevout.Hash)
		entry.ParentOutIdx = input.Prevout.Index
	case chain.CovenantRegister:
		entry.NameHash = out.Covenant.Items[0]
		entry.Type = walletdb.NameActionRegister
		if err := walletdb.UpsertNameHash(q, a.id, entry.NameHash, walletdb.NameStatusOwned); err != nil {
			return err
		}
	case chain.CovenantTransfer:
		entry.NameHash = out.Covenant.Items[0]
		entry.Type = walletdb.NameActionTransfer
		if err := walletdb.UpsertNameHash(q, a.id, entry.NameHash, walletdb.NameStatusTransferring); err != nil {
			return err
		}
	case chain.CovenantFinalize:
		name := string(out.Covenant.Items[2])
		entry.Name = name
		entry.Type = walletdb.NameActionFinalizeIn
		if err := walletdb.UpsertName(q, a.id, entry.Name, walletdb.NameStatusOwned); err != nil {
			return err
		}
	case chain.CovenantRevoke:
		input := tx.Inputs[outIdx]
		entry.NameHash = out.Covenant.Items[0]
		entry.Type = walletdb.NameActionRevoke
		entry.ParentTxHash = hex.EncodeToString(input.Prevout.Hash)
		entry.ParentOutIdx = input.Prevout.Index
		if err := walletdb.UpsertNameHash(q, a.id, entry.NameHash, walletdb.NameStatusRevoked); err != nil {
			return err
		}
	default:
		return nil
	}

	return errors.Wrap(walletdb.UpdateNameHistory(q, entry), "error saving name history")
}

func (a *Account) checkAddrInDB(q walletdb.Querier, addr *chain.Address) (*walletdb.Address, error) {
	addrBech := addr.String(a.network)
	dbAddr, err := walletdb.GetAddress(q, a.id, addrBech)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return dbAddr, errors.Wrap(err, "error getting address")
}

func (a *Account) send(dTx walletdb.Transactor, addr *chain.Address, value uint64, feeRate uint64) (*chain.Transaction, error) {
	txb := new(TxBuilder)
	txb.AddOutput(&chain.Output{
		Value:    value,
		Address:  addr,
		Covenant: chain.EmptyCovenant,
	})

	var tx *chain.Transaction

	tx, err := a.fundTx(dTx, txb, feeRate)
	if err != nil {
		return nil, err
	}
	if err := a.sendTx(dTx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (a *Account) sendReveals(dTx walletdb.Transactor, bids []*walletdb.RevealableBid, name string, nsHeight int, feeRate uint64) (*chain.Transaction, error) {
	var tx *chain.Transaction
	var err error
	txb := new(TxBuilder)
	coins := make([]*chain.Coin, len(bids))
	for i, bid := range bids {
		bidCoin, err := walletdb.GetCoinByOutpoint(dTx, a.id, bid.TxHash, bid.OutIdx)
		if err != nil {
			return nil, err
		}

		coin := ConvertDBCoin(bidCoin)
		nonce := chain.GenerateNonce(a.ring.PublicEK(), name, coin.Address, bid.BidValue)
		coins[i] = coin

		txb.AddCoin(coin)
		txb.AddOutput(&chain.Output{
			Value:   bid.BidValue,
			Address: chain.MustAddressFromBech32(bidCoin.Address),
			Covenant: &chain.Covenant{
				Type: chain.CovenantReveal,
				Items: [][]byte{
					chain.HashName(name),
					bio.Uint32LE(uint32(nsHeight)),
					nonce,
				},
			},
		})
	}

	tx, err = a.fundTx(dTx, txb, feeRate)
	if err != nil {
		return nil, err
	}

	return tx, a.sendTx(dTx, tx)
}

func (a *Account) sendRedeems(dTx walletdb.Transactor, reveals []*walletdb.RedeemableReveal, name string, nsHeight int, feeRate uint64) (*chain.Transaction, error) {
	var tx *chain.Transaction
	var err error
	txb := new(TxBuilder)
	coins := make([]*chain.Coin, len(reveals))
	for i, rev := range reveals {
		revCoin, err := walletdb.GetCoinByOutpoint(dTx, a.id, rev.TxHash, rev.OutIdx)
		if err != nil {
			return nil, err
		}

		coin := ConvertDBCoin(revCoin)
		coins[i] = coin

		txb.AddCoin(coin)
		txb.AddOutput(&chain.Output{
			Value:   revCoin.Value,
			Address: coin.Address,
			Covenant: &chain.Covenant{
				Type: chain.CovenantRedeem,
				Items: [][]byte{
					chain.HashName(name),
					bio.Uint32LE(uint32(nsHeight)),
				},
			},
		})
	}

	tx, err = a.fundTx(dTx, txb, feeRate)
	if err != nil {
		return nil, err
	}
	if err := a.sendTx(dTx, tx); err != nil {
		return nil, err
	}
	return tx, err
}

func (a *Account) sendUpdate(q walletdb.Transactor, name string, resource *chain.Resource, feeRate uint64) (*chain.Transaction, error) {
	hasName, err := walletdb.HasOwnedName(q, a.id, name)
	if err != nil {
		return nil, errors.Wrap(err, "error checking for id")
	}

	state, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	revCoin, err := walletdb.GetCoinByOutpoint(q, a.id, state.Info.Owner.Hash, state.Info.Owner.Index)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("you did not win this auction")
	}
	if revCoin.SpendingTxHash != "" {
		return nil, errors.New("name is already registered")
	}

	buf := new(bytes.Buffer)
	if resource == nil {
		buf.WriteByte(0x00)
	} else {
		if _, err := resource.WriteTo(buf); err != nil {
			return nil, errors.Wrap(err, "error serializing resource")
		}
	}

	txb := new(TxBuilder)
	coin := ConvertDBCoin(revCoin)
	txb.AddCoin(coin)

	if hasName {
		txb.AddOutput(&chain.Output{
			Value:   uint64(state.Info.Value),
			Address: chain.MustAddressFromBech32(revCoin.Address),
			Covenant: &chain.Covenant{
				Type: chain.CovenantUpdate,
				Items: [][]byte{
					chain.HashName(name),
					bio.Uint32LE(uint32(state.Info.Height)),
					buf.Bytes(),
				},
			},
		})
	} else {
		renewalBlockB, err := a.client.GetRenewalBlock(a.network, state.Info.Height)
		if err != nil {
			return nil, errors.New("error getting renewal block")
		}
		renewalBlock := new(chain.Block)
		if _, err := renewalBlock.ReadFrom(bytes.NewReader(renewalBlockB)); err != nil {
			return nil, errors.New("error parsing renewal block")
		}

		txb.AddOutput(&chain.Output{
			Value:   uint64(state.Info.Value),
			Address: chain.MustAddressFromBech32(revCoin.Address),
			Covenant: &chain.Covenant{
				Type: chain.CovenantRegister,
				Items: [][]byte{
					chain.HashName(name),
					bio.Uint32LE(uint32(state.Info.Height)),
					buf.Bytes(),
					renewalBlock.Hash(),
				},
			},
		})
	}

	tx, err := a.fundTx(q, txb, feeRate)
	if err != nil {
		return nil, err
	}
	if err := a.sendTx(q, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (a *Account) sendRenewal(q walletdb.Transactor, name string, feeRate uint64) (*chain.Transaction, error) {
	if !chain.IsNameValid(name) {
		return nil, errors.New("invalid name")
	}

	state, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	inCoin, err := walletdb.GetCoinByOutpoint(q, a.id, state.Info.Owner.Hash, state.Info.Owner.Index)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("you do not own this name")
	}
	coin := ConvertDBCoin(inCoin)
	covType := coin.Covenant.Type
	if covType != chain.CovenantRegister &&
		covType != chain.CovenantUpdate &&
		covType != chain.CovenantRenew &&
		covType != chain.CovenantFinalize {
		return nil, errors.New("name must be registered")
	}

	if a.rescanHeight < state.Info.Renewal+a.network.TreeInterval {
		return nil, errors.New("must wait to renew")
	}

	renewalBlock, err := a.getRenewalBlock(state.Info.Height)
	if err != nil {
		return nil, err
	}

	txb := new(TxBuilder)
	txb.AddCoin(coin)
	txb.AddOutput(&chain.Output{
		Value:   coin.Value,
		Address: coin.Address,
		Covenant: &chain.Covenant{
			Type: chain.CovenantRenew,
			Items: [][]byte{
				chain.HashName(name),
				bio.Uint32LE(uint32(state.Info.Height)),
				renewalBlock.Hash(),
			},
		},
	})

	tx, err := a.fundTx(q, txb, feeRate)
	if err != nil {
		return nil, err
	}
	if err := a.sendTx(q, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (a *Account) fundTx(q walletdb.Querier, txb *TxBuilder, feeRate uint64) (*chain.Transaction, error) {
	if feeRate == 0 {
		smartFee, err := a.client.EstimateSmartFee(10)
		if err != nil {
			a.lgr.Warning("error estimating smart fee", "err", err)
			feeRate = 100
		} else if smartFee < 100 {
			a.lgr.Warning("smart fee less than minimum")
			feeRate = 100
		} else {
			feeRate = smartFee
		}
	}

	dbCoins, err := walletdb.GetFundingCoins(q, a.id, a.network, a.rescanHeight)
	if err != nil {
		return nil, err
	}

	coins := make([]*chain.Coin, len(dbCoins))
	for i := 0; i < len(dbCoins); i++ {
		coins[i] = ConvertDBCoin(dbCoins[i])
	}

	changeAddr := a.addrManager.ChangeAddress()
	if err := txb.Fund(coins, changeAddr, feeRate); err != nil {
		return nil, err
	}

	if err := txb.Sign(a.ring); err != nil {
		return nil, err
	}

	return txb.Build(), nil
}

func (a *Account) sendTx(dTx walletdb.Transactor, tx *chain.Transaction) error {
	hashStr := tx.IDHex()
	_, err := walletdb.UpsertTransaction(dTx, a.id, &walletdb.Transaction{
		Hash:        hashStr,
		Idx:         -1,
		BlockHeight: -1,
		BlockHash:   hex.EncodeToString(chain.ZeroHash),
		Raw:         tx.Bytes(),
		Time:        -1,
	})
	if err != nil {
		return errors.Wrap(err, "error upserting transaction")
	}

	for i := range tx.Inputs {
		if err := a.scanInput(dTx, tx, -1, 0, i); err != nil {
			return errors.Wrap(err, "error scanning input")
		}
	}
	for i := range tx.Outputs {
		if err := a.scanOutput(dTx, tx, -1, 0, i, false); err != nil {
			return errors.Wrap(err, "error scanning input")
		}
	}

	if _, err := a.client.SendRawTransaction(tx.Bytes()); err != nil {
		return errors.Wrap(err, "error broadcasting transaction")
	}

	for i := 0; i < 30; i++ {
		// wait a little bit since the transaction will appear
		// in the node's mempool for a moment before being
		// removed if rejected
		time.Sleep(100 * time.Millisecond)

		mp, err := a.client.GetRawMempool()
		if err != nil {
			return errors.Wrap(err, "error checking for tx in mempool")
		}

		for _, hash := range mp {
			if hash == hashStr {
				return nil
			}
		}
	}

	return errors.New("transaction did not appear in mempool")
}

func (a *Account) txTransactor(cb func(dTx walletdb.Transactor) (*chain.Transaction, error)) (*chain.Transaction, error) {
	var tx *chain.Transaction
	err := a.engine.Transaction(func(dTx walletdb.Transactor) error {
		inTx, err := cb(dTx)
		if err != nil {
			return err
		}
		tx = inTx
		return nil
	})
	return tx, err
}

func (a *Account) getRenewalBlock(height int) (*chain.Block, error) {
	renewalBlockB, err := a.client.GetRenewalBlock(a.network, height)
	if err != nil {
		return nil, errors.New("error getting renewal block")
	}
	renewalBlock := new(chain.Block)
	if _, err := renewalBlock.ReadFrom(bytes.NewReader(renewalBlockB)); err != nil {
		return nil, errors.New("error parsing renewal block")
	}
	return renewalBlock, nil
}

func (a *Account) requireNameState(name string, expState string) (*client.NameInfoRes, error) {
	state, err := a.client.GetNameInfo(name)
	if err != nil {
		return nil, err
	}
	if state.Info == nil {
		return nil, errors.Errorf("name state must be %s", expState)
	}
	if state.Info.State != expState {
		return nil, errors.Errorf("name state must be %s", expState)
	}

	var checkBlock int
	switch expState {
	case "BIDDING":
		checkBlock = state.Info.Stats.BidPeriodEnd - 1
	case "REVEAL":
		checkBlock = state.Info.Stats.RevealPeriodEnd - 1
	}

	if checkBlock > 0 && a.rescanHeight >= checkBlock {
		return nil, errors.Errorf("%s period has passed", strings.ToLower(expState))
	}

	return state, nil
}
