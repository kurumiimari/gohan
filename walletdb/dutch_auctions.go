package walletdb

import (
	"database/sql"
	"github.com/kurumiimari/gohan/chain"
	"github.com/pkg/errors"
)

const (
	dutchAuctionSelect = `
SELECT 
	id,
	dutch_auction_listings.account_id,
	name,
	transfer_tx_hash,
	transfer_out_idx,
	finalize_tx_hash,
	finalize_out_idx,
	fill_tx_hash,
	fill_out_idx,
	cancel_transfer_tx_hash,
	cancel_transfer_out_idx,
	cancel_finalize_tx_hash,
	cancel_finalize_out_idx,
	lock_address,
	payment_address,
	fee_address,
	lock_time,
	start_price,
	end_price,
	fee_rate_percent,
	num_decrements,
	decrement_duration_secs
FROM dutch_auction_listings
JOIN addresses AS lock_addr ON lock_addr.address = dutch_auction_listings.lock_address 
`
)

type DutchAuctionStatus string

type DutchAuction struct {
	ID                     int
	AccountID              string
	Name                   string
	TransferOutpoint       *chain.Outpoint
	FinalizeOutpoint       *chain.Outpoint
	FillOutpoint           *chain.Outpoint
	CancelTransferOutpoint *chain.Outpoint
	CancelFinalizeOutpoint *chain.Outpoint
	LockAddress            *chain.Address
	LockAddressIndex       uint32
	PaymentAddress         *chain.Address
	FeeAddress             *chain.Address
	LockTime               uint32
	StartPrice             uint64
	EndPrice               uint64
	FeeRatePercent         float64
	NumDecrements          int
	DecrementDurationSecs  int
}

func TransferDutchAuctionListing(
	tx Transactor,
	accountID string,
	name string,
	transferOutpoint *chain.Outpoint,
	listingAddress *chain.Address,
) error {
	_, err := tx.Exec(`
INSERT INTO dutch_auction_listings(
	account_id,
	name, 
	transfer_listing_tx_hash,
	transfer_listing_out_idx, 
	listing_address
) VALUES(?, ?, ?, ?, ?) ON CONFLICT (transfer_listing_tx_hash, transfer_listing_out_idx) DO NOTHING
`,
		accountID,
		name,
		transferOutpoint.Hash.String(),
		transferOutpoint.Index,
		listingAddress,
	)
	return errors.WithStack(err)
}

func GetDutchAuctionByName(tx Transactor, accountID string, name string) (*DutchAuction, error) {
	row := tx.QueryRow(
		dutchAuctionQuery(`
WHERE dutch_auction_listings.account_id = ? 
AND name = ? 
AND finalize_tx_hash IS NOT NULL 
AND fill_tx_hash IS NULL 
AND cancel_transfer_tx_hash IS NULL
`),
		accountID,
		name,
	)
	if err := row.Err(); err != nil {
		return nil, errors.WithStack(err)
	}
	return scanDutchAuctionRow(row)
}

func FinalizeDutchAuctionListing(
	tx Transactor,
	transferOutpoint *chain.Outpoint,
	finalizeOutpoint *chain.Outpoint,
) error {
	_, err := tx.Exec(`
UPDATE dutch_auction_listings
SET finalize_listing_tx_hash = ?, finalize_listing_out_idx = ?
WHERE transfer_listing_tx_hash = ? AND transfer_listing_out_idx = ?
`,
		finalizeOutpoint.Hash.String(),
		finalizeOutpoint.Index,
		transferOutpoint.Hash.String(),
		transferOutpoint.Index,
	)
	return errors.WithStack(err)
}

func UpdateDutchAuctionListingParams(
	tx Transactor,
	name string,
	paymentAddress *chain.Address,
	lockTime uint32,
	startPrice,
	endPrice uint64,
	feeAddress *chain.Address,
	feePercent float64,
	numDecrements int,
	decrementDurationSecs int64,
) error {
	_, err := tx.Exec(`
UPDATE dutch_auction_listings SET
	payment_address = ?,
	lock_time = ?,
	start_price = ?,
	end_price = ?,
	fee_address = ?,
	fee_percent = ?,
	num_decrements = ?,
	decrement_duration_secs = ?
WHERE name = ?
AND finalize_listing_tx_hash IS NOT NULL
AND fill_tx_hash IS NULL
AND transfer_cancel_tx_hash IS NULL
`,
		paymentAddress,
		lockTime,
		startPrice,
		endPrice,
		feeAddress,
		feePercent,
		numDecrements,
		decrementDurationSecs,
		name,
	)
	return errors.WithStack(err)
}

func scanDutchAuctionRow(scanner Scanner) (*DutchAuction, error) {
	row := new(DutchAuction)
	var xferTxHash string
	var xferOutIdx uint32
	var finalizeTxHash sql.NullString
	var finalizeOutIdx sql.NullInt32
	var fillTxHash sql.NullString
	var fillOutIdx sql.NullInt32
	var cancelXferTxHash sql.NullString
	var cancelXferOutIdx sql.NullInt32
	var cancelFinalizeTxHash sql.NullString
	var cancelFinalizeOutIdx sql.NullInt32
	var lockTime sql.NullInt32

	err := scanner.Scan(
		&row.ID,
		&row.AccountID,
		&row.Name,
		&xferTxHash,
		&xferOutIdx,
		&finalizeTxHash,
		&finalizeOutIdx,
		&fillTxHash,
		&fillOutIdx,
		&cancelXferTxHash,
		&cancelXferOutIdx,
		&cancelFinalizeTxHash,
		&cancelFinalizeOutIdx,
		&row.LockAddress,
		&row.LockAddressIndex,
		&row.PaymentAddress,
		&row.FeeAddress,
		&lockTime,
		&row.StartPrice,
		&row.EndPrice,
		&row.FeeRatePercent,
		&row.NumDecrements,
		&row.DecrementDurationSecs,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	row.TransferOutpoint = scanOutpoint(xferTxHash, xferOutIdx)
	row.FinalizeOutpoint = scanOutpoint(finalizeTxHash.String, uint32(finalizeOutIdx.Int32))
	row.FillOutpoint = scanOutpoint(fillTxHash.String, uint32(fillOutIdx.Int32))
	row.CancelTransferOutpoint = scanOutpoint(cancelXferTxHash.String, uint32(cancelXferOutIdx.Int32))
	row.CancelFinalizeOutpoint = scanOutpoint(cancelFinalizeTxHash.String, uint32(cancelFinalizeOutIdx.Int32))
	row.LockTime = uint32(lockTime.Int32)
	return row, nil
}

func dutchAuctionQuery(suffix string) string {
	return dutchAuctionSelect + " " + suffix
}
