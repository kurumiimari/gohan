package itest

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type AccountBalanceSuite struct {
	suite.Suite
	hsd     *HSD
	client  *api.Client
	cleanup func()
}

func (s *AccountBalanceSuite) SetupTest() {
	t := s.T()
	s.hsd = startHSD()
	s.client, s.cleanup = startDaemon(t)

	_, err := s.client.CreateAccount(&api.CreateAccountReq{
		ID:       "alice",
		Password: "password",
	})
	require.NoError(t, err)
}

func (s *AccountBalanceSuite) TearDownTest() {
	s.cleanup()
	s.hsd.Stop()
}

func (s *AccountBalanceSuite) TestImmatureCoinbaseBalance() {
	t := s.T()
	info, err := s.client.GetAccount("alice")
	require.NoError(t, err)
	mineTo(t, s.hsd.Client, s.client, 1, info.ReceiveAddress)
	awaitHeight(t, s.client, "alice", 1)
	info, err = s.client.GetAccount("alice")
	require.NoError(t, err)
	require.EqualValues(t, 0, info.Balances.Available)
	require.EqualValues(t, 2000000000, info.Balances.Immature)
	require.EqualValues(t, 0, info.Balances.RevealLocked)
	require.EqualValues(t, 0, info.Balances.BidLocked)
}

func (s *AccountBalanceSuite) TestMatureCoinbaseBalance() {
	t := s.T()
	info, err := s.client.GetAccount("alice")
	require.NoError(t, err)
	mineTo(t, s.hsd.Client, s.client, 1, info.ReceiveAddress)
	awaitHeight(t, s.client, "alice", 1)
	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.CoinbaseMaturity, ZeroRegtestAddr)
	mineTo(t, s.hsd.Client, s.client, 1, info.ReceiveAddress)
	awaitHeight(t, s.client, "alice", chain.NetworkRegtest.CoinbaseMaturity+2)
	info, err = s.client.GetAccount("alice")
	require.NoError(t, err)
	require.EqualValues(t, 2000000000, info.Balances.Available)
	require.EqualValues(t, 2000000000, info.Balances.Immature)
	require.EqualValues(t, 0, info.Balances.RevealLocked)
	require.EqualValues(t, 0, info.Balances.BidLocked)
}

func TestAccountBalanceSuite(t *testing.T) {
	suite.Run(t, new(AccountBalanceSuite))
}
