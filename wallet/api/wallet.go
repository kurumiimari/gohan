package api

import (
	"github.com/gorilla/mux"
	"net/http"
)

func WalletParams(r *http.Request) string {
	params := mux.Vars(r)
	walletID := params["walletID"]
	return walletID
}

type UnlockReq struct {
	Password string `json:"password"`
}

func (a *API) HandleWalletUnlockPOST(w http.ResponseWriter, r *http.Request) {
	walletID := WalletParams(r)
	req := new(UnlockReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	wallet, err := a.node.Wallet(walletID)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	if err := wallet.Unlock(req.Password); err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	w.WriteHeader(204)
}

func (a *API) HandleWalletLockPOST(w http.ResponseWriter, r *http.Request) {
	walletID := WalletParams(r)
	wallet, err := a.node.Wallet(walletID)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	if err := wallet.Lock(); err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	w.WriteHeader(204)
}

type GetWalletAccountsRes struct {
	Accounts []string `json:"accounts"`
}

func (a *API) HandleAccountsGET(w http.ResponseWriter, r *http.Request) {
	walletID := WalletParams(r)
	wallet, err := a.node.Wallet(walletID)
	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	MarshalResponseJSON(w, &GetWalletAccountsRes{Accounts: wallet.Accounts()})
}

type CreateWalletReq struct {
	Name     string `json:"name"`
	XPub     string `json:"xpub"`
	Mnemonic string `json:"mnemonic"`
	Password string `json:"password"`
}

type CreateWalletRes struct {
	Name      string  `json:"name"`
	Mnemonic  *string `json:"mnemonic"`
	WatchOnly bool
}

func (a *API) HandleWalletCreate(w http.ResponseWriter, r *http.Request) {
	req := new(CreateWalletReq)
	if !UnmarshalRequestJSON(w, r, req) {
		return
	}

	var err error
	var mnemonic string
	if req.XPub != "" {
		_, err = a.node.ImportXPub(req.Name, req.Password, req.XPub)
	} else if req.Mnemonic != "" {
		_, err = a.node.ImportMnemonic(req.Name, req.Password, req.Mnemonic)
	} else {
		_, mnemonic, err = a.node.CreateWallet(req.Name, req.Password)
	}

	if err != nil {
		MarshalErrorJSON(w, err, 400)
		return
	}

	res := &CreateWalletRes{
		Name:      req.Name,
		WatchOnly: req.XPub != "",
	}
	if mnemonic != "" {
		res.Mnemonic = &mnemonic
	}
	MarshalResponseJSON(w, res)
}

type GetWalletsRes struct {
	Wallets []string `json:"wallets"`
}

func (a *API) HandleWalletsGET(w http.ResponseWriter, r *http.Request) {
	wallets := a.node.Wallets()
	MarshalResponseJSON(w, &GetWalletsRes{Wallets: wallets})
}
