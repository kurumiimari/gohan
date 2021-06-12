package gcrypto

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHashJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Hash
		out  string
	}{
		{
			"converts hex values",
			[]byte{0xde, 0xad, 0xbe, 0xef},
			"\"deadbeef\"",
		},
		{
			"handles empty hashes",
			[]byte{},
			"null",
		},
		{
			"handles nil hashes",
			nil,
			"null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j, err := json.Marshal(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.out, string(j))
			var h Hash
			err = json.Unmarshal(j, &h)
			require.NoError(t, err)
			require.True(t, tt.in.Equal(h))
		})
	}
}
