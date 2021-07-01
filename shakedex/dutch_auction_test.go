package shakedex

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/kurumiimari/gohan/bio"
	"github.com/kurumiimari/gohan/chain"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

const dummyProof = "{\"name\":\"cakcp\",\"lockingTxHash\":\"00\",\"lockingOutputIdx\":0,\"publicKey\":\"00\",\"paymentAddr\":\"rs1q5aygjrhe6wt0qnhrshwpknfct7xflpa7792h67\",\"feeAddr\":\"rs1qrntg75vjhnl766uk2ylq0zl7u3n6rw8cw7ea8t\",\"data\":[{\"price\":1000,\"lockTime\":50,\"fee\":10,\"signature\":\"00\"}]}"

func TestReadDutchAuctionProof(t *testing.T) {
	t.Parallel()

	f, err := os.OpenFile("testdata/proof.txt", os.O_RDONLY, 0700)
	require.NoError(t, err)
	proof, err := ReadDutchAuctionProof(f)
	require.NoError(t, err)

	t.Run("ok", func(t *testing.T) {
		require.Equal(t, regtestAuction(), proof)
	})

	tests := []struct {
		name string
		body string
	}{
		{
			"no header",
			dummyProof,
		},
		{
			"no body",
			"",
		},
		{
			"invalid header",
			"blargle\n" + dummyProof,
		},
		{
			"invalid proof",
			V1DutchAuctionMagic + "\n{\"bad\": \"juju\"}",
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, tt.name), func(t *testing.T) {
			_, err := ReadDutchAuctionProof(bytes.NewReader([]byte(tt.body)))
			require.Error(t, err)
			require.True(t, errors.Is(err, ErrMalFormedProofFile))
		})
	}
}

func TestDutchAuctionBidVerification(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		require.True(t, regtestAuction().VerifyAllBids(4000000))
	})

	tests := []struct {
		name    string
		mangler func(*DutchAuction)
		ok      bool
	}{
		{
			"invalid bid value",
			func(auction *DutchAuction) {
				auction.Bids[0].Value = 9999
			},
			false,
		},
		{
			"invalid bid address",
			func(auction *DutchAuction) {
				_, err := rand.Read(auction.PaymentAddress.Hash)
				require.NoError(t, err)
			},
			false,
		},
		{
			"invalid sig",
			func(auction *DutchAuction) {
				_, err := rand.Read(auction.Bids[0].Signature)
				require.NoError(t, err)
			},
			false,
		},
		{
			"invalid prevout",
			func(auction *DutchAuction) {
				_, err := rand.Read(auction.LockingOutpoint.Hash)
				require.NoError(t, err)
			},
			false,
		},
		{
			"invalid bid fee",
			func(auction *DutchAuction) {
				auction.Bids[0].Fee = 9999
			},
			// bids are not committed to
			true,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, tt.name), func(t *testing.T) {
			auction := regtestAuction()
			tt.mangler(auction)
			res := auction.VerifyAllBids(4000000)
			if tt.ok {
				require.True(t, res)
			} else {
				require.False(t, res)
			}
		})
	}
}

func TestCreateDutchAuction(t *testing.T) {
	t.Parallel()

	t.Run("invalid start and end bids", func(t *testing.T) {
		_, err := CreateDutchAuction(nil, 0, "foobar", 0, 100, 200, 0, 10, 0, nil, nil, nil)
		require.Error(t, err)
	})

	privKey, _ := btcec.PrivKeyFromBytes(
		btcec.S256(),
		bio.MustDecodeHex("15fc9484e00440b66dd097231c63d6598ccb95e0942be16fa49d14cfed1f1448"),
	)

	payAddr := chain.NewAddressFromHash(chain.ZeroHash)
	feeHash := make([]byte, 20)
	feeHash[0] = 1
	feeAddr := chain.NewAddressFromScript(feeHash)

	lockCoin := &chain.Coin{
		Value: 4000000,
		Prevout: &chain.Outpoint{
			Hash: []byte{
				0x17, 0x0b, 0x24, 0x08, 0xbc, 0x42, 0xe2, 0x37,
				0xe3, 0xd5, 0x0f, 0x05, 0x93, 0x96, 0x00, 0xaf,
				0xac, 0xf2, 0x66, 0xee, 0xe1, 0xec, 0x17, 0xdc,
				0x25, 0xc9, 0xde, 0xb3, 0xfa, 0x93, 0x52, 0x21,
			},
			Index: 0,
		},
	}

	tests := []struct {
		name           string
		start          uint64
		end            uint64
		feeRatePercent float64
		nDecrements    int
		decDuration    time.Duration
		startFee       uint64
		endFee         uint64
		endAt          int64
	}{
		{
			"ten hourly decrements 1000000 to 500000",
			1000000,
			500000,
			0.005,
			10,
			time.Hour,
			5000,
			2500,
			100 + 10*int64(time.Hour/time.Second),
		},
		{
			"ten hourly decrements 999977 to 499977",
			999977,
			499977,
			0.005,
			10,
			time.Hour,
			4999,
			2499,
			100 + 10*int64(time.Hour/time.Second),
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, tt.name), func(t *testing.T) {
			auction, err := CreateDutchAuction(
				lockCoin.Prevout,
				lockCoin.Value,
				"xsqtx",
				100,
				tt.start,
				tt.end,
				tt.feeRatePercent,
				tt.nDecrements,
				tt.decDuration,
				payAddr,
				feeAddr,
				privKey,
			)
			require.NoError(t, err)
			require.True(t, auction.VerifyAllBids(lockCoin.Value))

			require.Equal(t, tt.nDecrements+1, len(auction.Bids))
			require.EqualValues(t, tt.start-tt.startFee, auction.Bids[0].Value)
			require.EqualValues(t, tt.startFee, auction.Bids[0].Fee)
			require.EqualValues(t, tt.end-tt.endFee, auction.Bids[len(auction.Bids)-1].Value)
			require.EqualValues(t, tt.endFee, auction.Bids[len(auction.Bids)-1].Fee)
			require.EqualValues(t, tt.endAt, auction.Bids[len(auction.Bids)-1].LockTime)
		})
	}
}

