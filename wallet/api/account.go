package api

import (
	"encoding/base64"
	"github.com/gorilla/mux"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/gcrypto"
	"github.com/kurumiimari/gohan/wallet"
	"github.com/pkg/errors"
	"net/http"
)

func AccountParams(r *http.Request) string {
	params := mux.Vars(r)
	accountID := params["accountID"]
	return accountID
}

func (a *API) HandleAccountsGET(w http.ResponseWriter, r *http.Request) {
	accounts := a.node.Accounts()
	MarshalResponseJSON(w, &GetAccountsRes{
		Accounts: accounts,
	})
}

func (a *API) HandleAccountsPOST(w http.ResponseWriter, r *http.Request) {
	req := new(CreateAccountReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	var err error
	var mnemonic string
	if req.XPub != "" {
		_, err = a.node.ImportXPub(req.ID, req.Password, req.XPub, req.Index)
	} else if req.Mnemonic != "" {
		_, err = a.node.ImportMnemonic(req.ID, req.Password, req.Mnemonic, req.Index)
	} else {
		_, mnemonic, err = a.node.CreateWallet(req.ID, req.Password, req.Index)
	}

	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	res := &CreateAccountRes{
		ID:        req.ID,
		WatchOnly: req.XPub != "",
	}
	if mnemonic != "" {
		res.Mnemonic = &mnemonic
	}
	MarshalResponseJSON(w, res)
}

func (a *API) HandleAccountUnlockPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	req := new(UnlockReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	if err := acc.Unlock(req.Password); err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	w.WriteHeader(204)
}

func (a *API) HandleAccountLockPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	acc.Lock()
	w.WriteHeader(204)
}

type GetAccountsRes struct {
	Accounts []string `json:"accounts"`
}

func (a *API) HandleAccountGET(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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
		ID:       acc.ID(),
		Index:    acc.Index(),
		Balances: balances,
		AddressDepth: &AccountAddressDepth{
			recvDepth + 1,
			chgDepth + 1,
		},
		LookaheadDepth: &AccountAddressDepth{
			recvLook,
			chgLook,
		},
		ReceiveAddress: recvAddr.String(),
		ChangeAddress:  chgAddr.String(),
		XPub:           acc.XPub(),
		RescanHeight:   acc.RescanHeight(),
	}
	MarshalResponseJSON(w, res)
}

func (a *API) HandleAccountTransactionsGET(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleCoinsGET(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	coins, err := acc.Coins()
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}
	MarshalResponseJSON(w, &CoinsGetRes{
		Coins: coins,
	})
}

func (a *API) HandleNamesGET(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleNameGET(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleGenerateReceiveAddress(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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
		Address:    addr.String(),
		Derivation: chain.Derivation{0, idx}.String(),
	})
}

func (a *API) HandleGenerateChangeAddress(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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
		Address:    addr.String(),
		Derivation: chain.Derivation{1, idx}.String(),
	})
}

func (a *API) HandleAccountOpensPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleAccountBidsPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleAccountRevealsPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleAccountRedeemsPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleAccountUpdatesPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleAccountTransfersPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleAccountFinalizesPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleAccountRenewalsPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleAccountRevokesPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleAccountSendPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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
	acc, err := a.getAccount(r)
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

func (a *API) HandleRescanPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleSignMessagePOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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
		Signature: base64.StdEncoding.EncodeToString(chain.SerializeSignature(sig)),
	})
}

func (a *API) HandleSignMessageWithNamePOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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
		Signature: base64.StdEncoding.EncodeToString(chain.SerializeSignature(sig)),
	})
}

func (a *API) HandleUnspentBidsGET(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleUnspentRevealsGET(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
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

func (a *API) HandleDutchAuctionListingTransfersPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	req := new(DutchAuctionListingTransferReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.TransferDutchAuctionListing(
		req.Name,
		req.FeeRate,
	)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	MarshalResponseJSON(w, tx)
}

func (a *API) HandleDutchAuctionListingFinalizesPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	req := new(DutchAuctionListingFinalizeReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.FinalizeDutchAuctionListing(req.Name, req.FeeRate)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	MarshalResponseJSON(w, tx)
}

func (a *API) HandleUpdateDutchAuctionListingsPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	req := new(UpdateDutchAuctionListingsReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	res, err := acc.UpdateDutchAuctionListing(
		req.Name,
		req.StartPrice,
		req.EndPrice,
		req.FeeAddress,
		req.FeePercent,
		req.NumDecrements,
		req.DecrementDurationSecs,
	)
	if err != nil {
		MarshalErrorJSON(w, err, 500)
		return
	}

	MarshalResponseJSON(w, res)
}

func (a *API) HandleDutchAuctionFillTransfersPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	req := new(TransferDutchAuctionFillReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.FillDutchAuction(
		req.Name,
		&chain.Outpoint{
			Hash:  gcrypto.Hash(req.LockScriptTxHash),
			Index: req.LockScriptOutIdx,
		},
		req.PaymentAddress,
		req.FeeAddress,
		req.PublicKey,
		req.Signature,
		req.LockTime,
		req.Bid,
		req.AuctionFee,
		req.FeeRate,
	)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	MarshalResponseJSON(w, tx)
}

func (a *API) HandleDutchAuctionFillFinalizesPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	req := new(DutchAuctionFillFinalizeReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.FinalizeDutchAuction(req.Name, req.FeeRate)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	MarshalResponseJSON(w, tx)
}

func (a *API) HandleDutchAuctionCancelTransfersPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	req := new(DutchAuctionCancelTransferReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.TransferDutchAuctionCancel(req.Name, req.FeeRate)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	MarshalResponseJSON(w, tx)
}

func (a *API) HandleDutchAuctionCancelFinalizesPOST(w http.ResponseWriter, r *http.Request) {
	acc, err := a.getAccount(r)
	if err != nil {
		MarshalErrorJSON(w, err, 404)
		return
	}

	req := new(DutchAuctionCancelFinalizeReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	tx, err := acc.FinalizeDutchAuctionCancel(req.Name, req.FeeRate)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	MarshalResponseJSON(w, tx)
}

func (a *API) getAccount(r *http.Request) (*wallet.Account, error) {
	accountID := AccountParams(r)
	account, err := a.node.Account(accountID)
	if err != nil {
		return nil, err
	}
	return account, nil
}
