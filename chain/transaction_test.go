package chain

import (
	"bytes"
	"fmt"
	"github.com/kurumiimari/gohan/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func readTXData(t *testing.T, network string, hash string) []byte {
	return readTestData(t, fmt.Sprintf("tx_%s_%s", network, hash))
}

func TestDecoding_SimpleCoinbase(t *testing.T) {
	txData := readTXData(t, "main", "8f99b0037eb07812737aaa1005af85fc4429e20a65f66bf15d148be02abca587")
	tx := new(Transaction)
	rLen, err := tx.ReadFrom(bytes.NewReader(txData))
	require.NoError(t, err)
	require.EqualValues(t, len(txData), rLen)
	require.EqualValues(t, 0, tx.Version)
	require.Equal(t, 1, len(tx.Inputs))
	testutil.RequireEqualJSONFile(
		t,
		"tx_main_8f99b0037eb07812737aaa1005af85fc4429e20a65f66bf15d148be02abca587.json",
		tx,
	)
}
