package api

import (
	"encoding/json"
	"errors"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/gcrypto"
	"github.com/kurumiimari/gohan/wallet"
	"github.com/kurumiimari/gohan/walletdb"
)

type CreateAccountReq struct {
	ID       string `json:"id"`
	XPub     string `json:"xpub"`
	Mnemonic string `json:"mnemonic"`
	Password string `json:"password"`
	Index    uint32 `json:"index"`
}

type CreateAccountRes struct {
	ID        string  `json:"id"`
	Mnemonic  *string `json:"mnemonic"`
	WatchOnly bool
}

type UnlockReq struct {
	Password string `json:"password"`
}

type AccountAddressDepth struct {
	Receive uint32 `json:"receive"`
	Change  uint32 `json:"change"`
}

type AccountGetRes struct {
	ID             string               `json:"id"`
	Index          uint32               `json:"index"`
	Balances       *walletdb.Balances   `json:"balances"`
	AddressDepth   *AccountAddressDepth `json:"address_depth"`
	LookaheadDepth *AccountAddressDepth `json:"lookahead_depth"`
	ReceiveAddress string               `json:"receive_address"`
	ChangeAddress  string               `json:"change_address"`
	XPub           string               `json:"xpub"`
	RescanHeight   int                  `json:"rescan_height"`
}

type CoinsGetRes struct {
	Coins []*walletdb.Coin `json:"coins"`
}

type GetNamesRes struct {
	Names []*walletdb.Name `json:"names"`
}

type GetNameRes struct {
	History []*walletdb.RichNameHistoryEntry `json:"history"`
}

type GenAddressRes struct {
	Address    string `json:"address"`
	Derivation string `json:"derivation"`
}

type CreateOpenReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

type CreateBidReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	Value      uint64 `json:"value"`
	Lockup     uint64 `json:"lockup"`
	CreateOnly bool   `json:"create_only"`
}

type CreateRevealReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

type CreateRedeemReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

type CreateUpdateReq struct {
	Name       string          `json:"name"`
	Resource   *chain.Resource `json:"resource"`
	FeeRate    uint64          `json:"fee_rate"`
	CreateOnly bool            `json:"create_only"`
}

type CreateTransferReq struct {
	Name       string `json:"name"`
	Address    string `json:"address"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

type CreateFinalizeReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

type CreateRenewalsReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

type CreateRevokeReq struct {
	Name       string `json:"name"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

type CreateSendReq struct {
	Value      uint64 `json:"value"`
	Address    string `json:"address"`
	FeeRate    uint64 `json:"fee_rate"`
	CreateOnly bool   `json:"create_only"`
}

type RescanReq struct {
	Height int `json:"height"`
}

type SignMessageReq struct {
	Address string `json:"address"`
	Message string `json:"data"`
}

type SignMessageRes struct {
	Signature string `json:"signature"`
}

type SignMessageWithNameReq struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type UnspentBidsRes struct {
	UnspentBids []*wallet.UnspentBid `json:"unspent_bids"`
}

type UnspentRevealsRes struct {
	UnspentReveals []*wallet.UnspentReveal `json:"unspent_reveals"`
}

type UpdateDutchAuctionListingsReq struct {
	Name                  string         `json:"name"`
	FeeAddress            *chain.Address `json:"fee_address"`
	StartPrice            uint64         `json:"start_price"`
	EndPrice              uint64         `json:"end_price"`
	FeePercent            float64        `json:"auction_fee_rate"`
	NumDecrements         int            `json:"num_decrements"`
	DecrementDurationSecs int64          `json:"decrement_duration_secs"`
}

type DutchAuctionListingTransferReq struct {
	Name    string
	FeeRate uint64
}

type DutchAuctionListingFinalizeReq struct {
	Name    string
	FeeRate uint64
}

type DutchAuctionFillFinalizeReq struct {
	Name    string
	FeeRate uint64
}

type TransferDutchAuctionFillReq struct {
	Name             string         `json:"name"`
	LockScriptTxHash gcrypto.Hash   `json:"lock_script_tx_hash"`
	LockScriptOutIdx uint32         `json:"lock_script_out_idx"`
	PaymentAddress   *chain.Address `json:"payment_address"`
	FeeAddress       *chain.Address `json:"fee_address"`
	PublicKey        gcrypto.Hash   `json:"public_key"`
	Signature        gcrypto.Hash   `json:"signature"`
	LockTime         uint32         `json:"lock_time"`
	Bid              uint64         `json:"bid"`
	AuctionFee       uint64         `json:"fee"`
	FeeRate          uint64         `json:"fee_rate"`
}

type DutchAuctionCancelTransferReq struct {
	Name    string
	FeeRate uint64
}

type DutchAuctionCancelFinalizeReq struct {
	Name    string
	FeeRate uint64
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
