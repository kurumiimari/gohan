package cmd

import (
	"fmt"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/shakedex"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	durationItems = []string{
		"1 hour",
		"3 hours",
		"8 hours",
		"1 day",
		"3 days",
		"5 days",
		"1 week",
		"2 weeks",
	}
	durationConfigs = [][2]int64{
		{5, int64((10 * time.Minute) / time.Second)},
		{12, int64((15 * time.Minute) / time.Second)},
		{8, int64(time.Hour / time.Second)},
		{24, int64(time.Hour / time.Second)},
		{24, int64((3 * time.Hour) / time.Second)},
		{40, int64((3 * time.Hour) / time.Second)},
		{42, int64((4 * time.Hour) / time.Second)},
		{42, int64((8 * time.Hour) / time.Second)},
	}

	shakedexURL string
)

var dutchAuctionsCmd = &cobra.Command{
	Use:   "dutch-auctions",
	Short: "Buy and sell names on ShakeDex using dutch auctions",
}

var transferDutchAuctionListingCmd = &cobra.Command{
	Use:   "transfer-listing [name] <override-fee-rate-subunits>",
	Short: "Transfers a name to a dutch auction locking script",
	Args:  cobra.MinimumNArgs(1),
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

		res, err := client.TransferDutchAuctionListing(accountID, name, feeRate)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var finalizeDutchAuctionListingCmd = &cobra.Command{
	Use:   "finalize-listing [name] <override-fee-rate-subunits>",
	Short: "Finalizes a name to a dutch auction locking script",
	Args:  cobra.MinimumNArgs(1),
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

		res, err := client.FinalizeDutchAuctionListing(accountID, name, feeRate)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var postDutchAuctionListingCmd = &cobra.Command{
	Use:   "post-listing [name]",
	Short: "Interactively posts a dutch auction listing",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		client, err := apiClient()
		if err != nil {
			return err
		}

		_, err = client.GetName(accountID, name)
		if err != nil {
			return err
		}

		startPrice, err := promptFloat("Start Price (whole HNS)")
		if err != nil {
			return err
		}
		endPrice, err := promptFloat("End Price (whole HNS)")
		if err != nil {
			return err
		}
		if startPrice < endPrice {
			return errors.New("start price must be higher than end price")
		}
		durationSel := promptui.Select{
			Label: "Auction duration:",
			Items: durationItems,
		}
		i, _, err := durationSel.Run()
		if err != nil {
			return err
		}

		shouldPost, err := promptBool("Post on ShakeDex web")
		if err != nil {
			return err
		}
		var feeAddr *chain.Address
		var feePercent float64
		sdClient := shakedex.NewClient(shakedexURL)
		if shouldPost {
			feeInfo, err := sdClient.FeeInfo()
			if err != nil {
				return err
			}
			if feeInfo.RatePercent > 0 {
				confirmList, err := promptBool(fmt.Sprintf("A fee of %f%% will be applied. Continue", feeInfo.RatePercent))
				if err != nil {
					return err
				}
				if !confirmList {
					fmt.Println("Aborted.")
					os.Exit(0)
				}

				feeAddr = feeInfo.Address
				feePercent = feeInfo.RatePercent
			}
		}

		var presignLoc string
		shouldWrite, err := promptBool("Do you want to write your presigns to disk")
		if err != nil {
			return err
		}
		if shouldWrite {
			presignLoc, err = promptStr("Where do you want to store your presigns", fmt.Sprintf("./%s.txt", name))
			if err != nil {
				return err
			}
		}

		presigns, err := client.UpdateDutchAuctionListing(accountID, &api.UpdateDutchAuctionListingsReq{
			Name:                  name,
			FeeAddress:            feeAddr,
			StartPrice:            uint64(startPrice * 1e6),
			EndPrice:              uint64(endPrice * 1e6),
			FeePercent:            feePercent,
			NumDecrements:         int(durationConfigs[i][0]),
			DecrementDurationSecs: durationConfigs[i][1],
		})
		if err != nil {
			return err
		}

		if presignLoc != "" {
			if strings.HasPrefix(presignLoc, "~") {
				hd, err := os.UserHomeDir()
				if err != nil {
					return errors.WithStack(err)
				}
				presignLoc = strings.Replace(presignLoc, "~", hd, 1)
			}
			f, err := os.OpenFile(presignLoc, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
			if err != nil {
				return errors.WithStack(err)
			}
			defer f.Close()
			if err := shakedex.WriteDutchAuctionProof(presigns, f); err != nil {
				return errors.WithStack(err)
			}
		}

		if shouldPost {
			fmt.Println("Uploading to ShakeDex Web...")
			if err := sdClient.UploadPresigns(presigns); err != nil {
				return err
			}
		}

		fmt.Println("Done!")
		return nil
	},
}

var transferDutchAuctionFill = &cobra.Command{
	Use:   "transfer-fill [proof-file] <override-fee-rate-subunits>",
	Short: "Fills a dutch auction using a proof file",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var feeRate uint64
		var err error
		if len(args) > 1 {
			feeRate, err = strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return errors.New("invalid override fee rate")
			}
		}

		f, err := os.OpenFile(args[0], os.O_RDONLY, 0)
		if err != nil {
			return err
		}

		auction, err := shakedex.ReadDutchAuctionProof(f)
		if err != nil {
			return err
		}

		client, err := apiClient()
		if err != nil {
			return err
		}

		bestBidI := auction.BestBid(uint32(time.Now().Unix()))
		if bestBidI == -1 {
			return errors.New("no available bids to fulfill")
		}
		bestBid := auction.Bids[bestBidI]

		res, err := client.TransferDutchAuctionFill(accountID, &api.TransferDutchAuctionFillReq{
			Name:             auction.Name,
			LockScriptTxHash: auction.LockingOutpoint.Hash,
			LockScriptOutIdx: auction.LockingOutpoint.Index,
			PaymentAddress:   auction.PaymentAddress,
			FeeAddress:       auction.FeeAddress,
			PublicKey:        auction.PublicKey,
			Signature:        bestBid.Signature,
			LockTime:         bestBid.LockTime,
			Bid:              bestBid.Value,
			AuctionFee:       bestBid.Fee,
			FeeRate:          feeRate,
		})
		if err != nil {
			return err
		}

		return printJSON(res)
	},
}

