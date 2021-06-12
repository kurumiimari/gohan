package itest

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type AccountSendSuite struct {
	suite.Suite
	hsd     *HSD
	client  *api.Client
	cleanup func()
}

func (s *AccountSendSuite) SetupTest() {
	t := s.T()
	s.hsd = startHSD()
	s.client, s.cleanup = startDaemon(t)

	_, err := s.client.CreateWallet(&api.CreateWalletReq{
		Name:     "alice",
		Password: "password",
	})
	require.NoError(t, err)
}

func (s *AccountSendSuite) TearDownTest() {
	s.cleanup()
	s.hsd.Stop()
}

func (s *AccountSendSuite) TestSendWalletLocked() {
	t := s.T()

	info, err := s.client.GetAccount("alice", "default")
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, info.ReceiveAddress)
	awaitHeight(t, s.client, "alice", "default", 1)
	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.CoinbaseMaturity + 1, ZeroRegtestAddr)

	_, err = s.client.Send("alice", "default", 10000, 10000, ZeroRegtestAddr, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "locked")
}

func (s *AccountSendSuite) TestSendInsufficientFunds() {
	t := s.T()

	info, err := s.client.GetAccount("alice", "default")
	require.NoError(t, err)

	err = s.client.Unlock("alice", "password")
	require.NoError(t, err)

	_, err = s.client.Send("alice", "default", 10000, 10000, ZeroRegtestAddr, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient funds")
	mineTo(t, s.hsd.Client, s.client, 1, info.ReceiveAddress)
	awaitHeight(t, s.client, "alice", "default", 1)
	_, err = s.client.Send("alice", "default", 10000, 10000, ZeroRegtestAddr, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient funds")
}

func (s *AccountSendSuite) TestSendBalanceDecreases() {
	t := s.T()

	info, err := s.client.GetAccount("alice", "default")
	require.NoError(t, err)

	err = s.client.Unlock("alice", "password")
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, info.ReceiveAddress)
	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.CoinbaseMaturity, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", 3)
	_, err = s.client.Send("alice", "default", 1000000, 100, ZeroRegtestAddr, false)
	require.NoError(t, err)

	info, err = s.client.GetAccount("alice", "default")
	require.NoError(t, err)
	require.EqualValues(t, 1998982000, info.Balances.Available)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.CoinbaseMaturity, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", 3)

	info, err = s.client.GetAccount("alice", "default")
	require.NoError(t, err)
	require.EqualValues(t, 1998982000, info.Balances.Available)
}

func TestAccountSend(t *testing.T) {
	suite.Run(t, new(AccountSendSuite))
}
