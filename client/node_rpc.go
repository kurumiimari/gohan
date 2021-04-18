package client

import (
	"encoding/base64"
	"encoding/hex"
	"github.com/kurumiimari/gohan/chain"
	"github.com/pkg/errors"
	"github.com/ybbus/jsonrpc/v2"
)

type NodeRPCClient struct {
	client jsonrpc.RPCClient
}

type BatchRawBlockRes struct {
	Data  []byte
	Error error
}

type BatchNameInfoRes struct {
	Info  *NameInfoRes
	Error error
}

func NewNodeRPCClient(url string, apiKey string) *NodeRPCClient {
	var client jsonrpc.RPCClient
	if apiKey == "" {
		client = jsonrpc.NewClient(url)
	} else {
		client = jsonrpc.NewClientWithOpts(url, &jsonrpc.RPCClientOpts{
			CustomHeaders: map[string]string{
				"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte("x:"+apiKey)),
			},
		})
	}

	return &NodeRPCClient{
		client: client,
	}
}

func (c *NodeRPCClient) GetRawBlock(height int) ([]byte, error) {
	var blockHex string
	err := c.client.CallFor(&blockHex, "getblockbyheight", height, false, false)
	if err != nil {
		return nil, errors.Wrap(err, "error getting raw block")
	}
	return hex.DecodeString(blockHex)
}

func (c *NodeRPCClient) GetRawBlocksBatch(start, count int) ([]*BatchRawBlockRes, error) {
	var reqs jsonrpc.RPCRequests
	for i := start; i < start+count; i++ {
		reqs = append(reqs, &jsonrpc.RPCRequest{
			Method: "getblockbyheight",
			Params: jsonrpc.Params(i, false, false),
			ID:     i,
		})
	}
	batchRes, err := c.client.CallBatch(reqs)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	blocksRes := make([]*BatchRawBlockRes, len(reqs))
	for _, bRes := range batchRes {
		rpcErr := bRes.Error
		if rpcErr != nil {
			blocksRes[bRes.ID] = &BatchRawBlockRes{
				Error: rpcErr,
			}
			continue
		}

		blockHex, ok := bRes.Result.(string)
		if !ok {
			blocksRes[bRes.ID] = &BatchRawBlockRes{
				Error: err,
			}
			continue
		}
		data, err := hex.DecodeString(blockHex)
		if err != nil {
			blocksRes[bRes.ID] = &BatchRawBlockRes{
				Error: err,
			}
			continue
		}

		blocksRes[bRes.ID] = &BatchRawBlockRes{
			Data:  data,
			Error: nil,
		}
	}
	return blocksRes, nil
}

func (c *NodeRPCClient) GetRenewalBlock(network *chain.Network, height int) ([]byte, error) {
	height = height - network.RenewalMaturity
	if height < 0 {
		height = 0
	}
	return c.GetRawBlock(height)
}

func (c *NodeRPCClient) GetNameInfo(name string) (*NameInfoRes, error) {
	res := new(NameInfoRes)
	err := c.client.CallFor(res, "getnameinfo", name)
	return res, errors.Wrap(err, "error getting name info")
}

func (c *NodeRPCClient) BatchGetNameInfo(names []string) ([]*BatchNameInfoRes, error) {
	reqs := make(jsonrpc.RPCRequests, len(names))
	for i := 0; i < len(names); i++ {
		reqs[i] = &jsonrpc.RPCRequest{
			Method: "getnameinfo",
			Params: jsonrpc.Params(names[i]),
			ID:     i,
		}
	}
	batchRes, err := c.client.CallBatch(reqs)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	namesRes := make([]*BatchNameInfoRes, len(reqs))
	for _, nRes := range batchRes {
		rpcErr := nRes.Error
		if rpcErr != nil {
			namesRes[nRes.ID] = &BatchNameInfoRes{
				Error: rpcErr,
			}
			continue
		}

		infoRes := new(NameInfoRes)
		if err := nRes.GetObject(infoRes); err != nil {
			namesRes[nRes.ID] = &BatchNameInfoRes{
				Error: err,
			}
			continue
		}

		namesRes[nRes.ID] = &BatchNameInfoRes{
			Info: infoRes,
		}
	}
	return namesRes, nil
}

func (c *NodeRPCClient) GetInfo() (*InfoRes, error) {
	res := new(InfoRes)
	err := c.client.CallFor(res, "getinfo")
	return res, errors.Wrap(err, "error getting node info")
}

func (c *NodeRPCClient) SendRawTransaction(tx []byte) (string, error) {
	var hash string
	err := c.client.CallFor(&hash, "sendrawtransaction", hex.EncodeToString(tx))
	return hash, errors.Wrap(err, "error sending raw transaction")
}

func (c *NodeRPCClient) GetRawMempool() ([]string, error) {
	var entries []string
	err := c.client.CallFor(&entries, "getrawmempool")
	return entries, errors.Wrap(err, "error getting raw mempool")
}

func (c *NodeRPCClient) GenerateToAddress(n int, address string) error {
	_, err := c.client.Call("generatetoaddress", n, address)
	return errors.Wrap(err, "error generating to address")
}

func (c *NodeRPCClient) EstimateSmartFee(n int) (uint64, error) {
	var fee float64
	_, err := c.client.Call("estimatesmartfee", n)
	return uint64(fee * 1000000), errors.Wrap(err, "error estimating smart fee")
}
