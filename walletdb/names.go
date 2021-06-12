package walletdb

import (
	"database/sql"
	"encoding/hex"
	"github.com/kurumiimari/gohan/chain"
	"github.com/pkg/errors"
)

type NameState string

const (
	NameStatusOwned        NameState = "OWNED"
	NameStatusUnowned      NameState = "UNOWNED"
	NameStatusTransferring NameState = "TRANSFERRING"
	NameStatusTransferred  NameState = "TRANSFERRED"
	NameStatusAuctioning   NameState = "AUCTIONING"
	NameStatusRevoked      NameState = "REVOKED"
)

type NameHistoryType string

const (
	NameActionOpen                        NameHistoryType = "OPEN"
	NameActionBid                         NameHistoryType = "BID"
	NameActionReveal                      NameHistoryType = "REVEAL"
	NameActionRedeem                      NameHistoryType = "REDEEM"
	NameActionUpdate                      NameHistoryType = "UPDATE"
	NameActionRegister                    NameHistoryType = "REGISTER"
	NameActionTransfer                    NameHistoryType = "TRANSFER"
	NameActionTransferDutchAuctionListing NameHistoryType = "TRANSFER_DUTCH_AUCTION_LISTING"
	NameActionFinalizeDutchAuctionListing NameHistoryType = "FINALIZE_DUTCH_AUCTION_LISTING"
	NameActionTransferDutchAuctionCancel  NameHistoryType = "TRANSFER_DUTCH_AUCTION_CANCEL"
	NameActionFinalizeDutchAuctionCancel  NameHistoryType = "FINALIZE_DUTCH_AUCTION_CANCEL"
	NameActionFillDutchAuction            NameHistoryType = "FILL_DUTCH_AUCTION"
	NameActionTransferFillDutchAuction    NameHistoryType = "TRANSFER_DUTCH_AUCTION_FILL"
	NameActionFinalizeFillDutchAuction    NameHistoryType = "FINALIZE_DUTCH_AUCTION_FILL"
	NameActionRevoke                      NameHistoryType = "REVOKE"
	NameActionFinalizeOut                 NameHistoryType = "FINALIZE_OUT"
	NameActionFinalizeIn                  NameHistoryType = "FINALIZE_IN"
)

type NameHistory struct {
	AccountID    string
	Name         string
	NameHash     []byte
	Type         NameHistoryType
	Outpoint     *chain.Outpoint
	Value        uint64
	BidValue     uint64
	ParentTxHash string
	ParentOutIdx uint32
	Confirmed    bool
}

type Name struct {
	Name   string    `json:"name"`
	Hash   string    `json:"hash"`
	Status NameState `json:"status"`
}

type RevealableBid struct {
	Coin  *Coin
	Value uint64
}

type RedeemableReveal struct {
	TxHash string
	OutIdx int
}

type RevocableTransfer struct {
	TxHash string
	OutIdx int
}

type UnspentBid struct {
	Name        string `json:"name"`
	BlockHeight int    `json:"block_height"`
	Lockup      uint64 `json:"lockup"`
	BidValue    uint64 `json:"bid_value"`
	TxHash      string `json:"tx_hash"`
	OutIdx      int    `json:"out_idx"`
}

type UnspentReveal struct {
	Name        string `json:"name"`
	BlockHeight int    `json:"block_height"`
	Value       uint64 `json:"value"`
	TxHash      string `json:"tx_hash"`
	OutIdx      int    `json:"out_idx"`
}

type RichNameHistoryEntry struct {
	Name         string           `json:"name"`
	Type         NameHistoryType  `json:"type"`
	OutIdx       int              `json:"out_idx"`
	ParentTxHash *string          `json:"parent_tx_hash"`
	ParentOutIdx *int             `json:"parent_out_idx"`
	Transaction  *RichTransaction `json:"transaction"`
}

func GetNames(q Querier, accountID string, count, offset int) ([]*Name, error) {
	rows, err := q.Query(`
SELECT name, hash, status FROM names
WHERE account_id = ?
ORDER BY name ASC
LIMIT ? OFFSET ?
`,
		accountID,
		count,
		offset,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error getting names")
	}

	var names []*Name
	for rows.Next() {
		name := new(Name)
		var status string
		if err := rows.Scan(&name.Name, &name.Hash, &status); err != nil {
			return nil, errors.Wrap(err, "error scanning names")
		}
		name.Status = NameState(status)
		names = append(names, name)
	}
	return names, nil
}

