package api

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/wallet"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/pkg/errors"
	"net/http"
)

func AccountParams(r *http.Request) (string, string) {
	params := mux.Vars(r)
	walletID := params["walletID"]
	accountID := params["accountID"]
	return walletID, accountID
}

type AccountAddressDepth struct {
	Receive uint32 `json:"receive"`
	Change  uint32 `json:"change"`
}

type AccountGetRes struct {
	Name           string               `json:"name"`
	Index          uint32               `json:"index"`
	WalletName     string               `json:"wallet_name"`
	Balances       *walletdb.Balances   `json:"balances"`
	AddressDepth   *AccountAddressDepth `json:"address_depth"`
	LookaheadDepth *AccountAddressDepth `json:"lookahead_depth"`
	ReceiveAddress string               `json:"receive_address"`
	ChangeAddress  string               `json:"change_address"`
	XPub           string               `json:"xpub"`
	RescanHeight   int                  `json:"rescan_height"`
}

func (a *API) HandleAccountGET(w http.ResponseWriter, r *http.Request) {
	acc, wall, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	recvDepth, chgDepth := acc.AddressDepth()
	recvLook, chgLook := acc.LookaheadDepth()
	balances, err := acc.Balances()
	if err != nil {
		MarshalErrorJSON(w, errors.Wrap(err, "error getting balances"), 500)
		return
	}
	recvAddr := acc.ReceiveAddress()
	chgAddr := acc.ChangeAddress()

	// Add one to address depths
	// to match HSD
	res := &AccountGetRes{
		Name:       acc.Name(),
		Index:      acc.Index(),
		WalletName: wall.ID(),
		Balances:   balances,
		AddressDepth: &AccountAddressDepth{
			recvDepth + 1,
			chgDepth + 1,
		},
		LookaheadDepth: &AccountAddressDepth{
			recvLook + 1,
			chgLook + 1,
		},
		ReceiveAddress: recvAddr.String(a.network),
		ChangeAddress:  chgAddr.String(a.network),
		XPub:           acc.XPub(),
		RescanHeight:   acc.RescanHeight(),
	}
	MarshalResponseJSON(w, res)
}

func (a *API) HandleAccountTransactionsGET(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	q := r.URL.Query()
	count := GetIntFromQuery(q, "count", 50)
	offset := GetIntFromQuery(q, "offset", 0)

	txs, err := acc.Transactions(count, offset)
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}

	MarshalResponseJSON(w, txs)
}

type CoinsGetRes struct {
	Version  int    `json:"version"`
	Height   int    `json:"height"`
	Value    uint64 `json:"value"`
	Address  string `json:"address"`
	Covenant struct {
		Type   int      `json:"type"`
		Action string   `json:"action"`
		Items  []string `json:"items"`
	} `json:"covenant"`
	Coinbase bool   `json:"coinbase"`
	Hash     string `json:"hash"`
	Index    int    `json:"index"`
}

func (a *API) HandleCoinsGET(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	res := make([]*CoinsGetRes, 0)
	coins, err := acc.Coins()
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}
	for _, coin := range coins {
		covItems := make([]string, len(coin.CovenantItems))
		for i, item := range coin.CovenantItems {
			covItems[i] = hex.EncodeToString(item)
		}

		res = append(res, &CoinsGetRes{
			Version: 0,
			Height:  coin.BlockHeight,
			Value:   coin.Value,
			Address: coin.Address,
			Covenant: struct {
				Type   int      `json:"type"`
				Action string   `json:"action"`
				Items  []string `json:"items"`
			}{
				int(chain.NewCovenantTypeFromString(coin.CovenantType)),
				coin.CovenantType,
				covItems,
			},
			Coinbase: coin.Coinbase,
			Hash:     coin.TxHash,
			Index:    coin.OutIdx,
		})
	}
	MarshalResponseJSON(w, res)
}

type GetNamesRes struct {
	Names []*walletdb.Name `json:"names"`
}

