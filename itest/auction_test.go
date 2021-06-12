package itest

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type AccountAuctionSuite struct {
	suite.Suite
	hsd       *HSD
	client    *api.Client
	cleanup   func()
	aliceInfo *api.AccountGetRes
	bobInfo   *api.AccountGetRes
}

func (s *AccountAuctionSuite) SetupTest() {
	chain.SetCurrNetwork(chain.NetworkRegtest)

	t := s.T()
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

	mineTo(t, s.hsd.Client, s.client, 1, s.aliceInfo.ReceiveAddress)
	mineTo(t, s.hsd.Client, s.client, 1, s.bobInfo.ReceiveAddress)
	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.CoinbaseMaturity, ZeroRegtestAddr)
	awaitHeight(t, s.client, "bob", 4)

	err = s.client.Unlock("alice", "password")
	require.NoError(t, err)

	err = s.client.Unlock("bob", "password")
	require.NoError(t, err)
}

func (s *AccountAuctionSuite) TearDownTest() {
	if s.cleanup != nil {
		s.cleanup()
	}
	s.hsd.Stop()
}

func (s *AccountAuctionSuite) doBids() {
	name := "awilauh"
	t := s.T()
	_, err := s.client.Open("alice", name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TreeInterval+2, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 11)

	_, err = s.client.Bid("alice", name, 100, 1000000, 2000000, false)
	require.NoError(t, err)
	_, err = s.client.Bid("alice", name, 100, 2000000, 4000000, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.BiddingPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 16)

	_, err = s.client.Reveal("alice", name, 100, false)
	require.NoError(t, err)
}

