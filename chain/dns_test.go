package chain

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
)

func TestResource_JSON(t *testing.T) {
	jsonData, err := ioutil.ReadFile("testdata/resource.json")
	require.NoError(t, err)

	resource := new(Resource)
	require.NoError(t, json.Unmarshal(jsonData, resource))
	marsh, err := json.MarshalIndent(resource, "", "  ")
	require.NoError(t, err)
	require.Equal(t, jsonData, marsh)
}
