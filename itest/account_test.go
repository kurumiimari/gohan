package itest

import (
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type AccountSuite struct {
	suite.Suite
	hsd     *HSD
	client  *api.Client
	cleanup func()
}

func (s *AccountSuite) SetupSuite() {
	s.hsd = startHSD()
}

func (s *AccountSuite) TearDownSuite() {
	s.hsd.Stop()
}

func (s *AccountSuite) SetupTest() {
	t := s.T()
	s.client, s.cleanup = startDaemon(t)

	_, err := s.client.CreateAccount(&api.CreateAccountReq{
		ID:       "alice",
		Mnemonic: Mnemonic,
		Password: "password",
	})
	require.NoError(t, err)
}

func (s *AccountSuite) TearDownTest() {
	s.cleanup()
}

func (s *AccountSuite) TestGetAccounts() {
	t := s.T()
	accounts, err := s.client.GetAccounts()
	require.NoError(t, err)
	require.EqualValues(t, []string{"alice"}, accounts.Accounts)
}

func (s *AccountSuite) TestGetAccountInfo() {
	t := s.T()
	info, err := s.client.GetAccount("alice")
	require.NoError(t, err)
	require.Equal(t, &api.AccountGetRes{
		ID:    "alice",
		Index: 0,
		Balances: &walletdb.Balances{
			Available:    0,
			Immature:     0,
			BidLocked:    0,
			RevealLocked: 0,
		},
		AddressDepth: &api.AccountAddressDepth{
			Receive: 1,
			Change:  1,
		},
		LookaheadDepth: &api.AccountAddressDepth{
			Receive: 10,
			Change:  10,
		},
		ReceiveAddress: "rs1qedtqrtu8eavsl7sepgy3fp336966pqxmyhnquc",
		ChangeAddress:  "rs1qhl5h3pqet2gqy93ua97rf4sxdkfcels9ayqhsa",
		XPub:           "rpubKBBUaydwRpVxLcm8YESMRikrSFRG9nsXDquhppVigpKymkS6fhoKxxJa1Ud76TgHUMMrvAvqJXyxkJKjWdmX6uSkQNYKHnuqDnDsLSVyVQnQ",
	}, info)
}

func (s *AccountSuite) TestGenerateReceiveAddress() {
	t := s.T()

	addr, err := s.client.GenerateAccountReceiveAddress("alice")
	require.NoError(t, err)
	require.Equal(t, &api.GenAddressRes{
		Address:    "rs1qztdtxtrcdvkyxsk5r8fhyymh0mw6ju338pe9wg",
		Derivation: "m/0/1",
	}, addr)
	info, err := s.client.GetAccount("alice")
	require.NoError(t, err)
	require.EqualValues(t, 11, info.LookaheadDepth.Receive)
	require.EqualValues(t, 2, info.AddressDepth.Receive)
	require.Equal(t, info.ReceiveAddress, "rs1qztdtxtrcdvkyxsk5r8fhyymh0mw6ju338pe9wg")
}

func (s *AccountSuite) TestGenerateChangeAddress() {
	t := s.T()

	addr, err := s.client.GenerateAccountChangeAddress("alice")
	require.NoError(t, err)
	require.Equal(t, &api.GenAddressRes{
		Address:    "rs1qn8s8ua95pve9f86pu8e9ksf2elxrv4cdvp0qdq",
		Derivation: "m/1/1",
	}, addr)
	info, err := s.client.GetAccount("alice")
	require.NoError(t, err)
	require.EqualValues(t, 11, info.LookaheadDepth.Change)
	require.EqualValues(t, 2, info.AddressDepth.Change)
	require.Equal(t, info.ChangeAddress, "rs1qn8s8ua95pve9f86pu8e9ksf2elxrv4cdvp0qdq")
}

func TestAccountSuite(t *testing.T) {
	suite.Run(t, new(AccountSuite))
}
