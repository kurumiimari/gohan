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
	"github.com/kurumiimari/gohan/shakedex"
	"github.com/kurumiimari/gohan/txscript"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip32"
	"golang.org/x/crypto/blake2b"
	"gopkg.in/tomb.v2"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	BlockFetchConcurrency = 5
)

var (
	accIDRegex *regexp.Regexp

	accLogger = log.ModuleLogger("account")
)

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
	keyLocker     *KeyLocker
	ring          Keyring
	addrBloom     *AddressBloom
	recvMgr       *AddressManager
	changeMgr     *AddressManager
	dutchMgr      *AddressManager
	id            string
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
	opts *walletdb.AccountOpts,
) (*Account, error) {
	box, err := UnmarshalSecretBox([]byte(opts.Seed))
	if err != nil {
		return nil, err
	}
	addrBloom, err := NewAddressBloomFromBytes(opts.AddressBloom)
	if err != nil {
		return nil, err
	}
	outBloom, err := NewOutpointBloomFromBytes(opts.OutpointBloom)
	if err != nil {
		return nil, err
	}

	keyLocker := NewKeyLocker(box, network)
	ring := NewAccountKeyring(keyLocker, opts.XPub, network)

	return &Account{
		tmb:       tmb,
		network:   network,
		engine:    engine,
		client:    client,
		bm:        bm,
		keyLocker: keyLocker,
		ring:      ring,
		addrBloom: addrBloom,
		recvMgr: NewAddressManager(
			ring,
			addrBloom,
			opts.ID,
			chain.ReceiveBranch,
			opts.RecvIdx,
			int64(opts.LookaheadTips[chain.ReceiveBranch]),
		),
		changeMgr: NewAddressManager(
			ring,
			addrBloom,
			opts.ID,
			chain.ChangeBranch,
			opts.ChangeIdx,
			int64(opts.LookaheadTips[chain.ChangeBranch]),
		),
		dutchMgr: NewAddressManager(
			ring,
			addrBloom,
			opts.ID,
			shakedex.AddressBranch,
			opts.DutchAuctionIdx,
			int64(opts.LookaheadTips[shakedex.AddressBranch]),
			WithLookSize(10),
			WithAddressMaker(HIP1AddressMaker),
		),
		id:            opts.ID,
		idx:           opts.Idx,
		rescanHeight:  opts.RescanHeight,
		outpointBloom: outBloom,
		lgr: accLogger.Child(
			"id",
			opts.ID,
		),
	}, nil
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

func (a *Account) ID() string {
	return a.id
}

func (a *Account) Index() uint32 {
	return a.idx
}

func (a *Account) Locked() bool {
	return a.keyLocker.Locked()
}

func (a *Account) Unlock(password string) error {
	err := a.keyLocker.Unlock(password)
	if err != nil {
		a.lgr.Warning("unlock attempt failed")
		return err
	}
	a.lgr.Info("wallet unlocked")
	return nil
}

func (a *Account) Lock() {
	a.keyLocker.Lock()
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
	return a.recvMgr.Depth(), a.changeMgr.Depth()
}

func (a *Account) LookaheadDepth() (uint32, uint32) {
	return a.recvMgr.Lookahead(), a.changeMgr.Lookahead()
}

func (a *Account) ReceiveAddress() *chain.Address {
	return a.recvMgr.Address()
}

func (a *Account) ChangeAddress() *chain.Address {
	return a.changeMgr.Address()
}

func (a *Account) XPub() string {
	return a.ring.XPub()
}

func (a *Account) GenerateReceiveAddress() (*chain.Address, uint32, error) {
	var addr *chain.Address
	err := a.engine.Transaction(func(tx walletdb.Transactor) error {
		genAddr, err := a.recvMgr.NextAddress(tx)
		if err != nil {
			return err
		}

		addr = genAddr
		return nil
	})
	return addr, a.recvMgr.Depth(), err
}

func (a *Account) GenerateChangeAddress() (*chain.Address, uint32, error) {
	var addr *chain.Address
	err := a.engine.Transaction(func(tx walletdb.Transactor) error {
		genAddr, err := a.changeMgr.NextAddress(tx)
		if err != nil {
			return err
		}

		addr = genAddr
		return nil
	})
	return addr, a.changeMgr.Depth(), err
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
		hist, err := walletdb.GetNameHistory(tx, a.id, name, count, offset)
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
		t, err := walletdb.ListTransactions(q, a.id, count, offset)
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

		recvAddr := a.recvMgr.Address()

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

	recvAddr := a.recvMgr.Address()
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
			Outpoint: &chain.Outpoint{
				Hash:  tx.ID(),
				Index: 0,
			},
			Value:    lockup,
			BidValue: value,
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
	winnerOutpoint := &chain.Outpoint{
		Hash:  winner.Hash,
		Index: winner.Index,
	}

	return a.txTransactor(func(dTx walletdb.Transactor) (*chain.Transaction, error) {
		coins, err := walletdb.GetRedeemableReveals(dTx, a.id, name)
		if err != nil {
			return nil, err
		}
		if len(coins) == 0 {
			return nil, errors.New("no reveals to redeem")
		}

		losingReveals := make([]*chain.Coin, 0)
		for _, rev := range coins {
			if rev.Prevout.Equal(winnerOutpoint) {
				continue
			}
			losingReveals = append(losingReveals, rev.AsChain())
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

	state, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	var tx *chain.Transaction
	err = a.engine.Transaction(func(q walletdb.Transactor) error {
		coin, err := walletdb.GetOwnedNameCoin(q, a.id, name)
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("you do not own this name")
		}
		if err != nil {
			return err
		}

		txb := new(TxBuilder)
		txb.AddCoin(coin.AsChain())
		txb.AddOutput(&chain.Output{
			Value:    coin.Value,
			Address:  coin.Address,
			Covenant: chain.NewTransferCovenant(name, state.Info.Height, address),
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

	state, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	var tx *chain.Transaction
	err = a.engine.Transaction(func(q walletdb.Transactor) error {
		coin, err := walletdb.GetTransferCoin(q, a.id, name)
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("either you do not own this name or it is not transferring")
		}
		if err != nil {
			return err
		}
		if a.rescanHeight < coin.Height+a.network.TransferLockup {
			return errors.New("transfer is still locked up")
		}

		txb := new(TxBuilder)
		addr := chain.NewAddress(
			coin.Covenant.Items[2][0],
			coin.Covenant.Items[3],
		)

		renewalBlock, err := a.client.GetRenewalBlock(a.network, a.rescanHeight)
		if err != nil {
			return err
		}

		txb.AddCoin(coin.AsChain())
		txb.AddOutput(&chain.Output{
			Value:   coin.Value,
			Address: addr,
			Covenant: chain.NewFinalizeCovenant(
				name,
				state.Info.Weak,
				renewalBlock.Hash(),
				state.Info.Height,
				state.Info.Claimed,
				state.Info.Renewals,
			),
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
		coin, err := walletdb.GetRevocableNameCoin(q, a.id, name)
		if err == sql.ErrNoRows {
			return nil, errors.New("you do not own this name")
		}
		if err != nil {
			return nil, err
		}

		txb := new(TxBuilder)
		txb.AddCoin(coin.AsChain())
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

func (a *Account) TransferDutchAuctionListing(
	name string,
	feeRate uint64,
) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	state, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	return a.txTransactor(func(dTx walletdb.Transactor) (*chain.Transaction, error) {
		coin, err := walletdb.GetOwnedNameCoin(dTx, a.id, name)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("you do not own this name")
		}
		if err != nil {
			return nil, err
		}

		listingADdress := a.dutchMgr.Address()
		txb := new(TxBuilder)
		txb.AddCoin(coin.AsChain())
		txb.AddOutput(&chain.Output{
			Value:    coin.Value,
			Address:  coin.Address,
			Covenant: chain.NewTransferCovenant(name, state.Info.Height, listingADdress),
		})

		tx, err := a.fundTx(dTx, txb, feeRate)
		if err != nil {
			return nil, err
		}

		err = walletdb.TransferDutchAuctionListing(
			dTx,
			a.id,
			name,
			&chain.Outpoint{
				Hash: tx.ID(),
			},
			listingADdress,
		)
		if err != nil {
			return nil, err
		}

		if err := a.sendTx(dTx, tx); err != nil {
			return nil, err
		}
		return tx, nil
	})
}

func (a *Account) FinalizeDutchAuctionListing(
	name string,
	feeRate uint64,
) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	state, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	renewalBlock, err := a.client.GetRenewalBlock(a.network, a.rescanHeight)
	if err != nil {
		return nil, err
	}

	return a.txTransactor(func(dTx walletdb.Transactor) (*chain.Transaction, error) {
		coin, err := walletdb.GetDutchAuctionTransferCoin(
			dTx,
			a.id,
			name,
		)
		if err != nil {
			return nil, err
		}

		txb := new(TxBuilder)
		txb.AddCoin(coin.AsChain())
		txb.AddOutput(&chain.Output{
			Value: coin.Value,
			Address: &chain.Address{
				Version: coin.Covenant.Items[2][0],
				Hash:    coin.Covenant.Items[3],
			},
			Covenant: chain.NewFinalizeCovenant(
				name,
				state.Info.Weak,
				renewalBlock.Hash(),
				state.Info.Height,
				state.Info.Claimed,
				state.Info.Renewals,
			),
		})

		tx, err := a.fundTx(dTx, txb, feeRate)
		if err != nil {
			return nil, err
		}

		err = walletdb.FinalizeDutchAuctionListing(
			dTx,
			coin.Prevout,
			&chain.Outpoint{
				Hash: tx.ID(),
			},
		)
		if err != nil {
			return nil, err
		}

		if err := a.sendTx(dTx, tx); err != nil {
			return nil, err
		}

		return tx, nil
	})
}

func (a *Account) UpdateDutchAuctionListing(
	name string,
	startPrice,
	endPrice uint64,
	feeAddress *chain.Address,
	feePercent float64,
	numDecrements int,
	decrementDurationSecs int64,
) (*shakedex.DutchAuction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	var presigns *shakedex.DutchAuction
	var err error
	err = a.engine.Transaction(func(tx walletdb.Transactor) error {
		coin, err := walletdb.GetTransferrableDutchAuctionCancelCoin(tx, a.id, name)
		if err != nil {
			return errors.New("no auction found")
		}

		privKey, err := a.ring.PrivateKey(coin.Derivation...)
		if err != nil {
			return err
		}

		paymentAddr := a.recvMgr.Address()
		now := time.Now().Unix()
		presigns, err = shakedex.CreateDutchAuction(
			coin.Prevout,
			coin.Value,
			name,
			now,
			startPrice,
			endPrice,
			feePercent,
			numDecrements,
			time.Duration(decrementDurationSecs)*time.Second,
			paymentAddr,
			feeAddress,
			privKey,
		)
		if err != nil {
			return err
		}
		return walletdb.UpdateDutchAuctionListingParams(
			tx,
			name,
			paymentAddr,
			uint32(now),
			startPrice,
			endPrice,
			feeAddress,
			feePercent,
			numDecrements,
			decrementDurationSecs,
		)
	})
	return presigns, err
}

func (a *Account) FillDutchAuction(
	name string,
	finalizeOutpoint *chain.Outpoint,
	paymentAddress *chain.Address,
	feeAddress *chain.Address,
	publicKey []byte,
	signature []byte,
	lockTime uint32,
	bid,
	auctionFee,
	feeRate uint64,
) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	state, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	ownerOutpoint := &chain.Outpoint{
		Hash:  state.Info.Owner.Hash,
		Index: state.Info.Owner.Index,
	}

	if !ownerOutpoint.Equal(finalizeOutpoint) {
		return nil, errors.New("locking script does not own name")
	}

	lockCoin, err := a.client.GetCoinByOutpoint(
		finalizeOutpoint.Hash.String(),
		int(finalizeOutpoint.Index),
	)
	if err != nil {
		return nil, err
	}
	if lockCoin.Covenant.Type != int(chain.CovenantFinalize) {
		return nil, errors.New("locking coin is not a finalize")
	}
	if lockCoin.Covenant.Items[0] != chain.HashName(name).String() {
		return nil, errors.New("locking coin is not for this name")
	}
	if time.Now().Unix() < int64(lockTime) {
		return nil, errors.New("coin is locked")
	}

	auction := &shakedex.DutchAuction{
		Name:            name,
		LockingOutpoint: finalizeOutpoint,
		PublicKey:       publicKey,
		PaymentAddress:  paymentAddress,
		FeeAddress:      feeAddress,
		Bids: []*shakedex.DutchAuctionBid{
			{
				Value:     bid,
				LockTime:  lockTime,
				Fee:       auctionFee,
				Signature: signature,
			},
		},
	}
	if !auction.VerifyAllBids(lockCoin.Value) {
		return nil, errors.New("auction contains invalid bids")
	}

	lockCoinAddr, err := chain.NewAddressFromBech32(lockCoin.Address)
	if err != nil {
		return nil, err
	}
	lockCoinCovItems, err := bio.DecodeHexArray(lockCoin.Covenant.Items)
	if err != nil {
		return nil, err
	}
	lockCoinPrevoutHash, err := hex.DecodeString(lockCoin.Hash)
	if err != nil {
		return nil, err
	}

	txb := new(TxBuilder)
	tmplTx, err := auction.TXTemplate(lockCoin.Value, 0)
	if err != nil {
		return nil, err
	}
	txb.AddCoin(&chain.Coin{
		Version: uint8(lockCoin.Version),
		Height:  lockCoin.Height,
		Value:   lockCoin.Value,
		Address: lockCoinAddr,
		Covenant: &chain.Covenant{
			Type:  chain.CovenantFinalize,
			Items: lockCoinCovItems,
		},
		Prevout: &chain.Outpoint{
			Hash:  lockCoinPrevoutHash,
			Index: uint32(lockCoin.Index),
		},
		Coinbase: false,
	})
	txb.Locktime = tmplTx.LockTime
	txb.Outputs = tmplTx.Outputs
	txb.Witnesses = tmplTx.Witnesses

	recvAddr := a.ReceiveAddress()
	txb.Outputs[0].Covenant.Items = [][]byte{
		chain.HashName(name),
		bio.Uint32LE(uint32(state.Info.Height)),
		{recvAddr.Version},
		recvAddr.Hash,
	}

	return a.txTransactor(func(dTx walletdb.Transactor) (*chain.Transaction, error) {
		tx, err := a.fundTx(dTx, txb, feeRate)
		if err != nil {
			return nil, err
		}

		// reorder to preserve singlereverse, then resign
		if len(tx.Outputs) > len(tmplTx.Outputs) {
			tx.Outputs[len(tx.Outputs)-1], tx.Outputs[len(tx.Outputs)-2] =
				tx.Outputs[len(tx.Outputs)-2], tx.Outputs[len(tx.Outputs)-1]

			for i := 1; i < len(txb.Coins); i++ {
				pk, err := a.ring.PrivateKey(txb.Coins[i].Derivation...)
				if err != nil {
					return nil, err
				}

				newWitness, err := txscript.P2PKHWitnessSignature(
					tx,
					i,
					txb.Coins[i].Value,
					pk,
				)
				tx.Witnesses[i] = newWitness
			}
		}

		if err := a.sendTx(dTx, tx); err != nil {
			return nil, err
		}
		return tx, nil
	})
}

func (a *Account) FinalizeDutchAuction(
	name string,
	feeRate uint64,
) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	state, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	return a.txTransactor(func(dTx walletdb.Transactor) (*chain.Transaction, error) {
		coin, recipAddr, err := walletdb.GetFinalizableDutchAuctionFillCoin(dTx, a.id, name)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("you did not win this auction")
		}
		if err != nil {
			return nil, err
		}
		if a.rescanHeight < coin.Height+a.network.TransferLockup {
			return nil, errors.New("transfer is still locked up")
		}

		var flags uint8
		if state.Info.Weak {
			flags |= 1
		}

		renewalBlock, err := a.client.GetRenewalBlock(a.network, a.rescanHeight)
		if err != nil {
			return nil, err
		}

		txb := new(TxBuilder)
		txb.AddCoin(coin.AsChain())
		txb.AddOutput(&chain.Output{
			Value:   coin.Value,
			Address: recipAddr.Address,
			Covenant: chain.NewFinalizeCovenant(
				name,
				state.Info.Weak,
				renewalBlock.Hash(),
				state.Info.Height,
				state.Info.Claimed,
				state.Info.Renewals,
			),
		})
		prevTxDb, err := walletdb.GetTransactionByOutpoint(dTx, a.id, coin.Prevout.Hash)
		if err != nil {
			return nil, err
		}
		prevTx := new(chain.Transaction)
		if _, err := prevTx.ReadFrom(bytes.NewReader(prevTxDb.Raw)); err != nil {
			return nil, err
		}
		script := prevTx.Witnesses[coin.Prevout.Index].Items[1]
		wit := &chain.Witness{
			Items: [][]byte{
				script,
			},
		}
		txb.Witnesses = append(txb.Witnesses, wit)

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

func (a *Account) TransferDutchAuctionCancel(name string, feeRate uint64) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	state, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	return a.txTransactor(func(dTx walletdb.Transactor) (*chain.Transaction, error) {
		coin, err := walletdb.GetTransferrableDutchAuctionCancelCoin(dTx, a.id, name)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("no auction found")
		}
		if err != nil {
			return nil, err
		}

		destAddress := a.recvMgr.Address()

		txb := new(TxBuilder)
		txb.AddCoin(coin.AsChain())
		txb.AddOutput(&chain.Output{
			Value:    coin.Value,
			Address:  coin.Address,
			Covenant: chain.NewTransferCovenant(name, state.Info.Height, destAddress),
		})

		privKey, err := a.ring.PrivateKey(coin.Derivation[0], coin.Derivation[1])
		if err != nil {
			return nil, err
		}
		wit, err := txscript.HIP1CancelWitnessSignature(txb.Build(), 0, coin.Value, privKey)
		if err != nil {
			return nil, err
		}
		txb.Witnesses = append(txb.Witnesses, wit)

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

func (a *Account) FinalizeDutchAuctionCancel(name string, feeRate uint64) (*chain.Transaction, error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	state, err := a.requireNameState(name, "CLOSED")
	if err != nil {
		return nil, err
	}

	return a.txTransactor(func(dTx walletdb.Transactor) (*chain.Transaction, error) {
		coin, err := walletdb.GetFinalizableDutchAuctionCancelCoin(dTx, a.id, name)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("auction not found")
		}
		if err != nil {
			return nil, err
		}
		if a.rescanHeight < coin.Height+a.network.TransferLockup {
			return nil, errors.New("transfer is still locked up")
		}

		renewalBlock, err := a.client.GetRenewalBlock(a.network, a.rescanHeight)
		if err != nil {
			return nil, err
		}

		txb := new(TxBuilder)
		txb.AddCoin(coin.AsChain())
		txb.AddOutput(&chain.Output{
			Value:   coin.Value,
			Address: a.recvMgr.Address(),
			Covenant: chain.NewFinalizeCovenant(
				name,
				state.Info.Weak,
				renewalBlock.Hash(),
				state.Info.Height,
				state.Info.Claimed,
				state.Info.Renewals,
			),
		})

		prevTxDb, err := walletdb.GetTransactionByOutpoint(dTx, a.id, coin.Prevout.Hash)
		if err != nil {
			return nil, err
		}
		prevTx := new(chain.Transaction)
		if _, err := prevTx.ReadFrom(bytes.NewReader(prevTxDb.Raw)); err != nil {
			return nil, err
		}
		script := prevTx.Witnesses[coin.Prevout.Index].Items[1]
		wit := &chain.Witness{
			Items: [][]byte{
				script,
			},
		}
		txb.Witnesses = append(txb.Witnesses, wit)

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
		dba, err := walletdb.GetAddress(tx, a.id, addr)
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

	if dbAddr.Address.IsScriptHash() {
		return nil, errors.New("cannot sign messages with script hash addresses")
	}

	key, err := a.ring.PrivateKey(dbAddr.Derivation...)
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
		c, err := walletdb.GetCoinByPrevout(tx, a.id, &chain.Outpoint{
			Hash:  owner.Hash,
			Index: owner.Index,
		})
		if err != nil {
			return err
		}
		dbCoin = c
		return err
	})
	if err != nil {
		return nil, err
	}

	return a.SignMessage(dbCoin.Address, msg)
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
			return err
		}

		for resIdx, block := range blocks {
			if err := a.scanBlock(resIdx+i, block); err != nil {
				return err
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
			if a.addrBloom.Test(out.Address) {
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
			var shouldIndexTx bool
			coinbase := tx.Inputs[0].Prevout.Hash.IsZero()

			for inIdx, input := range tx.Inputs {
				if !a.outpointBloom.Test(input.Prevout) {
					continue
				}

				indexedInput, err := a.scanInput(dTx, tx, height, txIdx, inIdx)
				if err != nil {
					return err
				}
				if !indexedInput {
					continue
				}
				shouldIndexTx = true
				spends++
			}

			for outIdx, out := range tx.Outputs {
				if out.Covenant.Type == chain.CovenantTransfer {
					indexedTransfer, err := a.scanTransfer(dTx, tx, outIdx)
					if err != nil {
						return err
					}
					if !indexedTransfer {
						continue
					}
					shouldIndexTx = true
					coins++
					continue
				}

				if out.Covenant.Type == chain.CovenantFinalize {
					indexedFinalize, err := a.scanFinalize(dTx, tx, outIdx)
					if err != nil {
						return err
					}
					if !indexedFinalize {
						continue
					}
					shouldIndexTx = true
					coins++
					continue
				}

				if !a.addrBloom.Test(out.Address) {
					continue
				}

				if err := a.scanOutput(dTx, tx, height, txIdx, outIdx, coinbase); err != nil {
					return err
				}
				shouldIndexTx = true
				coins++
			}

			if !shouldIndexTx {
				continue
			}

			dbTx := &walletdb.Transaction{
				Hash:        tx.IDHex(),
				Idx:         txIdx,
				BlockHeight: height,
				BlockHash:   block.HashHex(),
				Raw:         tx.Bytes(),
				Time:        int(block.Time),
			}
			if _, err := walletdb.UpsertTransaction(dTx, a.id, dbTx); err != nil {
				return err
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

func (a *Account) scanInput(q walletdb.Transactor, tx *chain.Transaction, height, txIdx, inIdx int) (bool, error) {
	prevout := tx.Inputs[inIdx].Prevout
	coin, err := walletdb.GetCoinByPrevout(q, a.id, prevout)
	if errors.Is(err, sql.ErrNoRows) {
		a.lgr.Warning(
			"input bloom filter false positive",
			"height", height,
			"prevout", fmt.Sprintf("%s/%d", prevout.Hash, prevout.Index),
		)
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if coin.Spent {
		a.lgr.Info(
			"coin already spent",
			"height", height,
			"tx_idx", txIdx,
			"input_idx", inIdx,
			"addr", coin.Address,
			"outpoint_tx_hash", prevout.Hash.String(),
			"outpoint_idx", prevout.Index,
		)
		return false, nil
	}

	a.lgr.Info(
		"found spend",
		"height", height,
		"tx_idx", txIdx,
		"input_idx", inIdx,
		"addr", coin.Address.String(),
		"outpoint_tx_hash", prevout.Hash.String(),
		"outpoint_idx", prevout.Index,
	)

	err = walletdb.UpdateCoinSpent(
		q,
		coin.Prevout,
		tx.ID(),
	)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (a *Account) scanOutput(q walletdb.Transactor, tx *chain.Transaction, height, txIdx, outIdx int, coinbase bool) error {
	output := tx.Outputs[outIdx]
	addr := output.Address
	lookAddr, err := a.checkAddrInDB(q, addr)
	if err != nil {
		return errors.Wrap(err, "error getting addr from DB")
	}

	if lookAddr == nil {
		a.lgr.Warning(
			"addressBloom filter false positive",
			"addr", addr,
		)
		return nil
	}

	a.lgr.Info(
		"found coin",
		"height", height,
		"tx_idx", txIdx,
		"out_idx", outIdx,
		"addr", addr,
		"path", lookAddr.Derivation,
		"value", output.Value,
	)

	prevout := &chain.Outpoint{
		Hash:  tx.ID(),
		Index: uint32(outIdx),
	}
	err = walletdb.CreateCoin(
		q,
		a.id,
		prevout,
		output.Value,
		output.Address,
		output.Covenant,
		coinbase,
		walletdb.CoinTypeDefault,
	)
	if err != nil {
		return err
	}

	if err := a.scanCovenant(q, tx, output, outIdx); err != nil {
		return errors.Wrap(err, "error scanning covenants")
	}

	if err := a.updateOutputBloom(q, lookAddr, tx, outIdx); err != nil {
		return err
	}

	return nil
}

func (a *Account) scanCovenant(q walletdb.Transactor, tx *chain.Transaction, out *chain.Output, outIdx int) error {
	entry := &walletdb.NameHistory{
		AccountID: a.id,
		Outpoint: &chain.Outpoint{
			Hash:  tx.ID(),
			Index: uint32(outIdx),
		},
		Value: out.Value,
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
		entry.ParentTxHash = input.Prevout.Hash.String()
		entry.ParentOutIdx = input.Prevout.Index
	case chain.CovenantRedeem:
		input := tx.Inputs[outIdx]
		entry.NameHash = out.Covenant.Items[0]
		entry.Type = walletdb.NameActionRedeem
		entry.ParentTxHash = input.Prevout.Hash.String()
		entry.ParentOutIdx = input.Prevout.Index
	case chain.CovenantRegister:
		input := tx.Inputs[outIdx]
		entry.NameHash = out.Covenant.Items[0]
		entry.Type = walletdb.NameActionRegister
		entry.ParentTxHash = input.Prevout.Hash.String()
		entry.ParentOutIdx = input.Prevout.Index
		if err := walletdb.UpsertNameHash(q, a.id, entry.NameHash, walletdb.NameStatusOwned); err != nil {
			return err
		}
	case chain.CovenantRevoke:
		input := tx.Inputs[outIdx]
		entry.NameHash = out.Covenant.Items[0]
		entry.Type = walletdb.NameActionRevoke
		entry.ParentTxHash = input.Prevout.Hash.String()
		entry.ParentOutIdx = input.Prevout.Index
		if err := walletdb.UpsertNameHash(q, a.id, entry.NameHash, walletdb.NameStatusRevoked); err != nil {
			return err
		}
	case chain.CovenantUpdate:
		input := tx.Inputs[outIdx]
		entry.NameHash = out.Covenant.Items[0]
		entry.Type = walletdb.NameActionUpdate
		entry.ParentTxHash = input.Prevout.Hash.String()
		entry.ParentOutIdx = input.Prevout.Index
	default:
		return nil
	}

	return errors.Wrap(walletdb.UpdateNameHistory(q, entry), "error saving name history")
}

func (a *Account) scanTransfer(q walletdb.Transactor, tx *chain.Transaction, outIdx int) (bool, error) {
	output := tx.Outputs[outIdx]
	transfereeAddr := &chain.Address{
		Version: output.Covenant.Items[2][0],
		Hash:    output.Covenant.Items[3],
	}

	var xferDirection string
	var checkAddr *chain.Address
	if a.addrBloom.Test(output.Address) {
		xferDirection = "OUT"
		checkAddr = output.Address
	} else if a.addrBloom.Test(transfereeAddr) {
		xferDirection = "IN"
		checkAddr = transfereeAddr
	} else {
		return false, nil
	}

	matchAddr, err := a.checkAddrInDB(q, checkAddr)
	if err != nil {
		return false, err
	}
	if matchAddr == nil {
		a.lgr.Warning("bloom filter false positive")
		return false, nil
	}

	if xferDirection == "OUT" {
		return a.scanOutgoingTransfer(q, matchAddr, transfereeAddr, tx, outIdx)
	}

	return a.scanIncomingTransfer(q, tx, outIdx)
}

func (a *Account) scanOutgoingTransfer(q walletdb.Transactor, matchAddr *walletdb.Address, transfereeAddr *chain.Address, tx *chain.Transaction, outIdx int) (bool, error) {
	output := tx.Outputs[outIdx]
	input := tx.Inputs[outIdx]
	outpoint := &chain.Outpoint{
		Hash:  tx.ID(),
		Index: uint32(outIdx),
	}
	entry := &walletdb.NameHistory{
		AccountID:    a.id,
		NameHash:     output.Covenant.Items[0],
		Outpoint:     outpoint,
		Value:        output.Value,
		ParentTxHash: input.Prevout.Hash.String(),
		ParentOutIdx: input.Prevout.Index,
	}

	saveCoin := true
	var coinType walletdb.CoinType

	// transfer to a script address
	if transfereeAddr.IsScriptHash() {
		tfreeAddr, err := a.checkAddrInDB(q, transfereeAddr)
		if err != nil {
			return false, err
		}
		// transferring to one of our shakedex addresses
		if tfreeAddr != nil && tfreeAddr.Derivation[0] == shakedex.AddressBranch {
			entry.Type = walletdb.NameActionTransferDutchAuctionListing
			coinType = walletdb.CoinTypeDutchAuctionListing
		} else {
			entry.Type = walletdb.NameActionTransfer
		}
	} else {
		// transferring from one of our shakedex addresses
		if matchAddr.Derivation[0] == shakedex.AddressBranch {
			addr, err := a.checkAddrInDB(q, transfereeAddr)
			if err != nil {
				return false, err
			}
			// we don't know the recipient, so it's a shakedex fill
			if addr == nil {
				saveCoin = false
				entry.Type = walletdb.NameActionFillDutchAuction
				coinType = walletdb.CoinTypeDutchAuctionFill
			} else {
				// if we know the recipient, it's a cancel
				entry.Type = walletdb.NameActionTransferDutchAuctionCancel
				coinType = walletdb.CoinTypeDutchAuctionCancel
			}
		} else {
			entry.Type = walletdb.NameActionTransfer
		}
	}

	if err := a.updateOutputBloom(q, matchAddr, tx, outIdx); err != nil {
		return false, err
	}

	if saveCoin {
		err := walletdb.CreateCoin(
			q,
			a.id,
			outpoint,
			output.Value,
			output.Address,
			output.Covenant,
			false,
			coinType,
		)
		if err != nil {
			return false, err
		}
	}

	if entry.Type == walletdb.NameActionTransfer {
		if err := walletdb.UpsertNameHash(q, a.id, output.Covenant.Items[0], walletdb.NameStatusTransferring); err != nil {
			return false, err
		}
	}

	if err := walletdb.UpdateNameHistory(q, entry); err != nil {
		return false, err
	}
	return saveCoin, nil
}

func (a *Account) scanIncomingTransfer(q walletdb.Transactor, tx *chain.Transaction, outIdx int) (bool, error) {
	input := tx.Inputs[outIdx]
	output := tx.Outputs[outIdx]

	if !txscript.IsHIP1LockingScript(tx.Witnesses[outIdx]) {
		return false, nil
	}

	_, err := walletdb.GetCoinByPrevout(q, a.id, input.Prevout)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}

	// we only care about people transferring
	// in shakedex fills here

	name, err := a.client.GetNameByHash(output.Covenant.Items[0])
	if err != nil {
		return false, err
	}

	outpoint := &chain.Outpoint{
		Hash:  tx.ID(),
		Index: uint32(outIdx),
	}
	err = walletdb.CreateCoin(
		q,
		a.id,
		outpoint,
		output.Value,
		output.Address,
		output.Covenant,
		false,
		walletdb.CoinTypeDutchAuctionFill,
	)
	if err != nil {
		return false, err
	}

	entry := &walletdb.NameHistory{
		AccountID: a.id,
		Name:      name,
		NameHash:  output.Covenant.Items[0],
		Type:      walletdb.NameActionTransferFillDutchAuction,
		Outpoint:  outpoint,
	}
	if err := walletdb.UpdateNameHistory(q, entry); err != nil {
		return false, err
	}
	return true, err
}

func (a *Account) scanFinalize(q walletdb.Transactor, tx *chain.Transaction, outIdx int) (bool, error) {
	input := tx.Inputs[outIdx]
	output := tx.Outputs[outIdx]

	if a.addrBloom.Test(output.Address) {
		addr, err := a.checkAddrInDB(q, output.Address)
		if err != nil {
			return false, err
		}
		if addr == nil {
			return false, nil
		}
		return a.scanIncomingFinalize(q, tx, outIdx)
	}

	if !a.outpointBloom.Test(input.Prevout) {
		return false, nil
	}

	return false, a.scanOutgoingFinalize(q, tx, outIdx)
}

func (a *Account) scanIncomingFinalize(q walletdb.Transactor, tx *chain.Transaction, outIdx int) (bool, error) {
	input := tx.Inputs[outIdx]
	output := tx.Outputs[outIdx]
	outpoint := &chain.Outpoint{
		Hash:  tx.ID(),
		Index: uint32(outIdx),
	}
	entry := &walletdb.NameHistory{
		AccountID:    a.id,
		Name:         string(output.Covenant.Items[2]),
		NameHash:     output.Covenant.Items[0],
		Outpoint:     outpoint,
		Value:        output.Value,
		ParentTxHash: input.Prevout.Hash.String(),
		ParentOutIdx: input.Prevout.Index,
	}

	var coinType walletdb.CoinType
	if txscript.IsHIP1LockingScript(tx.Witnesses[outIdx]) {
		coin, err := walletdb.GetDutchAuctionCoin(q, a.id, input.Prevout)
		if err != nil {
			return false, err
		}
		_, err = walletdb.GetAddress(q, a.id, coin.Address)
		if errors.Is(err, sql.ErrNoRows) {
			coinType = walletdb.CoinTypeDutchAuctionFill
			entry.Type = walletdb.NameActionFinalizeFillDutchAuction
			if err := walletdb.UpsertName(q, a.id, entry.Name, walletdb.NameStatusOwned); err != nil {
				return false, nil
			}
		} else if err == nil {
			if output.Address.IsScriptHash() {
				coinType = walletdb.CoinTypeDutchAuctionListing
				entry.Type = walletdb.NameActionFinalizeDutchAuctionListing
			} else {
				coinType = walletdb.CoinTypeDutchAuctionCancel
				entry.Type = walletdb.NameActionFinalizeDutchAuctionCancel
			}
		} else {
			return false, err
		}
	} else {
		if output.Address.IsScriptHash() {
			entry.Type = walletdb.NameActionFinalizeDutchAuctionListing
			coinType = walletdb.CoinTypeDutchAuctionListing
		} else {
			addr, err := a.checkAddrInDB(q, output.Address)
			if err != nil {
				return false, err
			}
			if err := a.updateOutputBloom(q, addr, tx, outIdx); err != nil {
				return false, err
			}
			entry.Type = walletdb.NameActionFinalizeIn
			if err := walletdb.UpsertName(q, a.id, entry.Name, walletdb.NameStatusOwned); err != nil {
				return false, nil
			}
		}
	}

	err := walletdb.CreateCoin(
		q,
		a.id,
		outpoint,
		output.Value,
		output.Address,
		output.Covenant,
		false,
		coinType,
	)
	if err != nil {
		return false, err
	}

	if err := walletdb.UpdateNameHistory(q, entry); err != nil {
		return false, err
	}
	return true, err
}

func (a *Account) scanOutgoingFinalize(q walletdb.Transactor, tx *chain.Transaction, outIdx int) error {
	input := tx.Inputs[outIdx]
	output := tx.Outputs[outIdx]
	outpoint := &chain.Outpoint{
		Hash:  tx.ID(),
		Index: uint32(outIdx),
	}

	_, err := walletdb.GetCoinByPrevout(q, a.id, input.Prevout)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	name := string(output.Covenant.Items[2])
	if err := walletdb.UpsertName(q, a.id, name, walletdb.NameStatusTransferred); err != nil {
		return nil
	}

	entry := &walletdb.NameHistory{
		AccountID:    a.id,
		Type:         walletdb.NameActionFinalizeOut,
		Name:         name,
		Outpoint:     outpoint,
		Value:        output.Value,
		ParentTxHash: input.Prevout.Hash.String(),
		ParentOutIdx: input.Prevout.Index,
	}
	return walletdb.UpdateNameHistory(q, entry)
}

func (a *Account) checkAddrInDB(q walletdb.Querier, addr *chain.Address) (*walletdb.Address, error) {
	dbAddr, err := walletdb.GetAddress(q, a.id, addr)
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
	for _, bid := range bids {
		nonce := chain.GenerateNonce(a.ring.PublicEK(), name, bid.Coin.Address, bid.Value)
		txb.AddCoin(bid.Coin.AsChain())
		txb.AddOutput(&chain.Output{
			Value:   bid.Value,
			Address: bid.Coin.Address,
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

func (a *Account) sendRedeems(dTx walletdb.Transactor, coins []*chain.Coin, name string, nsHeight int, feeRate uint64) (*chain.Transaction, error) {
	var tx *chain.Transaction
	var err error
	txb := new(TxBuilder)
	for _, coin := range coins {
		txb.AddCoin(coin)
		txb.AddOutput(&chain.Output{
			Value:   coin.Value,
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

	coin, err := walletdb.GetCoinByPrevout(q, a.id, &chain.Outpoint{
		Hash:  state.Info.Owner.Hash,
		Index: state.Info.Owner.Index,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("you do not own this name")
	}
	if coin.Spent {
		return nil, errors.New("name is already registered")
	}

	txb := new(TxBuilder)
	txb.AddCoin(coin.AsChain())
	if hasName {
		txb.AddOutput(&chain.Output{
			Value:    coin.Value,
			Address:  coin.Address,
			Covenant: chain.NewUpdateCovenant(name, state.Info.Height, resource),
		})
	} else {
		renewalBlock, err := a.client.GetRenewalBlock(a.network, state.Info.Height)
		if err != nil {
			return nil, err
		}

		txb.AddOutput(&chain.Output{
			Value:    uint64(state.Info.Value),
			Address:  coin.Address,
			Covenant: chain.NewRegisterCovenant(name, state.Info.Height, renewalBlock.Hash(), resource),
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

	coin, err := walletdb.GetOwnedNameCoin(q, a.id, name)
	if err != nil {
		return nil, err
	}

	if a.rescanHeight < state.Info.Renewal+a.network.TreeInterval {
		return nil, errors.New("must wait to renew")
	}

	renewalBlock, err := a.client.GetRenewalBlock(a.network, state.Info.Height)
	if err != nil {
		return nil, err
	}

	txb := new(TxBuilder)
	txb.AddCoin(coin.AsChain())
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
		coins[i] = dbCoins[i].AsChain()
	}

	changeAddr := a.changeMgr.Address()
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
		if _, err := a.scanInput(dTx, tx, -1, 0, i); err != nil {
			return errors.Wrap(err, "error scanning input")
		}
	}
	for i, out := range tx.Outputs {
		if out.Covenant.Type == chain.CovenantTransfer {
			if _, err := a.scanTransfer(dTx, tx, i); err != nil {
				return err
			}
			continue
		}

		if out.Covenant.Type == chain.CovenantFinalize {
			if _, err := a.scanFinalize(dTx, tx, i); err != nil {
				return err
			}
			continue
		}

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

func (a *Account) updateOutputBloom(q walletdb.Transactor, addr *walletdb.Address, tx *chain.Transaction, outIdx int) error {
	var mgr *AddressManager
	switch addr.Derivation[0] {
	case chain.ReceiveBranch:
		mgr = a.recvMgr
	case chain.ChangeBranch:
		mgr = a.changeMgr
	case shakedex.AddressBranch:
		mgr = a.dutchMgr
	default:
		return errors.New("unknown address, should not happen")
	}

	if err := mgr.SetAddressIdx(q, addr.Derivation[1]); err != nil {
		return err
	}

	prevout := &chain.Outpoint{
		Hash:  tx.ID(),
		Index: uint32(outIdx),
	}
	outpointBloomCopy := a.outpointBloom.Copy()
	outpointBloomCopy.Add(prevout)

	if err := walletdb.UpdateOutpointBloom(q, a.id, outpointBloomCopy.Bytes()); err != nil {
		return errors.Wrap(err, "error updating account addressBloom filters")
	}

	a.outpointBloom = outpointBloomCopy
	return nil
}

func ValidateAccountID(id string) error {
	if len(id) == 0 {
		return errors.New("account ID cannot be empty")
	}

	if len(id) > 64 {
		return errors.New("account ID cannot be more than 64 characters")
	}

	if !accIDRegex.MatchString(id) {
		return errors.New("account ID can only contain letters, numbers, -, _, and . characters")
	}

	return nil
}

func init() {
	var err error
	accIDRegex, err = regexp.Compile("^[\\w\\d\\-_.]+$")
	if err != nil {
		panic(err)
	}
}
