package itest

import (
	"github.com/kurumiimari/gohan/testutil"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type RescanSuite struct {
	suite.Suite
	hsd     *HSD
	client  *api.Client
	cleanup func()
}

func (s *RescanSuite) SetupSuite() {
	t := s.T()
	s.hsd = startHSDWithData("testdata/regtest-1.tar.gz")
	s.client, s.cleanup = startDaemon(t)

	_, err := s.client.CreateAccount(&api.CreateAccountReq{
		ID:       "alice",
		Mnemonic: "few derive language prison worth heavy prosper seven bone discover journey lonely sketch success marine robust crew egg fork misery certain drill seminar warrior",
		Password: "password",
	})
	require.NoError(t, err)
	_, err = s.client.CreateAccount(&api.CreateAccountReq{
		ID:       "bob",
		Mnemonic: "someone crack panel bright wheel comic wagon evoke train awful hotel broccoli admit salmon select electric become client canvas axis thing topic awkward used",
		Password: "password",
	})
	require.NoError(t, err)

	awaitHeight(t, s.client, "alice", 53)
	awaitHeight(t, s.client, "bob", 53)
}

func (s *RescanSuite) TearDownSuite() {
	s.cleanup()
	s.hsd.Stop()
}

func (s *RescanSuite) TestAccountInfo() {
	t := s.T()
	aliceInfo, err := s.client.GetAccount("alice")
	require.NoError(t, err)
	testutil.RequireEqualJSONFile(t, "alice-account.json", aliceInfo)
}

func (s *RescanSuite) TestNameHistoryOwned() {
	t := s.T()
	history, err := s.client.GetName("alice", "whncsjjgtc")
	require.NoError(t, err)
	testutil.RequireEqualJSONFile(t, "alice-history-owned.json", history)
}

func (s *RescanSuite) TestNameHistoryRevoked() {
	t := s.T()
	history, err := s.client.GetName("alice", "xpnjsegaep")
	require.NoError(t, err)
	testutil.RequireEqualJSONFile(t, "history-revoked.json", history)
}

func (s *RescanSuite) TestNameHistoryTransferredIn() {
	t := s.T()
	history, err := s.client.GetName("alice", "rhtnrfaemi")
	require.NoError(t, err)
	testutil.RequireEqualJSONFile(t, "alice-history-transferred-in.json", history)
}

func (s *RescanSuite) TestNames() {
	t := s.T()
	names, err := s.client.GetNames("alice")
	require.NoError(t, err)
	testutil.RequireEqualJSONFile(t, "alice-names.json", names)
	names, err = s.client.GetNames("bob")
	testutil.RequireEqualJSONFile(t, "bob-names.json", names)
}

func (s *RescanSuite) TestTransactions() {
	t := s.T()
	txs, err := s.client.GetAccountTransactions("alice", 1000, 0)
	require.NoError(t, err)
	testutil.RequireEqualJSONFile(t, "alice-txs.json", txs)
}

func TestRescanSuite(t *testing.T) {
	suite.Run(t, new(RescanSuite))
}
