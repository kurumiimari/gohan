package cmd

import "github.com/spf13/cobra"

var walletsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Lists all accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClient()
		if err != nil {
			return err
		}
		wallets, err := client.GetAccounts()
		if err != nil {
			return err
		}
		return printJSON(wallets)
	},
}

func init() {
	rootCmd.AddCommand(walletsCmd)
}
