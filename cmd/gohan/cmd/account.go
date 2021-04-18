package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/kurumiimari/gohan/chain"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"strconv"
)

var (
	walletID   string
	accountID  string
	createOnly bool
)

var accountInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Gets information about an account",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.GetAccount(walletID, accountID)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountTxsCmd = &cobra.Command{
	Use:   "transactions <page> <per-page>",
	Short: "Lists transactions for an account",
	Aliases: []string{
		"txs",
	},
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var page int
		var perPage int
		switch len(args) {
		case 0:
			page = 1
			perPage = 50
		case 1:
			page = intArg(args[0], 1)
			perPage = 50
		case 2:
			page = intArg(args[0], 1)
			perPage = intArg(args[1], 50)
		}

		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.GetAccountTransactions(walletID, accountID, perPage, (page-1)*perPage)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountNamesCmd = &cobra.Command{
	Use:   "names",
	Short: "Lists names for an account",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.GetNames(walletID, accountID)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountNameHistoryCmd = &cobra.Command{
	Use:   "name-history <name>",
	Short: "Lists history for a name belonging to an account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.GetName(walletID, accountID, name)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountSendCmd = &cobra.Command{
	Use:   "send <recipient-address> <amount-whole-hns> [override-fee-rate-subunits]",
	Short: "Sends funds",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := chain.NewAddressFromBech32(args[0])
		if err != nil {
			return errors.New("invalid recipient address")
		}

		amountHNS, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return errors.New("invalid amount")
		}

		var feeRate uint64
		if len(args) == 3 {
			var err error
			feeRate, err = strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return errors.New("invalid override fee rate")
			}
		}

		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.Send(walletID, accountID, uint64(amountHNS*1000000), feeRate, args[0], createOnly)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountOpenCmd = &cobra.Command{
	Use:   "open <name> [override-fee-rate-subunits]",
	Short: "Opens a name for bidding",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		var feeRate uint64
		if len(args) == 2 {
			var err error
			feeRate, err = strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return errors.New("invalid override fee rate")
			}
		}

		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.Open(walletID, accountID, name, feeRate, createOnly)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountBidCmd = &cobra.Command{
	Use:   "bid <name> <bid-amount-whole-hns> <blind-amount-whole-hns> [override-fee-rate-subunits]",
	Short: "Sends a bid",
	Args:  cobra.RangeArgs(3, 4),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		bid, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return errors.New("invalid bid amount")
		}

		blind, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return errors.New("invalid blind amount")
		}

		var feeRate uint64
		if len(args) > 3 {
			feeRate, err = strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return errors.New("invalid override fee rate")
			}
		}

		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.Bid(
			walletID,
			accountID,
			name,
			feeRate,
			uint64(bid*1e6),
			uint64(blind*1e6),
			createOnly,
		)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountRevealCmd = &cobra.Command{
	Use:   "reveal <name> [override-fee-rate-subunits]",
	Short: "Sends a reveal",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		var feeRate uint64
		var err error
		if len(args) > 1 {
			feeRate, err = strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return errors.New("invalid override fee rate")
			}
		}

		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.Reveal(walletID, accountID, name, feeRate, createOnly)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountRedeemCmd = &cobra.Command{
	Use:   "redeem <name> [override-fee-rate-subunits]",
	Short: "Sends a redeem",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		var feeRate uint64
		var err error
		if len(args) > 1 {
			feeRate, err = strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return errors.New("invalid override fee rate")
			}
		}

		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.Redeem(walletID, accountID, name, feeRate, createOnly)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountUpdateCmd = &cobra.Command{
	Use:   "update <name> <resource> [override-fee-rate-subunits]",
	Short: "Sends an update",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		var resource *chain.Resource

		if len(args) > 1 {
			resource = new(chain.Resource)
			if err := json.Unmarshal([]byte(args[1]), resource); err != nil {
				return errors.Wrap(err, "invalid resource")
			}
		}

		var feeRate uint64
		var err error
		if len(args) > 2 {
			feeRate, err = strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return errors.New("invalid override fee rate")
			}
		}

		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.Update(walletID, accountID, name, resource, feeRate, createOnly)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountTransferCmd = &cobra.Command{
	Use:   "transfer <name> <recipient> [override-fee-rate-subunits]",
	Short: "Transfers a name",
	Args:  cobra.RangeArgs(2, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		_, err := chain.NewAddressFromBech32(args[1])
		if err != nil {
			return errors.New("invalid recipient address")
		}

		var feeRate uint64
		if len(args) > 2 {
			feeRate, err = strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return errors.New("invalid override fee rate")
			}
		}

		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.Transfer(walletID, accountID, name, args[1], feeRate, createOnly)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountFinalizeCmd = &cobra.Command{
	Use:   "finalize <name> [override-fee-rate-subunits]",
	Short: "Finalizes a transferring name",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		var feeRate uint64
		var err error
		if len(args) > 1 {
			feeRate, err = strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return errors.New("invalid override fee rate")
			}
		}

		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.Finalize(walletID, accountID, name, feeRate, createOnly)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountZapCmd = &cobra.Command{
	Use:   "zap",
	Short: "Zaps pending transactions",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClient()
		if err != nil {
			return err
		}
		if err := client.Zap(walletID, accountID); err != nil {
			return err
		}
		fmt.Println("OK")
		return nil
	},
}

