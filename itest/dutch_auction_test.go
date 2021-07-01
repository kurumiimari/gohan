package itest

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/shakedex"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type DutchAuctionSuite struct {
	suite.Suite
	hsd       *HSD
	client    *api.Client
	cleanup   func()
	name      string
	aliceInfo *api.AccountGetRes
	bobInfo   *api.AccountGetRes
	auction   *shakedex.DutchAuction
}

func (s *DutchAuctionSuite) SetupTest() {
	t := s.T()
	s.name = "awilauh"
	s.hsd = startHSD()
	s.client, s.cleanup = startDaemon(t)

	_, err := s.client.CreateAccount(&api.CreateAccountReq{
		ID:       "alice",
		Mnemonic: Mnemonic,
		Password: "password",
	})
	require.NoError(t, err)

	_, err = s.client.CreateAccount(&api.CreateAccountReq{
		ID:       "bob",
		Password: "password",
	})
	require.NoError(t, err)

	s.aliceInfo, err = s.client.GetAccount("alice")
	require.NoError(t, err)
	s.bobInfo, err = s.client.GetAccount("bob")
	require.NoError(t, err)

	err = s.client.Unlock("alice", "password")
	require.NoError(t, err)
	err = s.client.Unlock("bob", "password")
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, s.aliceInfo.ReceiveAddress)
	mineTo(t, s.hsd.Client, s.client, 1, s.bobInfo.ReceiveAddress)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.CoinbaseMaturity-1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 3)

	s.doAuction()
}

func (s *DutchAuctionSuite) TearDownTest() {
	s.cleanup()
	s.hsd.Stop()
}

func (s *DutchAuctionSuite) doAuction() {
	t := s.T()

	_, err := s.client.Open("alice", s.name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TreeInterval+2, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 10)

	_, err = s.client.Bid("alice", s.name, 100, 1000000, 2000000, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.BiddingPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 15)

	_, err = s.client.Reveal("alice", s.name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.RevealPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 15+chain.NetworkRegtest.RevealPeriod)

	_, err = s.client.Update("alice", s.name, nil, 100, false)
	require.NoError(t, err)
	mineTo(t, s.hsd.Client, s.client, 1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 16+chain.NetworkRegtest.RevealPeriod)
}

func (s *DutchAuctionSuite) transferListing() {
	t := s.T()

	_, err := s.client.TransferDutchAuctionListing("alice", s.name, 100)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 17+chain.NetworkRegtest.RevealPeriod)
}

func (s *DutchAuctionSuite) finalizeListing() {
	t := s.T()

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TransferLockup, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 17+chain.NetworkRegtest.RevealPeriod+chain.NetworkRegtest.TransferLockup)

	_, err := s.client.FinalizeDutchAuctionListing("alice", s.name, 100)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 18+chain.NetworkRegtest.RevealPeriod+chain.NetworkRegtest.TransferLockup)
}

func (s *DutchAuctionSuite) generate() {
	t := s.T()
	var err error
	s.auction, err = s.client.UpdateDutchAuctionListing("alice", &api.UpdateDutchAuctionListingsReq{
		Name:                  s.name,
		FeeAddress:            nil,
		StartPrice:            100,
		EndPrice:              50,
		FeePercent:            0,
		NumDecrements:         1,
		DecrementDurationSecs: 60,
	})
	require.NoError(t, err)
}

func (s *DutchAuctionSuite) fill() {
	t := s.T()

	_, err := s.client.TransferDutchAuctionFill("bob", &api.TransferDutchAuctionFillReq{
		Name:             s.name,
		LockScriptTxHash: s.auction.LockingOutpoint.Hash,
		LockScriptOutIdx: s.auction.LockingOutpoint.Index,
		PaymentAddress:   s.auction.PaymentAddress,
		FeeAddress:       s.auction.FeeAddress,
		PublicKey:        s.auction.PublicKey,
		Signature:        s.auction.Bids[0].Signature,
		LockTime:         s.auction.Bids[0].LockTime,
		Bid:              s.auction.Bids[0].Value,
		AuctionFee:       s.auction.Bids[0].Fee,
		FeeRate:          100,
	})
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1+chain.NetworkRegtest.TransferLockup, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 19+chain.NetworkRegtest.RevealPeriod+chain.NetworkRegtest.TransferLockup*2)
	awaitHeight(t, s.client, "bob", 19+chain.NetworkRegtest.RevealPeriod+chain.NetworkRegtest.TransferLockup*2)
}

