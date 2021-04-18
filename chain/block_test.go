package chain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/kurumiimari/gohan/testutil"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
)

func readTestData(t *testing.T, name string) []byte {
	hexData, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s.txt", name))
	require.NoError(t, err)
	blockData, err := hex.DecodeString(string(hexData))
	require.NoError(t, err)
	return blockData
}

func readBlockData(t *testing.T, network string, n int) []byte {
	return readTestData(t, fmt.Sprintf("block_%s_%d", network, n))
}

func TestDecoding_MainnetBlock10000(t *testing.T) {
	blockData := readBlockData(t, "main", 10000)
	block := new(Block)
	rLen, err := block.ReadFrom(bytes.NewReader(blockData))
	require.NoError(t, err)
	require.EqualValues(t, len(blockData), rLen)
	require.EqualValues(t, 16552839, block.Nonce)
	require.EqualValues(t, 1586453348, block.Time)
	testutil.RequireEqualHexBytes(t, "00000000000000d4cc864a15c4edfed5deda15e92a5f1eafb009320aac337afc", block.PrevHash)
	testutil.RequireEqualHexBytes(t, "32521938a5a185fb86419b69f68facda957b583a3cf2e17fd51b891a489edd29", block.TreeRoot)
	testutil.RequireEqualHexBytes(t, "02cc316867efb14200000000000000000000000000000000", block.ExtraNonce)
	testutil.RequireEqualHexBytes(t, "0000000000000000000000000000000000000000000000000000000000000000", block.ReservedRoot)
	testutil.RequireEqualHexBytes(t, "95b8688e7eb1fbf969743c696834f292ca2f5df1223b20d6754383220b1d1149", block.WitnessRoot)
	testutil.RequireEqualHexBytes(t, "8853c5240d3893dcf969419dfea7194689e8178d2ed6758b10f4c9f53106ebe9", block.MerkleRoot)
	require.EqualValues(t, 0, block.Version)
	require.EqualValues(t, 0x1a027ff5, block.Bits)
	testutil.RequireEqualHexBytes(t, "0000000000000000000000000000000000000000000000000000000000000000", block.Mask)
	require.Equal(t, 6, len(block.Transactions))
}