func regtestAuction() *DutchAuction {
	return &DutchAuction{
		Name: "xsqtx",
		LockingOutpoint: &chain.Outpoint{
			Hash: []byte{
				0x17, 0x0b, 0x24, 0x08, 0xbc, 0x42, 0xe2, 0x37,
				0xe3, 0xd5, 0x0f, 0x05, 0x93, 0x96, 0x00, 0xaf,
				0xac, 0xf2, 0x66, 0xee, 0xe1, 0xec, 0x17, 0xdc,
				0x25, 0xc9, 0xde, 0xb3, 0xfa, 0x93, 0x52, 0x21,
			},
			Index: 0,
		},
		PublicKey: []byte{
			0x02, 0xd0, 0x2c, 0x2a, 0x2b, 0x24, 0x9b, 0xb6,
			0x55, 0x66, 0xe2, 0x27, 0x29, 0xc8, 0xa5, 0x34,
			0x19, 0xb3, 0xb3, 0x7d, 0xb1, 0x97, 0xed, 0xd1,
			0xf1, 0x90, 0xaf, 0x02, 0x6d, 0xc0, 0xee, 0x62,
			0xe6,
		},
		PaymentAddress: &chain.Address{
			Version: 0,
			Hash: []byte{
				0x13, 0x9c, 0xd5, 0x4b, 0xc5, 0xd6, 0x6e, 0xe7,
				0x2b, 0x10, 0x6c, 0xb5, 0x97, 0x10, 0xbc, 0x53,
				0x2e, 0xfe, 0x18, 0x98,
			},
		},
		Bids: []*DutchAuctionBid{
			{
				Value:    1000,
				LockTime: 50,
				Fee:      10,
				Signature: []byte{
					0x2d, 0x2c, 0xfd, 0x30, 0x8d, 0xba, 0xb2, 0xa4,
					0x7d, 0x7b, 0x63, 0xbb, 0x0e, 0x5b, 0xa0, 0xf7,
					0xe0, 0x1a, 0x77, 0x21, 0x65, 0xb1, 0x99, 0xbb,
					0xd5, 0x61, 0x55, 0xf2, 0xdf, 0x1b, 0x26, 0x85,
					0x27, 0xa0, 0x3f, 0x03, 0x30, 0x95, 0xb5, 0x80,
					0xe3, 0x93, 0x87, 0xc5, 0xff, 0x8c, 0xba, 0x86,
					0x02, 0x10, 0xd2, 0x6e, 0xd9, 0x1e, 0xeb, 0x88,
					0x19, 0x47, 0x23, 0x1c, 0x19, 0x3b, 0x8d, 0x36,
					0x84,
				},
			},
			{
				Value:    100,
				LockTime: 100,
				Fee:      20,
				Signature: []byte{
					0xd2, 0xb5, 0xa9, 0x06, 0x7d, 0x0d, 0x5b, 0x88,
					0xe4, 0x90, 0xd8, 0x5e, 0xf3, 0x21, 0xab, 0x0b,
					0x56, 0x9f, 0x7d, 0x3d, 0x9a, 0x24, 0x2a, 0x1e,
					0x65, 0xd5, 0x0f, 0x95, 0x54, 0x0c, 0x7e, 0xe7,
					0x59, 0xa1, 0xf6, 0x6a, 0xee, 0xdd, 0xa1, 0x2c,
					0x9d, 0x58, 0x7a, 0xe7, 0xa3, 0xd6, 0x17, 0x7d,
					0xeb, 0x94, 0xe4, 0x54, 0x4e, 0x83, 0xae, 0x6a,
					0x29, 0x3b, 0x6c, 0xb7, 0xe8, 0x4d, 0x38, 0x22,
					0x84,
				},
			},
		},
		FeeAddress: &chain.Address{
			Version: 0,
			Hash: []byte{
				0x71, 0x41, 0x51, 0x79, 0xbd, 0xf8, 0xeb, 0xa9,
				0xaf, 0x93, 0x4a, 0x1e, 0x41, 0xd6, 0xc3, 0x24,
				0x8b, 0xb8, 0x50, 0x99,
			},
		},
	}
}
