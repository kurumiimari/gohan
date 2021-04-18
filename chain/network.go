package chain

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/pkg/errors"
)

const (
	CoinPurpose = 44
)

type Network struct {
	Net              wire.BitcoinNet
	Name             string
	WalletPort       int
	NodePort         int
	AddressHRP       string
	HasReserved      bool
	ClaimPeriod      int
	CoinbaseMaturity int
	RenewalMaturity  int
	RolloutInterval  int
	AuctionStart     int
	TreeInterval     int
	BiddingPeriod    int
	RevealPeriod     int
	TransferLockup   int
	KeyPrefix        *NetworkKeyPrefix

	chainParams *chaincfg.Params
}

type NetworkKeyPrefix struct {
	Private  uint8
	XPub     [4]byte
	XPriv    [4]byte
	CoinType uint32
}

var NetworkMain = &Network{
	Net:              1533997779,
	Name:             "main",
	WalletPort:       12039,
	NodePort:         12037,
	AddressHRP:       "hs",
	HasReserved:      true,
	ClaimPeriod:      210240,
	CoinbaseMaturity: 100,
	RenewalMaturity:  4320,
	RolloutInterval:  1008,
	AuctionStart:     2016,
	TreeInterval:     36,
	BiddingPeriod:    720,
	RevealPeriod:     1440,
	TransferLockup:   288,
	KeyPrefix: &NetworkKeyPrefix{
		Private:  0x80,
		XPub:     [4]byte{0x04, 0x88, 0xb2, 0x1e},
		XPriv:    [4]byte{0x04, 0x88, 0xad, 0xe4},
		CoinType: 5353,
	},
}

var NetworkRegtest = &Network{
	Net:              2922943951,
	Name:             "regtest",
	WalletPort:       14039,
	NodePort:         14037,
	AddressHRP:       "rs",
	HasReserved:      false,
	ClaimPeriod:      250000,
	CoinbaseMaturity: 2,
	RenewalMaturity:  50,
	RolloutInterval:  2,
	AuctionStart:     0,
	TreeInterval:     5,
	BiddingPeriod:    5,
	RevealPeriod:     10,
	TransferLockup:   10,
	KeyPrefix: &NetworkKeyPrefix{
		Private:  0x5a,
		XPub:     [4]byte{0xea, 0xb4, 0xfa, 0x05},
		XPriv:    [4]byte{0xea, 0xb4, 0x04, 0xc7},
		CoinType: 5355,
	},
}

func NetworkFromName(name string) (*Network, error) {
	switch name {
	case NetworkMain.Name:
		return NetworkMain, nil
	case NetworkRegtest.Name:
		return NetworkRegtest, nil
	default:
		return nil, errors.New("invalid network")
	}
}

func (n *Network) ChainParams() *chaincfg.Params {
	if n.chainParams != nil {
		return n.chainParams
	}

	params := &chaincfg.Params{
		Net:            n.Net,
		Name:           n.Name + "-hns",
		PrivateKeyID:   n.KeyPrefix.Private,
		HDPrivateKeyID: n.KeyPrefix.XPriv,
		HDPublicKeyID:  n.KeyPrefix.XPub,
		HDCoinType:     n.KeyPrefix.CoinType,
	}
	n.chainParams = params

	return n.chainParams
}

func init() {
	chaincfg.Register(NetworkMain.ChainParams())
	chaincfg.Register(NetworkRegtest.ChainParams())
}