var accountRescanCmd = &cobra.Command{
	Use:   "rescan [height]",
	Short: "Rescans from the provided height. Default to zero",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		heightInt, err := strconv.Atoi(args[0])
		if err != nil {
			return errors.New("invalid height")
		}

		client, err := apiClient()
		if err != nil {
			return err
		}
		if err := client.Rescan(walletID, accountID, heightInt); err != nil {
			return err
		}
		fmt.Printf("Rescanning from block %d.\n", heightInt)
		return nil
	},
}

var accountSignMessageCmd = &cobra.Command{
	Use:   "sign-message [address] [message]",
	Short: "Signs a message with the address's private key",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClient()
		if err != nil {
			return err
		}
		sig, err := client.SignMessage(walletID, accountID, args[0], args[1])
		if err != nil {
			return err
		}
		fmt.Println(sig)
		return nil
	},
}

var accountSignMessageWithNameCmd = &cobra.Command{
	Use:   "sign-message-with-name [name] [message]",
	Short: "Signs a message with a name's address",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := apiClient()
		if err != nil {
			return err
		}
		sig, err := client.SignMessageWithName(walletID, accountID, args[0], args[1])
		if err != nil {
			return err
		}
		fmt.Println(sig)
		return nil
	},
}

var accountUnspentBidsCmd = &cobra.Command{
	Use:   "unspent-bids <count> <offset>",
	Short: "Returns all bids that haven't been revealed yet",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var count int
		var offset int
		switch len(args) {
		case 0:
			count = 50
		case 1:
			count = intArg(args[0], 50)
		case 2:
			count = intArg(args[0], 50)
			offset = intArg(args[1], 0)
		}

		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.UnspentBids(walletID, accountID, count, offset)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var accountUnspentRevealsCmd = &cobra.Command{
	Use:   "unspent-reveals <count> <offset>",
	Short: "Returns all reveals that haven't been redeemed yet",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var count int
		var offset int
		switch len(args) {
		case 0:
			count = 50
		case 1:
			count = intArg(args[0], 50)
		case 2:
			count = intArg(args[0], 50)
			offset = intArg(args[1], 0)
		}

		client, err := apiClient()
		if err != nil {
			return err
		}
		res, err := client.UnspentReveals(walletID, accountID, count, offset)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

func init() {
	rootCmd.AddCommand(accountInfoCmd)
	rootCmd.AddCommand(accountTxsCmd)
	rootCmd.AddCommand(accountNamesCmd)
	rootCmd.AddCommand(accountNameHistoryCmd)
	rootCmd.AddCommand(accountSendCmd)
	rootCmd.AddCommand(accountOpenCmd)
	rootCmd.AddCommand(accountBidCmd)
	rootCmd.AddCommand(accountRevealCmd)
	rootCmd.AddCommand(accountRedeemCmd)
	rootCmd.AddCommand(accountUpdateCmd)
	rootCmd.AddCommand(accountTransferCmd)
	rootCmd.AddCommand(accountFinalizeCmd)
	rootCmd.AddCommand(accountZapCmd)
	rootCmd.AddCommand(accountRescanCmd)
	rootCmd.AddCommand(accountSignMessageCmd)
	rootCmd.AddCommand(accountSignMessageWithNameCmd)
	rootCmd.AddCommand(accountUnspentBidsCmd)
	rootCmd.AddCommand(accountUnspentRevealsCmd)
}
