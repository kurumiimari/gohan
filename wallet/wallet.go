package wallet

import (
	"bytes"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/client"
)

//var walletLogger = log.ModuleLogger("wallet")
//
//type Wallet struct {
//	tmb       *tomb.Tomb
//	network   *chain.Network
//	engine    *walletdb.Engine
//	client    *client.NodeRPCClient
//	bm        *BlockMonitor
//	keyLocker *KeyLocker
//	id        string
//	accounts  *sync.Map
//	watchOnly bool
//}
//
//func NewWallet(
//	tmb *tomb.Tomb,
//	network *chain.Network,
//	engine *walletdb.Engine,
//	client *client.NodeRPCClient,
//	bm *BlockMonitor,
//	keyLocker *KeyLocker,
//	id string,
//) *Wallet {
//	return &Wallet{
//		tmb:       tmb,
//		network:   network,
//		engine:    engine,
//		client:    client,
//		bm:        bm,
//		keyLocker: keyLocker,
//		accounts:  new(sync.Map),
//		id:        id,
//	}
//}
//
//func (w *Wallet) ID() string {
//	return w.id
//}
//
//func (w *Wallet) Start() error {
//	walletLogger.Info("opening wallet", "id", w.id)
//
//	var accounts []*walletdb.AccountOpts
//	err := w.engine.Transaction(func(tx walletdb.Transactor) error {
//		a, err := walletdb.GetAllAccounts(tx, w.id)
//		if err != nil {
//			return err
//		}
//		accounts = a
//		return nil
//	})
//	if err != nil {
//		return errors.WithStack(err)
//	}
//
//	for _, dbAcc := range accounts {
//		xpub, err := chain.NewMasterExtendedKeyFromXPub(dbAcc.XPub, w.network)
//		if err != nil {
//			panic(err)
//		}
//		ring := NewAccountKeyring(
//			w.keyLocker,
//			xpub,
//			w.network,
//			dbAcc.Idx,
//		)
//		acc := NewAccount(w.tmb, w.network, w.engine, w.client, w.bm, ring, dbAcc)
//		if err := acc.Start(); err != nil {
//			return errors.Wrap(err, "error starting account")
//		}
//		w.accounts.Store(dbAcc.ID, acc)
//	}
//
//	return nil
//}
//
//func (w *Wallet) Unlock(password string) error {
//	err := w.keyLocker.Unlock(password)
//	if err != nil {
//		walletLogger.Warning("unlock attempt failed", "wid", w.id)
//		return err
//	}
//	walletLogger.Info("wallet unlocked", "wid", w.id)
//	return nil
//}
//
//func (w *Wallet) Lock() error {
//	if !w.keyLocker.Locked() {
//		return errors.New("wallet is already locked")
//	}
//	w.keyLocker.Lock()
//	walletLogger.Info("wallet locked", "wid", w.id)
//	return nil
//}
//
//func (w *Wallet) Account(accountID string) (*Account, error) {
//	accIface, ok := w.accounts.Load(accountID)
//	if !ok {
//		return nil, fmt.Errorf("account %s not found", accountID)
//	}
//	return accIface.(*Account), nil
//}
//
//func (w *Wallet) Accounts() []string {
//	var accountIds []string
//	w.accounts.Range(func(key, value interface{}) bool {
//		accountIds = append(accountIds, key.(string))
//		return true
//	})
//	return accountIds
//}

func GetRawBlocksConcurrently(client *client.NodeRPCClient, start, count int) ([]*chain.Block, error) {
	results, err := client.GetRawBlocksBatch(start, count)
	if err != nil {
		return nil, err
	}

	var blocks []*chain.Block
	for _, res := range results {
		if res.Error != nil {
			return nil, err
		}
		block := new(chain.Block)
		if _, err := block.ReadFrom(bytes.NewReader(res.Data)); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}
