package itest

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type RenewSuite struct {
	suite.Suite
	hsd     *HSD
	client  *api.Client
	cleanup func()
	name    string
}

func (s *RenewSuite) SetupTest() {
	t := s.T()
	s.name = "awilauh"
	s.hsd = startHSD()
	s.client, s.cleanup = startDaemon(t)

	_, err := s.client.CreateWallet(&api.CreateWalletReq{
		Name:     "alice",
		Password: "password",
	})
	require.NoError(t, err)

	err = s.client.Unlock("alice", "password")
	require.NoError(t, err)

	info, err := s.client.GetAccount("alice", "default")
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, info.ReceiveAddress)
	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.CoinbaseMaturity, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", 3)

	_, err = s.client.Open("alice", "default", s.name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TreeInterval+2, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", 10)

	_, err = s.client.Bid("alice", "default", s.name, 100, 1000000, 2000000, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.BiddingPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", 15)

	_, err = s.client.Reveal("alice", "default", s.name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.RevealPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", 15+chain.NetworkRegtest.RevealPeriod)

	_, err = s.client.Update("alice", "default", s.name, nil, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", 16+chain.NetworkRegtest.RevealPeriod)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TreeInterval, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", 21+chain.NetworkRegtest.RevealPeriod)
}

func (s *RenewSuite) TearDownTest() {
	s.cleanup()
	s.hsd.Stop()
}

func (s *RenewSuite) TestRenewOK() {
	t := s.T()
	_, err := s.client.Renew("alice", "default", s.name, 100, false)
	require.NoError(t, err)
}

func TestRenewSuite(t *testing.T) {
	suite.Run(t, new(RenewSuite))
}
