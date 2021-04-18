package cmd

import (
	"fmt"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"syscall"
)

var (
	importCmdWatchOnly bool
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Imports a wallet",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClient()
		if err != nil {
			return err
		}

		fmt.Print("Please enter a password to encrypt your wallet: ")
		// need the cast below for it to compile on windows
		pwB, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println("")
		if err != nil {
			return errors.Wrap(err, "error reading password")
		}
		password := string(pwB)

		if importCmdWatchOnly {
			return handleWatchOnly(client, password, walletID)
		}
		return handleMnemonic(client, password, walletID)
	},
}

func handleMnemonic(client *api.Client, password, name string) error {
	fmt.Print("Please paste in your mnemonic:")
	// need the cast below for it to compile on windows
	mnemonicB, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println("")
	if err != nil {
		return errors.Wrap(err, "error reading mnemonic")
	}

	fmt.Print("Creating wallet... ")
	_, err = client.CreateWallet(&api.CreateWalletReq{
		Name:     name,
		Mnemonic: string(mnemonicB),
		Password: password,
	})
	if err != nil {
		return errors.Wrap(err, "error creating wallet")
	}

	fmt.Println("Done.")
	return nil
}

func handleWatchOnly(client *api.Client, password, name string) error {
	fmt.Print("Please paste in your xpub: ")
	xPubB, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println("")
	if err != nil {
		return errors.Wrap(err, "error reading xpub")
	}

	fmt.Print("Creating wallet... ")
	_, err = client.CreateWallet(&api.CreateWalletReq{
		Name:     name,
		XPub:     string(xPubB),
		Password: password,
	})
	if err != nil {
		return errors.Wrap(err, "error creating wallet")
	}

	fmt.Println("Done.")
	return nil
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.PersistentFlags().BoolVar(&importCmdWatchOnly, "watch-only", false, "Whether this wallet is watch-only.")
}
