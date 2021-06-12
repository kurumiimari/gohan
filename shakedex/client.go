package shakedex

import (
	"fmt"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/ghttp"
)

type Client struct {
	url string
}

type FeeInfo struct {
	RatePercent float64
	Address     *chain.Address
}

var NoFee = new(FeeInfo)

func NewClient(url string) *Client {
	return &Client{
		url: url,
	}
}

func (c *Client) FeeInfo() (*FeeInfo, error) {
	rawRes := struct {
		Rate float64 `json:"rate"`
		Addr string  `json:"addr"`
	}{}

	if err := c.doGet("api/v1/fee_info", &rawRes); err != nil {
		httpError, ok := err.(*ghttp.Error)
		if !ok {
			return nil, err
		}

		if httpError.StatusCode == 404 {
			return NoFee, nil
		}

		return nil, err
	}

	ratePercent := rawRes.Rate / 100
	addr, err := chain.NewAddressFromBech32(rawRes.Addr)
	if err != nil {
		return nil, err
	}
	return &FeeInfo{
		RatePercent: ratePercent,
		Address:     addr,
	}, nil
}

func (c *Client) UploadPresigns(auction *DutchAuction) error {
	return c.doPost("api/v1/auctions", auction.AsCanonical(), nil)
}

func (c *Client) doGet(path string, resObj interface{}) error {
	return ghttp.DefaultClient.DoGetJSON(fmt.Sprintf("%s/%s", c.url, path), resObj)
}

func (c *Client) doPost(path string, body interface{}, resObj interface{}) error {
	return ghttp.DefaultClient.DoPostJSON(fmt.Sprintf("%s/%s", c.url, path), body, resObj)
}