var finalizeDutchAuctionFillCmd = &cobra.Command{
	Use:   "finalize-fill [name] <override-fee-rate-subunits>",
	Short: "Finalizes a fill",
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

		res, err := client.FinalizeDutchAuctionFill(accountID, name, feeRate)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var transferDutchAuctionCancelCmd = &cobra.Command{
	Use:   "transfer-cancel [name] <override-fee-rate-subunits>",
	Short: "Cancels a listing",
	Args:  cobra.MinimumNArgs(1),
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

		res, err := client.TransferDutchAuctionCancel(accountID, name, feeRate)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

var finalizeDutchAuctionCancelCmd = &cobra.Command{
	Use:   "finalize-cancel [name] <override-fee-rate-subunits>",
	Short: "Finalizes a cancel",
	Args:  cobra.MinimumNArgs(1),
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

		res, err := client.FinalizeDutchAuctionCancel(accountID, name, feeRate)
		if err != nil {
			return err
		}
		return printJSON(res)
	},
}

func promptStr(label string, dflt string) (string, error) {
	prompt := promptui.Prompt{
		Label:   label,
		Default: dflt,
	}
	val, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return val, nil
}

func promptFloat(label string) (float64, error) {
	prompt := promptui.Prompt{
		Label:    label,
		Validate: validateFloat,
	}
	strVal, err := prompt.Run()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strVal, 64)
}

func promptBool(label string) (bool, error) {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}
	_, err := prompt.Run()
	if err == promptui.ErrAbort {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func validateFloat(s string) error {
	_, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return errors.New("must be a number")
	}
	return nil
}

func init() {
	rootCmd.AddCommand(dutchAuctionsCmd)
	dutchAuctionsCmd.PersistentFlags().StringVar(&shakedexURL, "shakedex-url", "https://www.shakedex.com", "URL to ShakeDex Web.")
	dutchAuctionsCmd.AddCommand(transferDutchAuctionListingCmd)
	dutchAuctionsCmd.AddCommand(finalizeDutchAuctionListingCmd)
	dutchAuctionsCmd.AddCommand(postDutchAuctionListingCmd)
	dutchAuctionsCmd.AddCommand(transferDutchAuctionFill)
	dutchAuctionsCmd.AddCommand(finalizeDutchAuctionFillCmd)
	dutchAuctionsCmd.AddCommand(transferDutchAuctionCancelCmd)
	dutchAuctionsCmd.AddCommand(finalizeDutchAuctionCancelCmd)
}