func (s *AccountAuctionSuite) TestOpenNameBlacklisted() {
	t := s.T()
	_, err := s.client.Open("alice", "localhost", 100, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid name")
}

func (s *AccountAuctionSuite) TestOpenNameInvalid() {
	t := s.T()
	_, err := s.client.Open("alice", "-notvalid-", 100, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid name")
}

func (s *AccountAuctionSuite) TestOpenNameBadRollout() {
	t := s.T()
	_, err := s.client.Open("alice", "supername", 100, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name not rolled out yet")
}

func (s *AccountAuctionSuite) TestOpenNameOK() {
	name := "awilauh"
	t := s.T()
	_, err := s.client.Open("alice", name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 5)

	info, err := s.hsd.Client.GetNameInfo(name)
	require.NoError(t, err)
	require.Equal(t, "OPENING", info.Info.State)
}

func (s *AccountAuctionSuite) TestBidNameBlacklisted() {
	t := s.T()
	_, err := s.client.Bid("alice", "localhost", 100, 1000000, 2000000, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid name")
}

func (s *AccountAuctionSuite) TestBidNameInvalid() {
	t := s.T()
	_, err := s.client.Bid("alice", "-notvalid-", 100, 1000000, 2000000, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid name")
}

func (s *AccountAuctionSuite) TestBidNameBadRollout() {
	t := s.T()
	_, err := s.client.Bid("alice", "supername", 100, 1000000, 2000000, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name not rolled out yet")
}

func (s *AccountAuctionSuite) TestBidValueExceedsLockup() {
	t := s.T()
	_, err := s.client.Bid("alice", "awilauh", 100, 2000000, 1000000, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "value exceeds lockup")
}

func (s *AccountAuctionSuite) TestBidNameNeverOpened() {
	t := s.T()
	_, err := s.client.Bid("alice", "awilauh", 100, 1000000, 2000000, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name state must be BIDDING")
}

func (s *AccountAuctionSuite) TestBidNameNotBidding() {
	name := "awilauh"
	t := s.T()
	_, err := s.client.Open("alice", name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 5)

	_, err = s.client.Bid("alice", name, 100, 1000000, 2000000, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name state must be BIDDING")
}

func (s *AccountAuctionSuite) TestBidOK() {
	name := "awilauh"
	t := s.T()
	_, err := s.client.Open("alice", name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TreeInterval+2, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 11)

	_, err = s.client.Bid("alice", name, 100, 1000000, 2000000, false)
	require.NoError(t, err)
}

func (s *AccountAuctionSuite) TestRevealNeverOpened() {
	t := s.T()
	_, err := s.client.Reveal("alice", "awilauh", 100, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name state must be REVEAL")
}

func (s *AccountAuctionSuite) TestRevealNameNotRevealing() {
	name := "awilauh"
	t := s.T()
	_, err := s.client.Open("alice", name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TreeInterval+2, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 11)

	_, err = s.client.Bid("alice", name, 100, 1000000, 2000000, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 12)

	_, err = s.client.Reveal("alice", name, 100, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name state must be REVEAL")
}

func (s *AccountAuctionSuite) TestRevealOK() {
	name := "awilauh"
	t := s.T()
	_, err := s.client.Open("alice", name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TreeInterval+2, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 11)

	_, err = s.client.Bid("alice", name, 100, 1000000, 2000000, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.BiddingPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 16)

	_, err = s.client.Reveal("alice", name, 100, false)
	require.NoError(t, err)
}

func (s *AccountAuctionSuite) TestRevealMultipleBidsOK() {
	name := "awilauh"
	t := s.T()
	_, err := s.client.Open("alice", name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TreeInterval+2, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 11)

	_, err = s.client.Bid("alice", name, 100, 1000000, 2000000, false)
	require.NoError(t, err)
	_, err = s.client.Bid("alice", name, 100, 2000000, 4000000, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.BiddingPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 16)

	_, err = s.client.Reveal("alice", name, 100, false)
	require.NoError(t, err)
}

func (s *AccountAuctionSuite) TestRedeemBidsAuctionNotClosed() {
	t := s.T()
	name := "awilauh"

	s.doBids()

	_, err := s.client.Redeem("alice", name, 100, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name state must be CLOSED")
}

func (s *AccountAuctionSuite) TestRedeemBidsOK() {
	t := s.T()
	name := "awilauh"

	s.doBids()

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.RevealPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 16+chain.NetworkRegtest.RevealPeriod)

	_, err := s.client.Redeem("alice", name, 100, false)
	require.NoError(t, err)
}

func (s *AccountAuctionSuite) TestRegisterNoRecordsOK() {
	t := s.T()
	name := "awilauh"

	s.doBids()

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.RevealPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 16+chain.NetworkRegtest.RevealPeriod)

	_, err := s.client.Redeem("alice", name, 100, false)
	require.NoError(t, err)
	tx, err := s.client.Update("alice", name, nil, 100, false)
	require.NoError(t, err)

	nameInfo, err := s.hsd.Client.GetNameInfo(name)
	require.NoError(t, err)
	require.Equal(t, nameInfo.Info.Owner.Hash, tx.Inputs[0].Prevout.Hash)
	require.EqualValues(t, nameInfo.Info.Owner.Index, tx.Inputs[0].Prevout.Index)
}

func (s *AccountAuctionSuite) TestRegisterWithRecordsOK() {
	t := s.T()
	name := "awilauh"

	s.doBids()

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.RevealPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 16+chain.NetworkRegtest.RevealPeriod)

	_, err := s.client.Redeem("alice", name, 100, false)
	require.NoError(t, err)
	resource := &chain.Resource{
		TTL: 2600,
		Records: []chain.Record{
			&chain.TXTRecord{
				Entries: []string{
					"clap them cheeks",
				},
			},
		},
	}
	tx, err := s.client.Update("alice", name, resource, 100, false)
	require.NoError(t, err)

	nameInfo, err := s.hsd.Client.GetNameInfo(name)
	require.NoError(t, err)
	require.Equal(t, nameInfo.Info.Owner.Hash, tx.Inputs[0].Prevout.Hash)
	require.EqualValues(t, nameInfo.Info.Owner.Index, tx.Inputs[0].Prevout.Index)
}

func TestAccountAuction(t *testing.T) {
	suite.Run(t, new(AccountAuctionSuite))
}
