package wallet

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
)

type DataDir struct {
	prefix string
	mtx    sync.Mutex
}

func NewDataDir(prefix string) (*DataDir, error) {
	res := &DataDir{
		prefix: prefix,
	}
	if err := res.createPrefix(); err != nil {
		return nil, errors.Wrap(err, "error creating prefix")
	}
	return res, nil
}

func (d *DataDir) EnsureNetwork(networkName string) error {
	err := d.ensureDir(path.Join(d.prefix, networkName))
	if err != nil {
		return errors.Wrap(err, "error ensuring network directory")
	}
	return nil
}

func (d *DataDir) ListWallets(networkName string) ([]string, error) {
	files, err := ioutil.ReadDir(d.networkDir(networkName))
	if err != nil {
		return nil, errors.Wrap(err, "error listing network directory")
	}

	out := make([]string, 0)
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		out = append(out, f.Name())
	}
	return out, nil
}

func (d *DataDir) NetworkPath(networkName string) string {
	return d.networkDir(networkName)
}

func (d *DataDir) ensureDir(dirPath string) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	dirExists, err := d.dirExists(dirPath)
	if err != nil {
		return errors.Wrap(err, "error reading network directory")
	}
	if dirExists {
		return nil
	}
	if err := os.Mkdir(dirPath, 0o700); err != nil {
		return errors.Wrap(err, "error creating network directory")
	}
	return nil
}

func (d *DataDir) createPrefix() error {
	if strings.HasPrefix(d.prefix, "~") {
		hd, err := os.UserHomeDir()
		if err != nil {
			return errors.Wrap(err, "error reading home directory")
		}
		d.prefix = strings.Replace(d.prefix, "~", hd, 1)
	}

	if err := d.ensureDir(d.prefix); err != nil {
		return errors.Wrap(err, "error opening prefix")
	}
	return nil
}

func (d *DataDir) dirExists(path string) (bool, error) {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "directory read error")
	}
	if !stat.IsDir() {
		return false, errors.New("not a directory")
	}
	return true, nil
}

func (d *DataDir) networkDir(networkName string) string {
	return path.Join(d.prefix, networkName)
}

func (d *DataDir) walletDir(networkName, walletName string) string {
	return path.Join(d.prefix, networkName, walletName)
}
