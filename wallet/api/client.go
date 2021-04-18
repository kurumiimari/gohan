package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/wallet"
	"github.com/kurumiimari/gohan/walletdb"
	"io/ioutil"
	"net/http"
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

func (c *Client) Wallets() (*GetWalletsRes, error) {
	res := new(GetWalletsRes)
	err := c.doGet("api/v1/wallets", res)
	return res, err
}

func (c *Client) PollBlock() error {
	return c.doPost("api/v1/poll_block", nil, nil)
}

func (c *Client) CreateWallet(req *CreateWalletReq) (*CreateWalletRes, error) {
	res := new(CreateWalletRes)
	err := c.doPost("api/v1/wallets", req, res)
	return res, err
}

func (c *Client) Unlock(walletID string, password string) error {
	return c.doPost(fmt.Sprintf("api/v1/wallets/%s/login", walletID), &LockReq{
		Password: password,
	}, nil)
}

func (c *Client) Lock(walletID string) error {
	return c.doPost(fmt.Sprintf("api/v1/wallets/%s/login", walletID), nil, nil)
}

func (c *Client) GetAccounts(walletID string) (*GetWalletAccountsRes, error) {
	res := new(GetWalletAccountsRes)
	err := c.doGet(fmt.Sprintf("api/v1/wallets/%s/accounts", walletID), res)
	return res, err
}

func (c *Client) GetAccount(walletID string, accountID string) (*AccountGetRes, error) {
	res := new(AccountGetRes)
	err := c.doGet(fmt.Sprintf("api/v1/wallets/%s/accounts/%s", walletID, accountID), res)
	return res, err
}

func (c *Client) GetNames(walletID string, accountID string) (*GetNamesRes, error) {
	res := new(GetNamesRes)
	err := c.doGet(c.walletPath(walletID, accountID, "names"), res)
	return res, err
}

func (c *Client) GetName(walletID, accountID, name string) (*GetNameRes, error) {
	res := new(GetNameRes)
	err := c.doGet(c.walletPath(walletID, accountID, "names", name), res)
	return res, err
}

func (c *Client) GetAccountTransactions(walletID string, accountID string, count, offset int) ([]*walletdb.RichTransaction, error) {
	var res []*walletdb.RichTransaction
	err := c.doGet(c.walletPath(walletID, accountID, fmt.Sprintf("transactions?count=%d&offset=%d", count, offset)), &res)
	return res, err
}

func (c *Client) GenerateAccountReceiveAddress(walletID string, accountID string) (*GenAddressRes, error) {
	res := new(GenAddressRes)
	err := c.doPost(c.walletPath(walletID, accountID, "receive_address"), nil, res)
	return res, err
}

func (c *Client) GenerateAccountChangeAddress(walletID string, accountID string) (*GenAddressRes, error) {
	res := new(GenAddressRes)
	err := c.doPost(c.walletPath(walletID, accountID, "change_address"), nil, res)
	return res, err
}

func (c *Client) Send(walletID, accountID string, value, feeRate uint64, address string, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.walletPath(walletID, accountID, "sends"), &CreateSendReq{
		Value:      value,
		Address:    address,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Open(walletID, accountID, name string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.walletPath(walletID, accountID, "opens"), &CreateOpenReq{
		Name:       name,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Bid(walletID, accountID, name string, feeRate, value, lockup uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.walletPath(walletID, accountID, "bids"), &CreateBidReq{
		Name:       name,
		FeeRate:    feeRate,
		Value:      value,
		Lockup:     lockup,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Reveal(walletID, accountID, name string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.walletPath(walletID, accountID, "reveals"), &CreateRevealReq{
		Name:       name,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Redeem(walletID, accountID, name string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.walletPath(walletID, accountID, "redeems"), &CreateRedeemReq{
		Name:       name,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Update(walletID, accountID, name string, resource *chain.Resource, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.walletPath(walletID, accountID, "updates"), &CreateUpdateReq{
		Name:       name,
		Resource:   resource,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Transfer(walletID, accountID, name, address string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.walletPath(walletID, accountID, "transfers"), &CreateTransferReq{
		Name:       name,
		Address:    address,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Finalize(walletID, accountID, name string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.walletPath(walletID, accountID, "finalizes"), &CreateFinalizeReq{
		Name:       name,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Renew(walletID, accountID, name string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.walletPath(walletID, accountID, "renewals"), &CreateRenewalsReq{
		Name:       name,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Revoke(walletID, accountID, name string, feeRate uint64, createOnly bool) (*chain.Transaction, error) {
	res := new(chain.Transaction)
	err := c.doPost(c.walletPath(walletID, accountID, "revokes"), &CreateRevokeReq{
		Name:       name,
		FeeRate:    feeRate,
		CreateOnly: createOnly,
	}, res)
	return res, err
}

func (c *Client) Zap(walletID, accountID string) error {
	return c.doPost(c.walletPath(walletID, accountID, "zap"), nil, nil)
}

func (c *Client) Rescan(walletID, accountID string, height int) error {
	return c.doPost(c.walletPath(walletID, accountID, "rescan"), &RescanReq{Height: height}, nil)
}

func (c *Client) SignMessage(walletID, accountID, address, message string) (string, error) {
	res := new(SignMessageRes)
	err := c.doPost(c.walletPath(walletID, accountID, "sign_message"), &SignMessageReq{
		Address: address,
		Message: message,
	}, res)
	return res.Signature, err
}

func (c *Client) SignMessageWithName(walletID, accountID, name, message string) (string, error) {
	res := new(SignMessageRes)
	err := c.doPost(c.walletPath(walletID, accountID, "sign_message_with_name"), &SignMessageWithNameReq{
		Name:    name,
		Message: message,
	}, res)
	return res.Signature, err
}

func (c *Client) UnspentBids(walletID, accountID string, count, offset int) (*UnspentBidsRes, error) {
	res := new(UnspentBidsRes)
	err := c.doGet(
		c.walletPath(walletID, accountID, c.QueryStringPath("unspent_bids", PaginationQuery(count, offset))),
		res,
	)
	return res, err
}

func (c *Client) UnspentReveals(walletID, accountID string, count, offset int) (*UnspentRevealsRes, error) {
	res := new(UnspentRevealsRes)
	err := c.doGet(
		c.walletPath(walletID, accountID, c.QueryStringPath("unspent_reveals", PaginationQuery(count, offset))),
		res,
	)
	return res, err
}

func (c *Client) doGet(path string, resObj interface{}) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", c.url, path), nil)
	if err != nil {
		return err
	}
	return c.doReq(req, resObj)
}

func (c *Client) doPost(path string, body interface{}, resObj interface{}) error {
	var bodyB []byte
	if body != nil {
		bodyJ, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyB = bodyJ
	}

	bodyR := bytes.NewReader(bodyB)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.url, path), bodyR)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doReq(req, resObj)
}

func (c *Client) doReq(req *http.Request, resObj interface{}) error {
	client := &http.Client{}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	resB, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode > 300 {
		return fmt.Errorf("http error %d: %s", res.StatusCode, resB)
	}

	if res.StatusCode == 204 {
		return nil
	}

	if err := json.Unmarshal(resB, resObj); err != nil {
		return err
	}
	return nil
}

func (c *Client) walletPath(walletID, accountID string, suffixes ...string) string {
	return fmt.Sprintf("api/v1/wallets/%s/accounts/%s/%s", walletID, accountID, strings.Join(suffixes, "/"))
}

func (c *Client) QueryStringPath(name string, q url.Values) string {
	return fmt.Sprintf("%s?%s", name, q.Encode())
}
