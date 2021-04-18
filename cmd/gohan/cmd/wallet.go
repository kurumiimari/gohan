package cmd

import "github.com/spf13/cobra"

var walletsCmd = &cobra.Command{
	Use:   "wallets",
	Short: "Lists all wallets",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClient()
		if err != nil {
			return err
		}
		wallets, err := client.Wallets()
		if err != nil {
			return err
		}
		return printJSON(wallets)
	},
}

var walletAccountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Lists a wallet's accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClient()
		if err != nil {
			return err
		}
		accounts, err := client.GetAccounts(walletID)
		if err != nil {
			return err
		}
		return printJSON(accounts)
	},
}

func init() {
	rootCmd.AddCommand(walletsCmd)
	rootCmd.AddCommand(walletAccountsCmd)
}