func HasOwnedName(q Transactor, accountID string, name string) (bool, error) {
	row := q.QueryRow("SELECT EXISTS(SELECT 1 FROM names WHERE name = ? and account_id = ? AND status = 'OWNED')", name, accountID)
	if row.Err() != nil {
		return false, errors.Wrap(row.Err(), "error checking for name existence")
	}
	var res bool
	err := row.Scan(&res)
	return res, errors.Wrap(err, "error scanning has name query")
}

func GetNameFromHash(q Transactor, accountID string, nameHash []byte) (string, error) {
	row := q.QueryRow("SELECT name FROM names WHERE account_id = ? AND hash = ?", accountID, hex.EncodeToString(nameHash))
	if row.Err() != nil {
		return "", errors.Wrap(row.Err(), "error getting name from hash")
	}
	var name string
	err := row.Scan(&name)
	return name, errors.Wrap(err, "error scanning name from hash")
}

func UpsertName(tx Transactor, accountID, name string, status NameState) error {
	_, err := tx.Exec(`
INSERT INTO names(
	account_id,
	name,
	hash,
	status
) VALUES (?, ?, ?, ?)
ON CONFLICT (account_id, name) DO UPDATE SET status = ?
`,
		accountID,
		name,
		hex.EncodeToString(chain.HashName(name)),
		string(status),
		string(status),
	)
	return errors.Wrap(err, "error upserting name")
}

func UpsertNameHash(tx Transactor, accountID string, nameHash []byte, status NameState) error {
	_, err := tx.Exec(`
UPDATE names SET status = ? WHERE account_id = ? AND hash = ?
`,
		string(status),
		accountID,
		hex.EncodeToString(nameHash),
	)
	return errors.Wrap(err, "error upserting name hash")
}

func UpdateNameHistory(tx Transactor, entry *NameHistory) error {
	if entry.Name == "" && entry.NameHash == nil {
		panic("must specify either a name or a name hash")
	}

	var bidValue *uint64
	if entry.Type == NameActionBid || entry.Type == NameActionReveal {
		bidValue = &entry.BidValue
	}
	var parentTxHash *string
	var parentOutIdx *uint32
	if entry.ParentTxHash != "" {
		parentTxHash = &entry.ParentTxHash
		parentOutIdx = &entry.ParentOutIdx
	}

	if entry.NameHash != nil && entry.Name == "" {
		name, err := GetNameFromHash(tx, entry.AccountID, entry.NameHash)
		if err != nil {
			return errors.Wrap(err, "error resolving name hash")
		}
		entry.Name = name
	}

	_, err := tx.Exec(`
INSERT INTO name_history (
	account_id,
	name,
	type,
	tx_hash,
	out_idx,
	value,
	bid_value,
	parent_tx_hash,
	parent_out_idx
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (account_id, tx_hash, out_idx) DO NOTHING
`,
		entry.AccountID,
		entry.Name,
		string(entry.Type),
		entry.Outpoint.Hash.String(),
		entry.Outpoint.Index,
		entry.Value,
		bidValue,
		parentTxHash,
		parentOutIdx,
	)
	return errors.WithStack(err)
}

func GetOwnedNameCoin(q Transactor, accountID string, name string) (*Coin, error) {
	row := q.QueryRow(
		coinQuery(`
WHERE coins.account_id = ?
AND coins.name_hash = ?
AND coins.covenant_type IN (?, ?, ?, ?)
AND coins.spending_tx_hash IS NULL
AND coins.type = ?
`),
		accountID,
		chain.HashName(name),
		uint8(chain.CovenantRegister),
		uint8(chain.CovenantUpdate),
		uint8(chain.CovenantRenew),
		uint8(chain.CovenantFinalize),
		CoinTypeDefault,
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}
	return scanCoin(row)
}

func GetRevocableNameCoin(q Transactor, accountID string, name string) (*Coin, error) {
	row := q.QueryRow(
		coinQuery(`
WHERE coins.account_id = ?
AND coins.name_hash = ?
AND coins.covenant_type IN (?, ?, ?, ?, ?)
AND coins.spending_tx_hash IS NULL
AND coins.type = ?
`),
		accountID,
		chain.HashName(name),
		uint8(chain.CovenantRegister),
		uint8(chain.CovenantUpdate),
		uint8(chain.CovenantRenew),
		uint8(chain.CovenantTransfer),
		uint8(chain.CovenantFinalize),
		CoinTypeDefault,
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}
	return scanCoin(row)
}

