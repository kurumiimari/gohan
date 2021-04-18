package itest

import (
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type WalletSuite struct {
	suite.Suite
	hsd *HSD
}

func (s *WalletSuite) SetupSuite() {
	s.hsd = startHSD()
}

func (s *WalletSuite) TearDownSuite() {
	s.hsd.Stop()
}

func (s *WalletSuite) TestCreateWallet_ImportMnemonic() {
	t := s.T()
	client, cleanup := startDaemon(t)
	defer cleanup()

	res, err := client.CreateWallet(&api.CreateWalletReq{
		Name:     "testwallet",
		Mnemonic: Mnemonic,
		Password: "password",
	})
	require.NoError(t, err)
	require.EqualValues(t, &api.CreateWalletRes{
		Name:      "testwallet",
		Mnemonic:  nil,
		WatchOnly: false,
	}, res)
}

func (s *WalletSuite) TestCreateWallet_New() {
	t := s.T()
	client, cleanup := startDaemon(t)
	defer cleanup()

	res, err := client.CreateWallet(&api.CreateWalletReq{
		Name:     "testwallet",
		Password: "password",
	})
	require.NoError(t, err)
	require.Equal(t, "testwallet", res.Name)
	require.NotNil(t, res.Mnemonic)
	require.False(t, res.WatchOnly)
}

func (s *WalletSuite) TestCreateWallet_WatchOnly() {
	t := s.T()
	client, cleanup := startDaemon(t)
	defer cleanup()

	res, err := client.CreateWallet(&api.CreateWalletReq{
		Name: "testwallet",
		XPub: "xpub6CMpnZHN1Zaqx2ctpHmqamD8NwEoEWpWia2pfojKZMmj5JfqKa1GNz4CZfZHr3LosxjFy98wV39XRX1BdkXxLwzyEYwyJ9eCFwyNtA5gniA",
	})
	require.NoError(t, err)
	require.EqualValues(t, &api.CreateWalletRes{
		Name:      "testwallet",
		Mnemonic:  nil,
		WatchOnly: true,
	}, res)
}

func (s *WalletSuite) TestUnlockWallet() {
	t := s.T()
	client, cleanup := startDaemon(t)
	defer cleanup()

	_, err := client.CreateWallet(&api.CreateWalletReq{
		Name:     "testwallet",
		Password: "password",
	})
	require.NoError(t, err)

	require.NoError(t, client.Unlock("testwallet", "password"))
	require.Error(t, client.Unlock("testwallet", "badpassword"))
}

func TestWalletSuite(t *testing.T) {
	suite.Run(t, new(WalletSuite))
}
