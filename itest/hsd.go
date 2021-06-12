package itest

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/client"
	"github.com/kurumiimari/gohan/log"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"sync/atomic"
	"time"
)

var hsdLogger = log.ModuleLogger("hsd")

type HSD struct {
	network       *chain.Network
	quitC         chan struct{}
	didQuitC      chan struct{}
	chainDataPath string
	prefix        string
	cmd           *exec.Cmd
	alive         int32
	Client        *client.NodeRPCClient
}

func NewHSD(network *chain.Network) *HSD {
	return &HSD{
		network:  network,
		quitC:    make(chan struct{}),
		didQuitC: make(chan struct{}, 1),
	}
}

func NewHSDWithChainData(network *chain.Network, chainDataPath string) *HSD {
	hsd := NewHSD(network)
	hsd.chainDataPath = chainDataPath
	return hsd
}

func (h *HSD) Start() error {
	prefix, err := ioutil.TempDir("", "gohan-hsd")
	if err != nil {
		return err
	}

	if h.chainDataPath != "" {
		f, err := os.Open(h.chainDataPath)
		if err != nil {
			return err
		}
		defer f.Close()
		gr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		tr := tar.NewReader(gr)
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			p := path.Join(prefix, "regtest", header.Name)
			switch header.Typeflag {
			case tar.TypeDir:
				if header.Name == "./" {
					continue
				}

				if err := os.MkdirAll(p, 0755); err != nil {
					return err
				}
			case tar.TypeReg:
				outFile, err := os.Create(p)
				if err != nil {
					return err
				}
				if _, err := io.Copy(outFile, tr); err != nil {
					return err
				}
				outFile.Close()
			}
		}
	}

	h.prefix = prefix
	h.cmd = exec.Command(
		"hsd",
		"--no-wallet",
		"--index-tx",
		"--log-level=error",
		"--workers=false",
		fmt.Sprintf("--network=%s", h.network.Name),
		fmt.Sprintf("--prefix=%s", h.prefix),
	)
	h.cmd.Stdout = os.Stdout
	h.cmd.Stderr = os.Stderr

	if err := h.cmd.Start(); err != nil {
		os.RemoveAll(h.prefix)
		return errors.Wrap(err, "error starting hsd")
	}

	h.Client = client.NewNodeClient(fmt.Sprintf("http://localhost:%d", h.network.NodePort), "")
	for i := 0; i < 4; i++ {
		if i == 3 {
			return errors.New("hsd never started")
		}

		_, err := h.Client.GetInfo()
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	hsdLogger.Info("started hsd", "prefix", prefix)
	atomic.SwapInt32(&h.alive, 1)

	go func() {
		<-h.quitC
		h.cmd.Process.Kill()
		close(h.quitC)
	}()

	go func() {
		h.cmd.Wait()
		hsdLogger.Info("shut down hsd")
		os.RemoveAll(h.prefix)
		h.didQuitC <- struct{}{}
		atomic.SwapInt32(&h.alive, 0)
	}()

	return nil
}

func (h *HSD) Stop() {
	h.quitC <- struct{}{}
	<-h.didQuitC
}

func (h *HSD) Alive() bool {
	return atomic.LoadInt32(&h.alive) == 1
}
