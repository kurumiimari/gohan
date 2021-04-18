package wallet

import (
	"github.com/kurumiimari/gohan/walletdb"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func setupEngine(t *testing.T) (*walletdb.Engine, func()) {
	dirName, err := ioutil.TempDir("", "walletdb_*")
	require.NoError(t, err)

	engine, err := walletdb.NewEngine(dirName)
	require.NoError(t, err)
	require.NoError(t, walletdb.MigrateDB(engine))
	return engine, func() {
		require.NoError(t, os.RemoveAll(dirName))
	}
}