func (s *DutchAuctionSuite) finalize() {
	t := s.T()

	_, err := s.client.FinalizeDutchAuctionFill("bob", s.name, 100)
	require.NoError(t, err)
}

func (s *DutchAuctionSuite) cancelTransfer() {
	t := s.T()

	_, err := s.client.TransferDutchAuctionCancel("alice", s.name, 100)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1+chain.NetworkRegtest.TransferLockup, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 19+chain.NetworkRegtest.RevealPeriod+chain.NetworkRegtest.TransferLockup*2)
	awaitHeight(t, s.client, "bob", 19+chain.NetworkRegtest.RevealPeriod+chain.NetworkRegtest.TransferLockup*2)
}

func (s *DutchAuctionSuite) cancelFinalize() {
	t := s.T()

	_, err := s.client.FinalizeDutchAuctionCancel("alice", s.name, 100)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1+chain.NetworkRegtest.TransferLockup, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 30+chain.NetworkRegtest.RevealPeriod+chain.NetworkRegtest.TransferLockup*2)
	awaitHeight(t, s.client, "bob", 30+chain.NetworkRegtest.RevealPeriod+chain.NetworkRegtest.TransferLockup*2)
}

func (s *DutchAuctionSuite) TestTransferDutchAuctionListing() {
	t := s.T()
	s.transferListing()
	hist, err := s.client.GetName("alice", s.name)
	require.NoError(t, err)
	require.Equal(t, walletdb.NameActionTransferDutchAuctionListing, hist.History[0].Type)
}

func (s *DutchAuctionSuite) TestFinalizeDutchAuctionListing() {
	t := s.T()

	s.transferListing()
	s.finalizeListing()

	hist, err := s.client.GetName("alice", s.name)
	require.NoError(t, err)
	require.Equal(t, walletdb.NameActionFinalizeDutchAuctionListing, hist.History[0].Type)
}

func (s *DutchAuctionSuite) TestTransferDutchAuctionFill() {
	t := s.T()

	s.transferListing()
	s.finalizeListing()
	s.generate()
	s.fill()

	hist, err := s.client.GetName("alice", s.name)
	require.NoError(t, err)
	require.Equal(t, walletdb.NameActionFillDutchAuction, hist.History[0].Type)

	hist, err = s.client.GetName("bob", s.name)
	require.NoError(t, err)
	require.Equal(t, walletdb.NameActionTransferFillDutchAuction, hist.History[0].Type)
}

func (s *DutchAuctionSuite) TestFinalizeDutchAuctionFill() {
	t := s.T()

	s.transferListing()
	s.finalizeListing()
	s.generate()
	s.fill()
	s.finalize()

	hist, err := s.client.GetName("bob", s.name)
	require.NoError(t, err)
	require.Equal(t, walletdb.NameActionFinalizeFillDutchAuction, hist.History[0].Type)

	names, err := s.client.GetNames("bob")
	require.NoError(t, err)
	require.Equal(t, s.name, names.Names[0].Name)
	require.Equal(t, walletdb.NameStatusOwned, names.Names[0].Status)
}

func (s *DutchAuctionSuite) TestTransferCancelDutchAuction() {
	t := s.T()

	s.transferListing()
	s.finalizeListing()
	s.cancelTransfer()

	hist, err := s.client.GetName("alice", s.name)
	require.NoError(t, err)
	require.Equal(t, walletdb.NameActionTransferDutchAuctionCancel, hist.History[0].Type)
}

func (s *DutchAuctionSuite) TestFinalizeCancelDutchAuction() {
	t := s.T()

	s.transferListing()
	s.finalizeListing()
	s.cancelTransfer()
	s.cancelFinalize()

	hist, err := s.client.GetName("alice", s.name)
	require.NoError(t, err)
	require.Equal(t, walletdb.NameActionFinalizeDutchAuctionCancel, hist.History[0].Type)
}

func TestDutchAuctionSuite(t *testing.T) {
	suite.Run(t, new(DutchAuctionSuite))
}
