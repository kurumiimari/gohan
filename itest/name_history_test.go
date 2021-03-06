package itest

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type NameHistorySuite struct {
	suite.Suite
	hsd       *HSD
	client    *api.Client
	cleanup   func()
	name      string
	aliceInfo *api.AccountGetRes
	bobInfo   *api.AccountGetRes
}

func (s *NameHistorySuite) SetupTest() {
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

	mineTo(t, s.hsd.Client, s.client, 1, s.aliceInfo.ReceiveAddress)
	mineTo(t, s.hsd.Client, s.client, 1, s.bobInfo.ReceiveAddress)
	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.CoinbaseMaturity, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 4)
	awaitHeight(t, s.client, "bob", 4)
}

func (s *NameHistorySuite) TearDownTest() {
	s.cleanup()
	s.hsd.Stop()
}

func (s *NameHistorySuite) doAuction() {
	t := s.T()

	_, err := s.client.Open("alice", s.name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TreeInterval+2, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 11)

	_, err = s.client.Bid("alice", s.name, 100, 1000000, 2000000, false)
	require.NoError(t, err)
	_, err = s.client.Bid("alice", s.name, 100, 2000000, 4000000, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.BiddingPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 16)

	_, err = s.client.Reveal("alice", s.name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.RevealPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 16+chain.NetworkRegtest.RevealPeriod)

	_, err = s.client.Update("alice", s.name, nil, 100, false)
	require.NoError(t, err)

	_, err = s.client.Redeem("alice", s.name, 100, false)
	require.NoError(t, err)
}

func (s *NameHistorySuite) TestCompleteAuction() {
	t := s.T()
	s.doAuction()

	history, err := s.client.GetName("alice", s.name)
	require.NoError(t, err)

	actions := []string{
		"REGISTER",
		"REDEEM",
		"REVEAL",
		"REVEAL",
		"BID",
		"BID",
		"OPEN",
	}

	require.Equal(t, len(actions), len(history.History))
	for i, action := range actions {
		require.Equal(t, walletdb.NameHistoryType(action), history.History[i].Type)
	}
}

func (s *NameHistorySuite) TestTransferFinalize() {
	t := s.T()
	s.doAuction()

	startBlock := 16 + chain.NetworkRegtest.RevealPeriod
	mineTo(t, s.hsd.Client, s.client, 1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", startBlock+1)
	startBlock += 1

	_, err := s.client.Transfer("alice", s.name, s.bobInfo.ReceiveAddress, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TransferLockup+1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", startBlock+chain.NetworkRegtest.TransferLockup+1)
	startBlock += chain.NetworkRegtest.TransferLockup + 1

	_, err = s.client.Finalize("alice", s.name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", startBlock+1)
	awaitHeight(t, s.client, "bob", startBlock+1)
	startBlock += 1

	history, err := s.client.GetName("alice", s.name)
	require.NoError(t, err)

	actions := []string{
		"FINALIZE_OUT",
		"TRANSFER",
		"REGISTER",
		"REDEEM",
		"REVEAL",
		"REVEAL",
		"BID",
		"BID",
		"OPEN",
	}

	require.Equal(t, len(actions), len(history.History))
	for i, action := range actions {
		require.Equal(t, walletdb.NameHistoryType(action), history.History[i].Type)
	}

	history, err = s.client.GetName("bob", s.name)
	require.NoError(t, err)

	actions = []string{
		"FINALIZE_IN",
	}

	require.Equal(t, len(actions), len(history.History))
	for i, action := range actions {
		require.Equal(t, walletdb.NameHistoryType(action), history.History[i].Type)
	}
}

func TestNameHistory(t *testing.T) {
	suite.Run(t, new(NameHistorySuite))
}
