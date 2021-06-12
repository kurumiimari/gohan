package testutil

import (
	"encoding/hex"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path"
	"testing"
)

func RequireEqualHexBytes(t *testing.T, exp string, act []byte) {
	require.Equal(t, exp, hex.EncodeToString(act))
}

func RequireEqualJSONFile(t *testing.T, expFilename string, actRaw interface{}) {
	expData, err := ioutil.ReadFile(path.Join("testdata", expFilename))
	require.NoError(t, err)
	var exp interface{}
	require.NoError(t, json.Unmarshal(expData, &exp))

	var act interface{}
	actJ, err := json.MarshalIndent(actRaw, "", "  ")
	//fmt.Println(string(actJ))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(actJ, &act))
	require.Equal(t, exp, act)
}
