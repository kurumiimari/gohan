package wallet
//
//import (
//	"fmt"
//	"github.com/kurumiimari/gohan/chain"
//	"github.com/kurumiimari/gohan/walletdb"
//	"github.com/stretchr/testify/require"
//	"testing"
//)
//
//func setupAddrManager(t *testing.T) (*AddrManager, *walletdb.Engine, func()) {
//	engine, cleanupEngine := setupEngine(t)
//	kd := setupKeyDeriver(t)
//	require.NoError(t, kd.Unlock(TestPassword))
//	xPub, err := kd.Derive(chain.Derivation{0})
//	require.NoError(t, err)
//	ring := NewAccountKeyring(kd, xPub, 0)
//	addrBloom := NewAddrBloomFromAddrs(nil)
//	addrManager := NewAddrManager(
//		chain.NetworkMain,
//		ring,
//		addrBloom,
//		"testing/default",
//		[2]uint32{0, 0},
//		[2]uint32{AddrLookahead - 1, AddrLookahead - 1},
//	)
//
//	for i := 0; i < AddrLookahead; i++ {
//		change, err := ring.DeriveAddress(chain.Derivation{chain.ChangeBranch, uint32(i)})
//		require.NoError(t, err)
//		recv, err := ring.DeriveAddress(chain.Derivation{chain.ReceiveBranch, uint32(i)})
//		require.NoError(t, err)
//		addrBloom.Add(change.Address)
//		addrBloom.Add(recv.Address)
//	}
//
//	require.NoError(t, engine.Transaction(func(q walletdb.Transactor) error {
//		w := &walletdb.Wallet{
//			Name:      "testing",
//			KDFAlg:    "",
//			EncAlg:    "",
//			KDFOpts:   "",
//			EncOpts:   "",
//			SeedCT:    []byte{0x00},
//			WatchOnly: false,
//		}
//		if _, err := walletdb.CreateWallet(q, w); err != nil {
//			return err
//		}
//		_, err := walletdb.CreateAccount(
//			q,
//			"default",
//			"testing",
//			xPub.String(),
//			addrBloom.Bytes(),
//			NewOutpointBloomFromOutpoints(nil).Bytes(),
//			NewNameBloomFromNames(nil).Bytes(),
//		)
//		if err != nil {
//			return err
//		}
//		return nil
//	}))
//	return addrManager, engine, cleanupEngine
//}
//
//func TestAddrManager_ChangeAddresses(t *testing.T) {
//	mgr, engine, cleanup := setupAddrManager(t)
//	defer cleanup()
//	q := engine.Querier()
//	defer q.Dispose()
//
//	tests := []string{
//		"hs1qltnc9xhdtjvtj3avge3l6k3pq9lr92p33egplv",
//		"hs1qfwdh7jm2hnk4hkeagja6x4hzazezuhgpmk5jn8",
//		"hs1q29rxmjnt9d9ffkks9vlgagjgr0sk82dd4xdrpa",
//	}
//	for i, tt := range tests {
//		der := fmt.Sprintf("m/0'/1/%d", i)
//		t.Run(fmt.Sprintf("derivation path %s", der), func(t *testing.T) {
//			if i == 0 {
//				currAddr, err := mgr.ChangeAddress()
//				require.NoError(t, err)
//				require.Equal(t, tt, currAddr.Address.String(chain.NetworkMain))
//				require.Equal(t, der, currAddr.Derivation.String())
//				return
//			}
//
//			nextAddr, err := mgr.GenChangeAddress(q)
//			require.NoError(t, err)
//			require.Equal(t, tt, nextAddr.Address.String(chain.NetworkMain))
//			require.Equal(t, der, nextAddr.Derivation.String())
//
//			currAddr, err := mgr.ChangeAddress()
//			require.NoError(t, err)
//			require.Equal(t, nextAddr, currAddr)
//		})
//	}
//
//	t.Run("lookahead", func(t *testing.T) {
//		require.EqualValues(t, 251, mgr.ChangeLookahead())
//		require.NoError(t, mgr.IncLookahead(q, chain.ChangeBranch, 1))
//		require.EqualValues(t, 251, mgr.ChangeLookahead())
//
//		require.NoError(t, mgr.IncLookahead(q, chain.ChangeBranch, 7))
//		require.EqualValues(t, 256, mgr.ChangeLookahead())
//
//		addr, err := mgr.ChangeAddress()
//		require.NoError(t, err)
//		require.Equal(t, "hs1q6mxtk30wy6yzkgw6p2490ge2zw4cpek8c8g9mt", addr.Address.String(chain.NetworkMain))
//		require.Equal(t, "m/0'/1/7", addr.Derivation.String())
//	})
//
//	t.Run("bloom", func(t *testing.T) {
//		for i := 0; i < 257; i++ {
//			addr, err := mgr.ring.DeriveAddress(chain.Derivation{chain.ChangeBranch, uint32(i)})
//			require.NoError(t, err)
//			require.True(t, mgr.HasAddress(addr.Address), fmt.Sprintf("missing address index %d", i))
//		}
//	})
//}
//
//func TestAddrManager_RecvAddresses(t *testing.T) {
//	mgr, engine, cleanup := setupAddrManager(t)
//	defer cleanup()
//	q := engine.Querier()
//	defer q.Dispose()
//
//	tests := []string{
//		"hs1qfwrleyunqs3prr5qjn4fxhkxgryrnkupkqmr0u",
//		"hs1qacaj8s3wv8m7ug27wm6mfcnrq8fz5qg82x07yg",
//		"hs1q6na257t4vzew3a4lkgwmafg557w0nqpukma95p",
//	}
//	for i, tt := range tests {
//		der := fmt.Sprintf("m/0'/0/%d", i)
//		t.Run(fmt.Sprintf("derivation path %s", der), func(t *testing.T) {
//			if i == 0 {
//				currAddr, err := mgr.RecvAddress()
//				require.NoError(t, err)
//				require.Equal(t, tt, currAddr.Address.String(chain.NetworkMain))
//				require.Equal(t, der, currAddr.Derivation.String())
//				return
//			}
//
//			nextAddr, err := mgr.GenRecvAddress(q)
//			require.NoError(t, err)
//			require.Equal(t, tt, nextAddr.Address.String(chain.NetworkMain))
//			require.Equal(t, der, nextAddr.Derivation.String())
//
//			currAddr, err := mgr.RecvAddress()
//			require.NoError(t, err)
//			require.Equal(t, nextAddr, currAddr)
//		})
//	}
//
//	t.Run("lookahead", func(t *testing.T) {
//		require.EqualValues(t, 251, mgr.RecvLookahead())
//		require.NoError(t, mgr.IncLookahead(q, chain.ReceiveBranch, 1))
//		require.EqualValues(t, 251, mgr.RecvLookahead())
//
//		require.NoError(t, mgr.IncLookahead(q, chain.ReceiveBranch, 7))
//		require.EqualValues(t, 256, mgr.RecvLookahead())
//
//		addr, err := mgr.RecvAddress()
//		require.NoError(t, err)
//		require.Equal(t, "hs1qd5nt64mxnmwaqpzg4u2gn2q98krn3u38zkmkj4", addr.Address.String(chain.NetworkMain))
//		require.Equal(t, "m/0'/0/7", addr.Derivation.String())
//	})
//
//	t.Run("bloom", func(t *testing.T) {
//		for i := 0; i < 257; i++ {
//			addr, err := mgr.ring.DeriveAddress(chain.Derivation{chain.ReceiveBranch, uint32(i)})
//			require.NoError(t, err)
//			require.True(t, mgr.HasAddress(addr.Address), fmt.Sprintf("missing address index %d", i))
//		}
//	})
//}
