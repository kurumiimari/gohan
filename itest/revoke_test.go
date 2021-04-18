package itest

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type RevokeSuite struct {
	suite.Suite
	hsd     *HSD
	client  *api.Client
	cleanup func()
	name    string
}

func (s *RevokeSuite) SetupTest() {
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

	info, err := s.client.GetAccount("alice", "default")
	require.NoError(t, err)

	err = s.client.Unlock("alice", "password")
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
}

func (s *RevokeSuite) TearDownTest() {
	s.cleanup()
	s.hsd.Stop()
}

func (s *RevokeSuite) TestRevokeAfterUpdate() {
	t := s.T()

	_, err := s.client.Revoke("alice", "default", s.name, 100, false)
	require.NoError(t, err)

	s.runRequirements([]string{
		"REVOKE",
		"REGISTER",
		"REVEAL",
		"BID",
		"OPEN",
	})
}

func (s *RevokeSuite) TestRevokeAfterTransfer() {
	t := s.T()

	_, err := s.client.Transfer("alice", "default", s.name, ZeroRegtestAddr, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", "default", 17+chain.NetworkRegtest.RevealPeriod)

	_, err = s.client.Revoke("alice", "default", s.name, 100, false)
	require.NoError(t, err)

	s.runRequirements([]string{
		"REVOKE",
		"TRANSFER",
		"REGISTER",
		"REVEAL",
		"BID",
		"OPEN",
	})
}

func (s *RevokeSuite) TestRevokeNotOpened() {
	t := s.T()

	_, err := s.client.Revoke("alice", "default", "notopened", 100, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name state must be CLOSED")
}

func (s *RevokeSuite) runRequirements(actions []string) {
	t := s.T()

	names, err := s.client.GetNames("alice", "default")
	require.NoError(t, err)
	require.Equal(t, 1, len(names.Names))
	require.Equal(t, walletdb.NameStatusRevoked, names.Names[0].Status)

	history, err := s.client.GetName("alice", "default", s.name)
	require.NoError(t, err)

	require.Equal(t, len(actions), len(history.History))
	for i, action := range actions {
		require.Equal(t, walletdb.NameHistoryType(action), history.History[i].Type)
	}

	require.Equal(t, history.History[1].OutIdx, *history.History[0].ParentOutIdx)
	require.Equal(t, history.History[1].Transaction.Hash, *history.History[0].ParentTxHash)
}

func TestRevoke(t *testing.T) {
	suite.Run(t, new(RevokeSuite))
}
