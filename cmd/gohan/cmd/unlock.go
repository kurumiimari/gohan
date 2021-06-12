package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"syscall"
)

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlocks a wallet",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClient()
		if err != nil {
			return err
		}

		fmt.Print("Please enter your password: ")
		// need the cast below for it to compile on windows
		pwB, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println("")
		if err != nil {
			return errors.Wrap(err, "error reading password")
		}

		err = client.Unlock(walletID, string(pwB))
		if err == nil {
			fmt.Println("Wallet unlocked.")
		}
		return err
	},
}

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Locks a wallet",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClient()
		if err != nil {
			return err
		}
		err = client.Lock(walletID)
		if err == nil {
			fmt.Println("Wallet locked.")
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(unlockCmd)
	rootCmd.AddCommand(lockCmd)
}
