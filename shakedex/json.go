package shakedex

import (
	"encoding/json"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/gcrypto"
	"github.com/kurumiimari/gohan/gjson"
	"github.com/pkg/errors"
)

type dutchAuctionJSON struct {
	Name           string                 `json:"name"`
	LockingTxHash  gjson.ByteString       `json:"locking_tx_hash"`
	LockingOutIdx  uint32                 `json:"locking_out_idx"`
	PublicKey      gjson.ByteString       `json:"public_key"`
	PaymentAddress *chain.Address         `json:"payment_address"`
	FeeAddress     *chain.Address         `json:"fee_address"`
	Bids           []*dutchAuctionBidJSON `json:"bids"`
}

type dutchAuctionBidJSON struct {
	Value     uint64           `json:"value"`
	LockTime  uint32           `json:"lock_time"`
	Fee       uint64           `json:"fee"`
	Signature gjson.ByteString `json:"signature"`
}

func (d *DutchAuction) MarshalJSON() ([]byte, error) {
	bids := make([]*dutchAuctionBidJSON, len(d.Bids))
	for i := 0; i < len(d.Bids); i++ {
		bid := d.Bids[i]
		bids[i] = &dutchAuctionBidJSON{
			Value:     bid.Value,
			LockTime:  bid.LockTime,
			Fee:       bid.Fee,
			Signature: bid.Signature,
		}
	}

	return json.Marshal(&dutchAuctionJSON{
		Name:           d.Name,
		LockingTxHash:  gjson.ByteString(d.LockingOutpoint.Hash),
		LockingOutIdx:  d.LockingOutpoint.Index,
		PublicKey:      d.PublicKey,
		PaymentAddress: d.PaymentAddress,
		FeeAddress:     d.FeeAddress,
		Bids:           bids,
	})
}

func (d *DutchAuction) UnmarshalJSON(b []byte) error {
	j := new(dutchAuctionJSON)
	if err := json.Unmarshal(b, j); err != nil {
		return errors.WithStack(err)
	}

	bids := make([]*DutchAuctionBid, len(j.Bids))
	for i := 0; i < len(j.Bids); i++ {
		bid := j.Bids[i]
		bids[i] = &DutchAuctionBid{
			Value:     bid.Value,
			LockTime:  bid.LockTime,
			Fee:       bid.Fee,
			Signature: bid.Signature,
		}
	}

	*d = DutchAuction{
		Name: j.Name,
		LockingOutpoint: &chain.Outpoint{
			Hash:  gcrypto.Hash(j.LockingTxHash),
			Index: j.LockingOutIdx,
		},
		PublicKey:      j.PublicKey,
		PaymentAddress: j.PaymentAddress,
		FeeAddress:     j.FeeAddress,
		Bids:           bids,
	}
	return nil
}
