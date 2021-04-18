package itest

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type RevealSuite struct {
	suite.Suite
	hsd     *HSD
	client  *api.Client
	cleanup func()
	name    string
}

func (s *RevealSuite) SetupTest() {
	t := s.T()
	s.name = "awilauh"
	s.hsd = startHSD()
	s.client, s.cleanup = startDaemon(t)

	_, err := s.client.CreateWallet(&api.CreateWalletReq{
		Name:     "alice",
		Password: "password",
	})
	require.NoError(t, err)
	_, err = s.client.CreateWallet(&api.CreateWalletReq{
		Name:     "bob",
		Password: "password",
	})
	require.NoError(t, err)

	err = s.client.Unlock("alice", "password")
	require.NoError(t, err)
	err = s.client.Unlock("bob", "password")
	require.NoError(t, err)

	aliceInfo, err := s.client.GetAccount("alice", "default")
	require.NoError(t, err)
	bobInfo, err := s.client.GetAccount("bob", "default")
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, aliceInfo.ReceiveAddress)
	mineTo(t, s.hsd.Client, s.client, 1, bobInfo.ReceiveAddress)
	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.CoinbaseMaturity, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", 4)

	_, err = s.client.Open("alice", "default", s.name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TreeInterval+2, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", 11)
}

func (s *RevealSuite) TearDownTest() {
	s.cleanup()
	s.hsd.Stop()
}

func (s *RevealSuite) TestMultipleBids() {
	t := s.T()
	for i := 0; i < 3; i++ {
		_, err := s.client.Bid(
			"alice",
			"default",
			s.name,
			100,
			uint64((i+1)*1000000),
			uint64((i+1)*2000000),
			false,
		)
		require.NoError(t, err)
	}

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.BiddingPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", chain.NetworkRegtest.BiddingPeriod+11)

	revs, err := s.client.Reveal("alice", "default", s.name, 100, false)
	require.NoError(t, err)
	require.Equal(t, 3, len(revs.Inputs))
}

func (s *RevealSuite) TestSingleBid() {
	t := s.T()
	_, err := s.client.Bid(
		"alice",
		"default",
		s.name,
		100,
		1000000,
		2000000,
		false,
	)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.BiddingPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", chain.NetworkRegtest.BiddingPeriod+11)

	_, err = s.client.Reveal("alice", "default", s.name, 100, false)
	require.NoError(t, err)
}

func (s *RevealSuite) TestNoBids() {
	t := s.T()

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.BiddingPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", chain.NetworkRegtest.BiddingPeriod+11)

	_, err := s.client.Reveal("alice", "default", s.name, 100, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no bids to reveal")
}

func TestRevealSuite(t *testing.T) {
	suite.Run(t, new(RevealSuite))
}
