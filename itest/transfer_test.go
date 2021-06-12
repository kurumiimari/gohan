package itest

import (
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type TransferSuite struct {
	suite.Suite
	hsd       *HSD
	client    *api.Client
	cleanup   func()
	name      string
	aliceInfo *api.AccountGetRes
	bobInfo   *api.AccountGetRes
}

func (s *TransferSuite) SetupTest() {
	t := s.T()
	s.name = "awilauh"
	s.hsd = startHSD()
	s.client, s.cleanup = startDaemon(t)

	_, err := s.client.CreateAccount(&api.CreateAccountReq{
		ID:       "alice",
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
	awaitHeight(t, s.client, "bob", 4)

	_, err = s.client.Open("alice", s.name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TreeInterval+2, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 11)

	_, err = s.client.Bid("alice", s.name, 100, 1000000, 2000000, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.BiddingPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 16)

	_, err = s.client.Reveal("alice", s.name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.RevealPeriod, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 16+chain.NetworkRegtest.RevealPeriod)

	_, err = s.client.Update("alice", s.name, nil, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", 17+chain.NetworkRegtest.RevealPeriod)
}

func (s *TransferSuite) TearDownTest() {
	s.cleanup()
	s.hsd.Stop()
}

func (s *TransferSuite) TestTransferState() {
	t := s.T()

	_, err := s.client.Transfer("alice", s.name, s.bobInfo.ReceiveAddress, 100, false)
	require.NoError(t, err)

	aliceNames, err := s.client.GetNames("alice")
	require.NoError(t, err)

	require.Equal(t, 1, len(aliceNames.Names))
	require.Equal(t, walletdb.NameStatusTransferring, aliceNames.Names[0].Status)
}

func (s *TransferSuite) TestTransferFinalizeOK() {
	t := s.T()

	_, err := s.client.Transfer("alice", s.name, s.bobInfo.ReceiveAddress, 100, false)
	require.NoError(t, err)

	startBlock := 17 + chain.NetworkRegtest.RevealPeriod
	mineTo(t, s.hsd.Client, s.client, chain.NetworkRegtest.TransferLockup+1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", startBlock+chain.NetworkRegtest.TransferLockup+1)
	startBlock += chain.NetworkRegtest.TransferLockup + 1

	_, err = s.client.Finalize("alice", s.name, 100, false)
	require.NoError(t, err)

	mineTo(t, s.hsd.Client, s.client, 1, ZeroRegtestAddr)
	awaitHeight(t, s.client, "alice", startBlock+1)
	awaitHeight(t, s.client, "bob", startBlock+1)

	aliceNames, err := s.client.GetNames("alice")
	require.NoError(t, err)
	bobNames, err := s.client.GetNames("bob")
	require.NoError(t, err)

	require.Equal(t, 1, len(aliceNames.Names))
	require.Equal(t, walletdb.NameStatusTransferred, aliceNames.Names[0].Status)
	require.Equal(t, 1, len(bobNames.Names))
	require.Equal(t, walletdb.NameStatusOwned, bobNames.Names[0].Status)
}

func TestTransfer(t *testing.T) {
	suite.Run(t, new(TransferSuite))
}