func (a *API) HandleNamesGET(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	q := r.URL.Query()
	count := GetIntFromQuery(q, "count", 50)
	offset := GetIntFromQuery(q, "offset", 0)

	names, err := acc.Names(count, offset)
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}

	MarshalResponseJSON(w, &GetNamesRes{Names: names})
}

type GetNameRes struct {
	History []*walletdb.RichNameHistoryEntry `json:"history"`
}

func (a *API) HandleNameGET(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	params := mux.Vars(r)
	name := params["name"]
	q := r.URL.Query()
	count := GetIntFromQuery(q, "count", 50)
	offset := GetIntFromQuery(q, "offset", 0)

	history, err := acc.History(name, count, offset)
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}

	MarshalResponseJSON(w, &GetNameRes{History: history})
}

type GenAddressRes struct {
	Address    string `json:"address"`
	Derivation string `json:"derivation"`
}

func (a *API) HandleGenerateReceiveAddress(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	addr, idx, err := acc.GenerateReceiveAddress()
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}

	MarshalResponseJSON(w, &GenAddressRes{
		Address:    addr.String(a.network),
		Derivation: chain.Derivation{0, idx}.String(),
	})
}

func (a *API) HandleGenerateChangeAddress(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	addr, idx, err := acc.GenerateChangeAddress()
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}

	MarshalResponseJSON(w, &GenAddressRes{
		Address:    addr.String(a.network),
		Derivation: chain.Derivation{1, idx}.String(),
	})
}

type CreateOpenReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

func (a *API) HandleAccountOpensPOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	req := new(CreateOpenReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.Open(req.Name, req.FeeRate)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	MarshalResponseJSON(w, tx)
}

type CreateBidReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	Value      uint64 `json:"value"`
	Lockup     uint64 `json:"lockup"`
	CreateOnly bool   `json:"create_only"`
}

func (a *API) HandleAccountBidsPOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	req := new(CreateBidReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.Bid(req.Name, req.FeeRate, req.Value, req.Lockup)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	MarshalResponseJSON(w, tx)
}

type CreateRevealReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

type MultiRes struct {
	Txs  []*chain.Transaction
	Errs []error
}

func (m *MultiRes) MarshalJSON() ([]byte, error) {
	var errs []string
	for _, err := range m.Errs {
		errs = append(errs, err.Error())
	}

	return json.Marshal(struct {
		Txs  []*chain.Transaction `json:"txs"`
		Errs []string             `json:"errs"`
	}{
		Txs:  m.Txs,
		Errs: errs,
	})
}

func (m *MultiRes) UnmarshalJSON(bytes []byte) error {
	tmp := struct {
		Txs  []*chain.Transaction `json:"txs"`
		Errs []string             `json:"errs"`
	}{}

	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return err
	}

	errs := make([]error, len(tmp.Errs))
	for i, err := range tmp.Errs {
		errs[i] = errors.New(err)
	}

	m.Txs = tmp.Txs
	m.Errs = errs
	return nil
}

func (a *API) HandleAccountRevealsPOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	req := new(CreateRevealReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.Reveal(req.Name, req.FeeRate)
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}

	MarshalResponseJSON(w, tx)
}

type CreateRedeemReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

func (a *API) HandleAccountRedeemsPOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	req := new(CreateRedeemReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.Redeem(req.Name, req.FeeRate)
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}
	MarshalResponseJSON(w, tx)
}

type CreateUpdateReq struct {
	Name       string          `json:"name"`
	Resource   *chain.Resource `json:"resource"`
	FeeRate    uint64          `json:"fee_rate"`
	CreateOnly bool            `json:"create_only"`
}

func (a *API) HandleAccountUpdatesPOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	req := new(CreateUpdateReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.Update(req.Name, req.Resource, req.FeeRate)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}
	MarshalResponseJSON(w, tx)
}

