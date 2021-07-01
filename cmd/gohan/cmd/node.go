package cmd

import (
	"github.com/kurumiimari/gohan"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/spf13/cobra"
	"gopkg.in/tomb.v2"
	"os"
	"os/signal"
	"syscall"
)

var (
	walletAPIKey string
	nodeAPIKey   string
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Returns status information about the wallet node",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClient()
		if err != nil {
			return err
		}
		status, err := client.Status()
		if err != nil {
			return err
		}
		return printJSON(status)
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the gohan daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		tmb := new(tomb.Tomb)

		go func() {
			sigC := make(chan os.Signal, 1)
			signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)
			select {
			case sig := <-sigC:
				cmdLogger.Info("caught signal, shutting down", "signal", sig.String())
				tmb.Kill(nil)
				return
			case <-tmb.Dying():
				return
			}
		}()

		return api.Start(tmb, gohan.Config.Network, gohan.Config.Prefix, walletAPIKey, nodeAPIKey, nodeURL)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(startCmd)
}
