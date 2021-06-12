package wallet

import (
	"database/sql"
	"github.com/kurumiimari/gohan/chain"
	"github.com/stretchr/testify/require"
	"testing"
)

const Mnemonic = "few derive language prison worth heavy prosper seven bone discover journey lonely sketch success marine robust crew egg fork misery certain drill seminar warrior"

func TestAddrManager(t *testing.T) {
	mk := chain.NewMasterExtendedKeyFromMnemonic(Mnemonic, "", chain.NetworkRegtest)
	derived := chain.DeriveExtendedKey(mk, chain.Derivation{
		chain.HardenNode(chain.CoinPurpose),
		chain.HardenNode(chain.NetworkRegtest.KeyPrefix.CoinType),
		chain.HardenNode(0),
	}...)
	ring := NewAccountKeyring(NewEKPrivateKeyer(mk), derived, chain.NetworkRegtest, 0)
	dTx := new(dummyTransactor)

	tests := []struct {
		name         string
		currIdx      uint32
		lookIdx      int64
		newLookahead uint32
		depth        uint32
		lookahead    uint32
	}{
		{
			"bootstrap",
			0,
			-1,
			0,
			0,
			10,
		},
		{
			"lookahead behind tip",
			0,
			20,
			0,
			0,
			20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bloom := NewAddressBloom()
			mgr := NewAddressManager(ring, bloom, "dummy", chain.ReceiveBranch, tt.currIdx, tt.lookIdx)
			require.NoError(t, mgr.SetAddressIdx(dTx, tt.newLookahead))
			require.EqualValues(t, tt.depth, mgr.Depth())
			require.EqualValues(t, tt.lookahead, mgr.Lookahead())
			require.True(t, ring.Address(chain.ReceiveBranch, mgr.Depth()).Equal(mgr.Address()))
		})
	}
}

type dummyTransactor struct{}

func (d *dummyTransactor) Query(q string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (d *dummyTransactor) QueryRow(q string, args ...interface{}) *sql.Row {
	return nil
}

func (d *dummyTransactor) Exec(q string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}
