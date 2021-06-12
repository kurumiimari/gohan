package api

import (
	"fmt"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/ghttp"
	"github.com/kurumiimari/gohan/shakedex"
	"github.com/kurumiimari/gohan/wallet"
	"github.com/kurumiimari/gohan/walletdb"
	"net/url"
	"strings"
)

type Client struct {
	url    string
	apiKey string
}

func NewClient(url string, apiKey string) *Client {
	return &Client{
		url:    url,
		apiKey: apiKey,
	}
}

func (c *Client) Status() (*wallet.NodeStatus, error) {
	res := new(wallet.NodeStatus)
	err := c.doGet("api/v1/status", res)
	return res, err
}

func (c *Client) PollBlock() error {
	return c.doPost("api/v1/poll_block", nil, nil)
}

func (c *Client) CreateAccount(req *CreateAccountReq) (*CreateAccountRes, error) {
	res := new(CreateAccountRes)
	err := c.doPost("api/v1/accounts", req, res)
	return res, err
}

func (c *Client) Unlock(accountID string, password string) error {
	return c.doPost(c.accountPath(accountID, "unlock"), &UnlockReq{
		Password: password,
	}, nil)
}

func (c *Client) Lock(accountID string) error {
	return c.doPost(c.accountPath(accountID, "lock"), nil, nil)
}

func (c *Client) GetAccounts() (*GetAccountsRes, error) {
	res := new(GetAccountsRes)
	err := c.doGet("api/v1/accounts", res)
	return res, err
}

func (c *Client) GetAccount(accountID string) (*AccountGetRes, error) {
	res := new(AccountGetRes)
	err := c.doGet(c.accountPath(accountID), res)
	return res, err
}

func (c *Client) GetNames(accountID string) (*GetNamesRes, error) {
	res := new(GetNamesRes)
	err := c.doGet(c.accountPath(accountID, "names"), res)
	return res, err
}

func (c *Client) GetName(accountID, name string) (*GetNameRes, error) {
	res := new(GetNameRes)
	err := c.doGet(c.accountPath(accountID, "names", name), res)
	return res, err
}

func (c *Client) GetAccountTransactions(accountID string, count, offset int) ([]*walletdb.RichTransaction, error) {
	var res []*walletdb.RichTransaction
	err := c.doGet(c.accountPath(accountID, fmt.Sprintf("transactions?count=%d&offset=%d", count, offset)), &res)
	return res, err
}

func (c *Client) GenerateAccountReceiveAddress(accountID string) (*GenAddressRes, error) {
	res := new(GenAddressRes)
	err := c.doPost(c.accountPath(accountID, "receive_address"), nil, res)
	return res, err
}

func (c *Client) GenerateAccountChangeAddress(accountID string) (*GenAddressRes, error) {
	res := new(GenAddressRes)
	err := c.doPost(c.accountPath(accountID, "change_address"), nil, res)
	return res, err
}

func (c *Client) Send(accountID string, value, feeRate uint64, address string, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "sends"), &CreateSendReq{
		Value:      value,
		Address:    address,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Open(accountID, name string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "opens"), &CreateOpenReq{
		Name:       name,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Bid(accountID, name string, feeRate, value, lockup uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "bids"), &CreateBidReq{
		Name:       name,
		FeeRate:    feeRate,
		Value:      value,
		Lockup:     lockup,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Reveal(accountID, name string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "reveals"), &CreateRevealReq{
		Name:       name,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Redeem(accountID, name string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "redeems"), &CreateRedeemReq{
		Name:       name,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Update(accountID, name string, resource *chain.Resource, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "updates"), &CreateUpdateReq{
		Name:       name,
		Resource:   resource,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Transfer(accountID, name, address string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "transfers"), &CreateTransferReq{
		Name:       name,
		Address:    address,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Finalize(accountID, name string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "finalizes"), &CreateFinalizeReq{
		Name:       name,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Renew(accountID, name string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "renewals"), &CreateRenewalsReq{
		Name:       name,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Revoke(accountID, name string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "revokes"), &CreateRevokeReq{
		Name:       name,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) TransferDutchAuctionListing(accountID, name string, feeRate uint64,
) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "dutch_auction_listing_transfers"), &DutchAuctionListingTransferReq{
		Name:    name,
		FeeRate: feeRate,
	}, res)
	return res, err
}

func (c *Client) FinalizeDutchAuctionListing(accountID, name string, feeRate uint64) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "dutch_auction_listing_finalizes"), &DutchAuctionListingFinalizeReq{
		Name:    name,
		FeeRate: feeRate,
	}, res)
	return res, err
}

func (c *Client) TransferDutchAuctionCancel(accountID, name string, feeRate uint64) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "dutch_auction_cancel_transfers"), &DutchAuctionCancelTransferReq{
		Name:    name,
		FeeRate: feeRate,
	}, res)
	return res, err
}

