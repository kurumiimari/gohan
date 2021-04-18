package chain

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

const TestMnemonic = "volume doll flush federal inflict tomato result property total curtain shield aisle"
const AltMnemonic = "run term hint cram stage surround cup frame flight miracle extend reward twelve cause dragon forum barely uncover iron slot napkin walk cancel acid"

func TestMasterExtendedKey(t *testing.T) {
	tests := []struct {
		password string
		network  *Network
		mnemonic string
		path     Derivation
		address  string
		xPub     string
	}{
		{
			"foo",
			NetworkMain,
			TestMnemonic,
			Derivation{
				HardenNode(CoinPurpose),
				HardenNode(NetworkMain.KeyPrefix.CoinType),
				HardenNode(1),
				1,
				10,
			},
			"hs1qzqjj4xdmgfx38t53hz0p2l29kpaklzq8rju9ha",
			"xpub6Ff4vcqhkMKDoKcqQcsqQH1b6HKDRNj2xa5HUBURupr46KQD4CrfVYjSNeXtWtmB43fXZ2KvUfKdgo8rvN8Q3QGiwa8rtuquXzc6xypyhWv",
		},
		{
			"foo",
			NetworkMain,
			TestMnemonic,
			Derivation{
				HardenNode(CoinPurpose),
				HardenNode(NetworkMain.KeyPrefix.CoinType),
				HardenNode(1),
				255,
			},
			"hs1qj4kqqhve6xz5t2uslth79u76v9qv2hqs29e7wr",
			"xpub6DcYyLPjE7t2E6dfEjU9oJBf29pcz5Q1QfXiMTQpuJnHRycbb2YLXBDdECxcn9b5MQYM1gtsHLixg5w83ocEWbrFNzr9Ud29327Y7bx3Sge",
		},
		{
			"foo",
			NetworkMain,
			TestMnemonic,
			Derivation{
				HardenNode(CoinPurpose),
				HardenNode(NetworkMain.KeyPrefix.CoinType),
				HardenNode(0),
			},
			"hs1qsnv4u00944detnt8almdnpmnuynvfvq5z2sphl",
			"xpub6D7KottY8BmBoCnzu8yPuNE83ZDmN8X9abWt8zCFgT6nGVgZwMPGEYKmcAiuqadSXh3aU5LQjZZKqym6gyRX7fdEAMXzd1UmGjcWuA6sta1",
		},
		{
			"",
			NetworkRegtest,
			AltMnemonic,
			Derivation{
				HardenNode(CoinPurpose),
				HardenNode(NetworkRegtest.KeyPrefix.CoinType),
				HardenNode(0),
			},
			"rs1qfcm0jew86vvg4kpkj5ep7jwg8z2v3sylsljcts",
			"rpubKBBUaydwRpVxLcm8YESMRikrSFRG9nsXDquhppVigpKymkS6fhoKxxJa1Ud76TgHUMMrvAvqJXyxkJKjWdmX6uSkQNYKHnuqDnDsLSVyVQnQ",
		},
		{
			"",
			NetworkRegtest,
			AltMnemonic,
			Derivation{
				HardenNode(CoinPurpose),
				HardenNode(NetworkRegtest.KeyPrefix.CoinType),
				HardenNode(0),
				0,
				0,
			},
			"rs1qedtqrtu8eavsl7sepgy3fp336966pqxmyhnquc",
			"rpubKBE1CA75cBARiPUTATW7AW7W3oCv24ffN5HA65jFDQFtr9a1Vsy3oTpFohcNEW8jrmf3jW3Lahrd4aA7fMdFwS8CTFKsj2Qr8cz52DrFeMow",
		},
	}

	for _, tt := range tests {
		t.Run(strings.Replace(tt.path.String(), "/", "-", -1), func(t *testing.T) {
			mk := NewMasterExtendedKeyFromMnemonic(tt.mnemonic, tt.password, tt.network)
			derived := DeriveExtendedKey(mk, tt.path...)
			require.Equal(t, tt.address, derived.Address().String(tt.network))
			require.Equal(t, tt.xPub, derived.PublicString())
		})
	}
}
