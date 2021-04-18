package chain

import (
	"github.com/kurumiimari/gohan/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsNameValid(t *testing.T) {
	tests := []struct {
		tName string
		name  string
	}{
		{
			"unicode characters",
			"你好",
		},
		{
			"non-printable characters",
			"honkety\u200bhonk",
		},
		{
			"ends with dash",
			"foobar-",
		},
		{
			"starts with dash",
			"-foobar",
		},
		{
			"bad character",
			"hello.",
		},
		{
			"blacklisted",
			"localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.tName, func(t *testing.T) {
			require.False(t, IsNameValid(tt.name))
		})
	}
}

func TestIsNameReserved(t *testing.T) {
	tests := []struct {
		name   string
		hash   string
		target string
		value  uint64
		root   bool
	}{
		{
			"twitter",
			"525ce500322a0f4c91070eb73829b9d96b2e70d964905fa88c8b20ea573029ea",
			"twitter.com.",
			630133143116,
			false,
		},
		{
			"craigslist",
			"4475619b1fc842831f9af645b268fcd49b20113060f97b9fc49355a69bd0413a",
			"craigslist.org.",
			503513487,
			false,
		},
		{
			"google",
			"6292be73bdfdc4ea12bdf3018c8c553d3022b37601bb2b19153c8804bdf8da15",
			"google.",
			660214983416,
			true,
		},
		{
			"eth",
			"4b3cdfda85c576e43c848d43fdf8e901d8d02553fec8ee56289d10b8dc47d997",
			"eth.ens.domains.",
			136503513487,
			false,
		},
		{
			"kp",
			"4707196b22054788dd1f05a16efb1ff54ed2ddbcd338d4bfc650e72e1829f694",
			"kp.",
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.True(t, IsNameReserved(NetworkMain, 1, tt.name))
			reservation := ReservedNameInfo(tt.name)
			require.Equal(t, tt.name, reservation.Name)
			testutil.RequireEqualHexBytes(t, tt.hash, reservation.Hash)
			require.Equal(t, tt.target, reservation.Target)
			require.Equal(t, tt.value, reservation.Value)
			require.Equal(t, tt.root, reservation.Root)
		})
	}
}