func GetTransferCoin(q Transactor, accountID string, name string) (*Coin, error) {
	row := q.QueryRow(
		coinQuery(`
WHERE coins.account_id = ?
AND coins.name_hash = ?
AND coins.covenant_type = ?
AND coins.spending_tx_hash IS NULL
AND coins.type = ?
`),
		accountID,
		chain.HashName(name),
		uint8(chain.CovenantTransfer),
		CoinTypeDefault,
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}
	return scanCoin(row)
}

func GetDutchAuctionTransferCoin(q Transactor, accountID string, name string) (*Coin, error) {
	row := q.QueryRow(
		coinQuery(`
WHERE coins.account_id = ?
AND coins.name_hash = ?
AND coins.covenant_type = ?
AND coins.spending_tx_hash IS NULL
AND coins.type = ?
`),
		accountID,
		chain.HashName(name),
		uint8(chain.CovenantTransfer),
		uint8(CoinTypeDutchAuctionListing),
	)
	if row.Err() != nil {
		return nil, errors.WithStack(row.Err())
	}
	return scanCoin(row)
}

func GetRevealableBids(q Transactor, accountID string, name string, network *chain.Network, revealHeight int) ([]*RevealableBid, error) {
	rows, err := q.Query(`
SELECT 
	coins.account_id AS account_id,
	txout.block_hash IS NOT NULL as spent,
	coins.name_hash AS name_hash,
	coins.type AS type,
	txin.block_height AS height,
	coins.value as VALUE,
	coins.address AS address,
	coins.covenant_type AS covenant_type,
	coins.covenant_items AS covenant_items,
	coins.tx_hash AS tx_hash, 
	coins.out_idx AS out_idx,
	coins.coinbase AS coinbase,
	addr.branch AS address_branch,
	addr.idx AS address_index,
	hist.bid_value AS bid_value
FROM coins
INNER JOIN addresses AS addr ON addr.address = coins.address
INNER JOIN transactions AS txin ON txin.account_id = coins.account_id AND txin.hash = coins.tx_hash
INNER JOIN name_history AS hist ON hist.account_id = coins.account_id AND hist.tx_hash = coins.tx_hash AND hist.out_idx = coins.out_idx 
LEFT JOIN transactions AS txout ON txout.account_id = coins.account_id AND txout.hash = coins.spending_tx_hash
WHERE coins.account_id = ? 
AND coins.name_hash = ?
AND txin.block_height > ?
AND hist.bid_value IS NOT NULL
AND coins.spending_tx_hash IS NULL
`,
		accountID,
		chain.HashName(name),
		revealHeight-network.BiddingPeriod,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer rows.Close()
	var bids []*RevealableBid
	for rows.Next() {
		bid := new(RevealableBid)
		coin, err := scanCoin(rows, &bid.Value)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		bid.Coin = coin
		bids = append(bids, bid)
	}
	return bids, errors.WithStack(rows.Err())
}

func GetRedeemableReveals(q Querier, accountID string, name string) ([]*Coin, error) {
	rows, err := q.Query(
		coinQuery(`
WHERE coins.account_id = ?
AND coins.name_hash = ?
AND coins.covenant_type = ?
AND coins.value > 0
AND coins.spending_tx_hash IS NULL
`),
		accountID,
		chain.HashName(name),
		uint8(chain.CovenantReveal),
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer rows.Close()
	var coins []*Coin
	for rows.Next() {
		coin, err := scanCoin(rows)
		if err != nil {
			return nil, err
		}
		coins = append(coins, coin)
	}
	return coins, errors.WithStack(rows.Err())
}

func GetNameHistory(q Querier, accountID, name string, count, offset int) ([]*RichNameHistoryEntry, error) {
	rows, err := q.Query(`
SELECT 
	name_history.name,
	name_history.type,
	name_history.out_idx,
	name_history.parent_tx_hash,
	name_history.parent_out_idx,
	transactions.hash,
	transactions.idx,
	transactions.block_height,
	transactions.block_hash,
	transactions.raw,
	transactions.time
FROM name_history
JOIN transactions ON (transactions.hash = name_history.tx_hash AND transactions.account_id = name_history.account_id) AND transactions.account_id = name_history.account_id
WHERE name_history.account_id = ? AND name_history.name = ?
ORDER BY CASE 
	WHEN transactions.block_height = -1 THEN 2147483647
	ELSE transactions.block_height
	END DESC, transactions.idx ASC
LIMIT ? OFFSET ?
`,
		accountID,
		name,
		count,
		offset,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error getting name history")
	}
	defer rows.Close()

	entries := make([]*RichNameHistoryEntry, 0)
	for rows.Next() {
		entry := new(RichNameHistoryEntry)
		entry.Transaction = new(RichTransaction)
		var parentTxHash sql.NullString
		var parentOutIdx sql.NullInt32
		var rawTx []byte
		err := rows.Scan(
			&entry.Name,
			&entry.Type,
			&entry.OutIdx,
			&parentTxHash,
			&parentOutIdx,
			&entry.Transaction.Hash,
			&entry.Transaction.Index,
			&entry.Transaction.Height,
			&entry.Transaction.Block,
			&rawTx,
			&entry.Transaction.Time,
		)
		if err != nil {
			return nil, errors.Wrap(err, "error scanning name history row")
		}
		if parentTxHash.Valid {
			entry.ParentTxHash = &parentTxHash.String
		}
		if parentOutIdx.Valid {
			v := int(parentOutIdx.Int32)
			entry.ParentOutIdx = &v
		}
		entry.Transaction.Hex = hex.EncodeToString(rawTx)
		if err := fillRichTransaction(q, accountID, entry.Transaction, rawTx); err != nil {
			return nil, errors.Wrap(err, "error filling rich transaction in name history")
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func GetUnspentBids(tx Transactor, accountID string, count, offset int) ([]*UnspentBid, error) {
	rows, err := tx.Query(`
SELECT name_history.tx_hash, name_history.out_idx, name_history.bid_value, name_history.value, name_history.name, transactions.block_height FROM name_history
JOIN transactions ON (transactions.hash = name_history.tx_hash AND transactions.account_id = name_history.account_id)
LEFT OUTER JOIN name_history AS child 
ON child.parent_tx_hash = name_history.tx_hash AND child.parent_out_idx = name_history.out_idx
WHERE name_history.account_id = ? 
AND child.parent_tx_hash IS NULL
AND name_history.type = 'BID'
ORDER BY name_history.name ASC
LIMIT ? OFFSET ?
`,
		accountID,
		count,
		offset,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()

	var unspents []*UnspentBid
	for rows.Next() {
		usb := new(UnspentBid)
		err := rows.Scan(
			&usb.TxHash,
			&usb.OutIdx,
			&usb.BidValue,
			&usb.Lockup,
			&usb.Name,
			&usb.BlockHeight,
		)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		unspents = append(unspents, usb)
	}
	if rows.Err() != nil {
		return nil, errors.WithStack(rows.Err())
	}
	return unspents, err
}

func GetUnspentReveals(tx Transactor, accountID string, count, offset int) ([]*UnspentReveal, error) {
	rows, err := tx.Query(`
SELECT name_history.tx_hash, name_history.out_idx, name_history.value, name_history.name, transactions.block_height FROM name_history
JOIN coins ON (coins.tx_hash = name_history.tx_hash AND coins.out_idx = name_history.out_idx)
JOIN transactions ON (transactions.hash = name_history.tx_hash AND transactions.account_id = name_history.account_id)
LEFT OUTER JOIN name_history AS child 
ON child.parent_tx_hash = name_history.tx_hash AND child.parent_out_idx = name_history.out_idx
WHERE name_history.account_id = ? 
AND child.parent_tx_hash IS NULL
AND name_history.type = 'REVEAL'
AND coins.value > 0
ORDER BY name_history.name ASC
LIMIT ? OFFSET ?
`,
		accountID,
		count,
		offset,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()

	var unspents []*UnspentReveal
	for rows.Next() {
		usb := new(UnspentReveal)
		err := rows.Scan(
			&usb.TxHash,
			&usb.OutIdx,
			&usb.Value,
			&usb.Name,
			&usb.BlockHeight,
		)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		unspents = append(unspents, usb)
	}
	if rows.Err() != nil {
		return nil, errors.WithStack(rows.Err())
	}
	return unspents, err
}
