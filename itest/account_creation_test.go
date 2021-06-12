package itest

import (
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type AccountCreationSuite struct {
	suite.Suite
	hsd *HSD
}

func (s *AccountCreationSuite) SetupSuite() {
	s.hsd = startHSD()
}

func (s *AccountCreationSuite) TearDownSuite() {
	s.hsd.Stop()
}

func (s *AccountCreationSuite) TestImportMnemonic() {
	t := s.T()
	client, cleanup := startDaemon(t)
	defer cleanup()

	res, err := client.CreateAccount(&api.CreateAccountReq{
		ID:       "testwallet",
		Mnemonic: Mnemonic,
		Password: "password",
	})
	require.NoError(t, err)
	require.EqualValues(t, &api.CreateAccountRes{
		ID:        "testwallet",
		Mnemonic:  nil,
		WatchOnly: false,
	}, res)
}

func (s *AccountCreationSuite) TestBrandNew() {
	t := s.T()
	client, cleanup := startDaemon(t)
	defer cleanup()

	res, err := client.CreateAccount(&api.CreateAccountReq{
		ID:       "testwallet",
		Password: "password",
	})
	require.NoError(t, err)
	require.Equal(t, "testwallet", res.ID)
	require.NotNil(t, res.Mnemonic)
	require.False(t, res.WatchOnly)
}

func (s *AccountCreationSuite) TestWatchOnly() {
	t := s.T()
	client, cleanup := startDaemon(t)
	defer cleanup()

	res, err := client.CreateAccount(&api.CreateAccountReq{
		ID:   "testwallet",
		XPub: "xpub6CMpnZHN1Zaqx2ctpHmqamD8NwEoEWpWia2pfojKZMmj5JfqKa1GNz4CZfZHr3LosxjFy98wV39XRX1BdkXxLwzyEYwyJ9eCFwyNtA5gniA",
	})
	require.NoError(t, err)
	require.EqualValues(t, &api.CreateAccountRes{
		ID:        "testwallet",
		Mnemonic:  nil,
		WatchOnly: true,
	}, res)
}

func (s *AccountCreationSuite) TestUnlock() {
	t := s.T()
	client, cleanup := startDaemon(t)
	defer cleanup()

	_, err := client.CreateAccount(&api.CreateAccountReq{
		ID:       "testwallet",
		Password: "password",
	})
	require.NoError(t, err)

	require.NoError(t, client.Unlock("testwallet", "password"))
	require.Error(t, client.Unlock("testwallet", "badpassword"))
}

func TestWalletSuite(t *testing.T) {
	suite.Run(t, new(AccountCreationSuite))
}
