package api

import (
	"fmt"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/client"
	"github.com/kurumiimari/gohan/wallet"
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/pkg/errors"
	"gopkg.in/tomb.v2"
	"net/http"
)

func Start(tmb *tomb.Tomb, network *chain.Network, prefix, apiKey, nodeAPIKey string) error {
	chain.SetCurrNetwork(network)
	nodeClient := client.NewNodeClient(fmt.Sprintf("http://localhost:%d", network.NodePort), nodeAPIKey)
	engine, err := walletdb.NewEngine(prefix)
	if err != nil {
		return err
	}

	if err := walletdb.MigrateDB(engine); err != nil {
		return err
	}

	bm := wallet.NewBlockMonitor(tmb, nodeClient, engine)
	service := wallet.NewNode(tmb, network, engine, nodeClient, bm)
	if err := service.Start(); err != nil {
		return errors.Wrap(err, "error opening wallets")
	}

	// start blockmonitor after node to make sure that
	// subscribers are all set
	if err := bm.Start(); err != nil {
		return errors.Wrap(err, "error starting block monitor")
	}

	walletAPI := NewAPI(network, service, apiKey)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", network.WalletPort),
		Handler: walletAPI,
	}

	tmb.Go(func() error {
		apiLogger.Info("starting HTTP server", "port", network.WalletPort)
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(http.ErrServerClosed, err) {
			return errors.Wrap(err, "error starting HTTP server")
		}
		return nil
	})

	apiLogger.Info("started wallet")
	<-tmb.Dying()
	srv.Close()
	apiLogger.Info("shut down wallet")
	return tmb.Err()
}
