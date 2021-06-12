package itest

import (
	"fmt"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/client"
	"github.com/kurumiimari/gohan/log"
	"github.com/kurumiimari/gohan/wallet/api"
	"github.com/stretchr/testify/require"
	"gopkg.in/tomb.v2"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

const ZeroRegtestAddr = "rs1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqn6kda"
const Mnemonic = "run term hint cram stage surround cup frame flight miracle extend reward twelve cause dragon forum barely uncover iron slot napkin walk cancel acid"

var daemonLogger = log.ModuleLogger("daemon")

func startHSD() *HSD {
	hsd := NewHSD(chain.NetworkRegtest)
	if err := hsd.Start(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return hsd
}

func startHSDWithData(dataPath string) *HSD {
	hsd := NewHSDWithChainData(chain.NetworkRegtest, dataPath)
	if err := hsd.Start(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return hsd
}

func startDaemon(t *testing.T) (*api.Client, func()) {
	prefix, err := ioutil.TempDir("", "gohan-test")
	require.NoError(t, err)
	tmb := new(tomb.Tomb)

	tmb.Go(func() error {
		return api.Start(tmb, chain.NetworkRegtest, prefix, "", "")
	})

	cleanup := func() {
		tmb.Kill(nil)
		<-tmb.Dead()
		require.NoError(t, os.RemoveAll(prefix))
	}

	nodeClient := api.NewClient("http://localhost:14039", "")
	for i := 0; i < 3; i++ {
		_, err := nodeClient.Status()
		if err == nil {
			daemonLogger.Info("started daemon", "prefix", prefix)
			return nodeClient, cleanup
		}

		time.Sleep(500 * time.Millisecond)
	}
	require.NoError(t, tmb.Err())
	return nil, cleanup
}

func awaitHeight(
	t *testing.T,
	apiClient *api.Client,
	accountID string,
	targetHeight int,
) {
	var info *api.AccountGetRes
	var err error
	for i := 0; i < 10; i++ {
		info, err = apiClient.GetAccount(accountID)
		require.NoError(t, err)

		if info.RescanHeight == targetHeight {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.Equal(t, targetHeight, info.RescanHeight)
}

func mineTo(
	t *testing.T,
	nodeClient *client.NodeRPCClient,
	apiClient *api.Client,
	count int,
	address string,
) {
	require.NoError(t, nodeClient.GenerateToAddress(count, address))
	require.NoError(t, apiClient.PollBlock())
}