func (c *Client) FinalizeDutchAuctionCancel(accountID, name string, feeRate uint64) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "dutch_auction_cancel_finalizes"), &DutchAuctionCancelFinalizeReq{
		Name:    name,
		FeeRate: feeRate,
	}, res)
	return res, err
}

func (c *Client) UpdateDutchAuctionListing(accountID string, req *UpdateDutchAuctionListingsReq) (*shakedex.DutchAuction, error) {
	res := new(shakedex.DutchAuction)
	err := c.doPost(c.accountPath(accountID, "dutch_auction_listings"), req, res)
	return res, err
}

func (c *Client) TransferDutchAuctionFill(accountID string, req *TransferDutchAuctionFillReq) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "dutch_auction_fill_transfers"), req, res)
	return res, err
}

func (c *Client) FinalizeDutchAuctionFill(accountID string, name string, feeRate uint64) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.accountPath(accountID, "dutch_auction_fill_finalizes"), &DutchAuctionFillFinalizeReq{
		Name:    name,
		FeeRate: feeRate,
	}, res)
	return res, err
}

func (c *Client) Zap(accountID string) error {
	return c.doPost(c.accountPath(accountID, "zap"), nil, nil)
}

func (c *Client) Rescan(accountID string, height int) error {
	return c.doPost(c.accountPath(accountID, "rescan"), &RescanReq{Height: height}, nil)
}

func (c *Client) SignMessage(accountID, address, message string) (string, error) {
	res := new(SignMessageRes)
	err := c.doPost(c.accountPath(accountID, "sign_message"), &SignMessageReq{
		Address: address,
		Message: message,
	}, res)
	return res.Signature, err
}

func (c *Client) SignMessageWithName(accountID, name, message string) (string, error) {
	res := new(SignMessageRes)
	err := c.doPost(c.accountPath(accountID, "sign_message_with_name"), &SignMessageWithNameReq{
		Name:    name,
		Message: message,
	}, res)
	return res.Signature, err
}

func (c *Client) UnspentBids(accountID string, count, offset int) (*UnspentBidsRes, error) {
	res := new(UnspentBidsRes)
	err := c.doGet(
		c.accountPath(accountID, c.QueryStringPath("unspent_bids", PaginationQuery(count, offset))),
		res,
	)
	return res, err
}

func (c *Client) UnspentReveals(accountID string, count, offset int) (*UnspentRevealsRes, error) {
	res := new(UnspentRevealsRes)
	err := c.doGet(
		c.accountPath(accountID, c.QueryStringPath("unspent_reveals", PaginationQuery(count, offset))),
		res,
	)
	return res, err
}

func (c *Client) doGet(path string, resObj interface{}) error {
	return ghttp.DefaultClient.DoGetJSON(fmt.Sprintf("%s/%s", c.url, path), resObj)
}

func (c *Client) doPost(path string, reqObj interface{}, resObj interface{}) error {
	return ghttp.DefaultClient.DoPostJSON(fmt.Sprintf("%s/%s", c.url, path), reqObj, resObj)
}

func (c *Client) accountPath(accountID string, suffixes ...string) string {
	return fmt.Sprintf("api/v1/accounts/%s/%s", accountID, strings.Join(suffixes, "/"))
}

func (c *Client) QueryStringPath(name string, q url.Values) string {
	return fmt.Sprintf("%s?%s", name, q.Encode())
}