type CreateTransferReq struct {
	Name       string `json:"name"`
	Address    string `json:"address"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

func (a *API) HandleAccountTransfersPOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	req := new(CreateTransferReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	addr, err := chain.NewAddressFromBech32(req.Address)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	tx, err := acc.Transfer(req.Name, addr, req.FeeRate)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}
	MarshalResponseJSON(w, tx)
}

type CreateFinalizeReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

func (a *API) HandleAccountFinalizesPOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	req := new(CreateFinalizeReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.Finalize(req.Name, req.FeeRate)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}
	MarshalResponseJSON(w, tx)
}

type CreateRenewalsReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

func (a *API) HandleAccountRenewalsPOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	req := new(CreateRenewalsReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.Renew(req.Name, req.FeeRate)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}
	MarshalResponseJSON(w, tx)
}

type CreateRevokeReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

func (a *API) HandleAccountRevokesPOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	req := new(CreateRevokeReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.Revoke(req.Name, req.FeeRate)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}
	MarshalResponseJSON(w, tx)
}

type CreateSendReq struct {
	Value      uint64 `json:"value"`
	Address    string `json:"address"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

func (a *API) HandleAccountSendPOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	req := new(CreateSendReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	addr, err := chain.NewAddressFromBech32(req.Address)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}
	tx, err := acc.Send(req.Value, req.FeeRate, addr)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}
	MarshalResponseJSON(w, tx)
}

func (a *API) HandleZapPost(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	if err := acc.Zap(); err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}
	w.WriteHeader(204)
}

type RescanReq struct {
	Height int `json:"height"`
}

func (a *API) HandleRescanPOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	req := new(RescanReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	if err := acc.Rescan(req.Height); err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}

	w.WriteHeader(204)
}

type SignMessageReq struct {
	Address string `json:"address"`
	Message string `json:"data"`
}

type SignMessageRes struct {
	Signature string `json:"signature"`
}

func (a *API) HandleSignMessagePOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	req := new(SignMessageReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	addr, err := chain.NewAddressFromBech32(req.Address)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	sig, err := acc.SignMessage(addr, []byte(req.Message))
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}

	MarshalResponseJSON(w, &SignMessageRes{
		Signature: base64.StdEncoding.EncodeToString(chain.SerializeRawSignature(sig)),
	})
}

type SignMessageWithNameReq struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

func (a *API) HandleSignMessageWithNamePOST(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	req := new(SignMessageWithNameReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	sig, err := acc.SignMessageWithName(req.Name, []byte(req.Message))
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}

	MarshalResponseJSON(w, &SignMessageRes{
		Signature: base64.StdEncoding.EncodeToString(chain.SerializeRawSignature(sig)),
	})
}

type UnspentBidsRes struct {
	UnspentBids []*wallet.UnspentBid `json:"unspent_bids"`
}

func (a *API) HandleUnspentBidsGET(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	q := r.URL.Query()
	count := GetIntFromQuery(q, "count", 50)
	offset := GetIntFromQuery(q, "offset", 0)

	bids, err := acc.UnspentBids(count, offset)
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}

	MarshalResponseJSON(w, &UnspentBidsRes{UnspentBids: bids})
}

type UnspentRevealsRes struct {
	UnspentReveals []*wallet.UnspentReveal `json:"unspent_reveals"`
}

func (a *API) HandleUnspentRevealsGET(w http.ResponseWriter, r *http.Request) {
	acc, _, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	q := r.URL.Query()
	count := GetIntFromQuery(q, "count", 50)
	offset := GetIntFromQuery(q, "offset", 0)

	revs, err := acc.UnspentReveals(count, offset)
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}

	MarshalResponseJSON(w, &UnspentRevealsRes{UnspentReveals: revs})
}

func (a *API) getAccount(r *http.Request) (*wallet.Account, *wallet.Wallet, error) {
	walletID, accountID := AccountParams(r)
	w, err := a.node.Wallet(walletID)
	if err != nil {
		return nil, nil, err
	}
	acc, err := w.Account(accountID)
	if err != nil {
		return nil, nil, err
	}
	return acc, w, nil
}
