package cmd

import (
	"fmt"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"syscall"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a wallet",
	Args:  cobra.MaximumNArgs(1),
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

		fmt.Println("Creating wallet...")

		res, err := client.CreateWallet(&api.CreateWalletReq{
			Name:     walletID,
			Password: string(pwB),
		})
		if err != nil {
			return errors.Wrap(err, "error creating wallet")
		}

		fmt.Println("Your wallet has been successfully created. Please take note of your seed phrase below.")
		fmt.Println("STORE YOUR SEED PHRASE SECURELY. IT WILL NOT BE SHOWN AGAIN.")
		fmt.Println("")
		fmt.Println(*res.Mnemonic)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
