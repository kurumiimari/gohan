package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/kurumiimari/gohan"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

func apiClient() (*api.Client, error) {
	var url string
	if serverURL == "" {
		url = fmt.Sprintf("http://localhost:%d", gohan.Config.Network.WalletPort)
	} else {
		url = serverURL
	}

	client := api.NewClient(url, walletAPIKey)

	_, err := client.Status()
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return nil, errors.New("connection to gohan refused - did you select the right network?")
		}
		return nil, err
	}

	return client, nil
}

func intArg(in string, deflt int) int {
	out, err := strconv.Atoi(in)
	if err != nil {
		return deflt
	}
	return out
}

func float64Arg(in string, deflt float64) float64 {
	out, err := strconv.ParseFloat(in, 64)
	if err != nil {
		return deflt
	}
	return out
}

func printJSON(in interface{}) error {
	out, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}
