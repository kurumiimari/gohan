package cmd

import (
	"github.com/kurumiimari/gohan"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/log"
	"github.com/kurumiimari/gohan/wallet"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
)

var (
	prefix    string
	network   string
	walletURL string
	nodeURL   string
)

var cmdLogger = log.ModuleLogger("cmd")

var rootCmd = &cobra.Command{
	Use:          "gohan",
	Short:        "A Handshake wallet node",
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		network, err := chain.NetworkFromName(network)
		if err != nil {
			return errors.Wrap(err, "invalid network")
		}

		dd, err := wallet.NewDataDir(prefix)
		if err != nil {
			return errors.Wrap(err, "invalid prefix")
		}
		if err := dd.EnsureNetwork(network.Name); err != nil {
			return errors.Wrap(err, "error creating network directory")
		}

		gohan.Config.Prefix = dd.NetworkPath(network.Name)
		gohan.Config.Network = network
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&prefix, "prefix", "~/.gohan", "Sets gohan's data directory")
	rootCmd.PersistentFlags().StringVarP(&network, "network", "n", "main", "Set's gohan's network")
	rootCmd.PersistentFlags().StringVarP(&walletURL, "wallet-url", "u", "", "Sets a custom node RPC server url")
	rootCmd.PersistentFlags().StringVarP(&accountID, "account-id", "a", "default", "Sets the account ID")
	rootCmd.PersistentFlags().StringVar(&walletAPIKey, "api-key", "", "Sets the wallet's API key.")
	rootCmd.PersistentFlags().StringVar(&nodeAPIKey, "node-api-key", "", "Sets the Handshake full node's API key.")
	rootCmd.PersistentFlags().StringVar(&nodeURL, "node-url", "", "Sets an alternate URL to the Handshake full node.")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
