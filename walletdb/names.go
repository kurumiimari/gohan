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
	NameStatusRevoked      NameState = "REVOKED"
)

type NameHistoryType string

const (
	NameActionOpen        NameHistoryType = "OPEN"
	NameActionBid         NameHistoryType = "BID"
	NameActionReveal      NameHistoryType = "REVEAL"
	NameActionRedeem      NameHistoryType = "REDEEM"
	NameActionUpdate      NameHistoryType = "UPDATE"
	NameActionRegister    NameHistoryType = "REGISTER"
	NameActionTransfer    NameHistoryType = "TRANSFER"
	NameActionRevoke      NameHistoryType = "REVOKE"
	NameActionFinalizeOut NameHistoryType = "FINALIZE_OUT"
	NameActionFinalizeIn  NameHistoryType = "FINALIZE_IN"
)

type NameHistory struct {
	AccountID    string
	Name         string
	NameHash     []byte
	Type         NameHistoryType
	TxHash       string
	OutIdx       int
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
	TxHash   string
	OutIdx   int
	BidValue uint64
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

	if entry.NameHash != nil {
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
		entry.TxHash,
		entry.OutIdx,
		entry.Value,
		bidValue,
		parentTxHash,
		parentOutIdx,
	)
	return errors.Wrap(err, "error saving name history")
}

func GetRevealableBids(q Transactor, accountID string, name string, network *chain.Network, revealHeight int) ([]*RevealableBid, error) {
	rows, err := q.Query(`
SELECT name_history.tx_hash, name_history.out_idx, name_history.bid_value FROM name_history
JOIN transactions ON (transactions.hash = name_history.tx_hash AND transactions.account_id = name_history.account_id)
LEFT OUTER JOIN name_history AS child 
ON child.parent_tx_hash = name_history.tx_hash AND child.parent_out_idx = name_history.out_idx
WHERE name_history.account_id = ? 
AND name_history.name = ? 
AND transactions.block_height > ?
AND name_history.bid_value IS NOT NULL
AND child.parent_tx_hash IS NULL
AND name_history.type = 'BID'
`,
		accountID,
		name,
		revealHeight-network.BiddingPeriod,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer rows.Close()
	var bids []*RevealableBid
	for rows.Next() {
		bid := new(RevealableBid)
		err := rows.Scan(
			&bid.TxHash,
			&bid.OutIdx,
			&bid.BidValue,
		)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		bids = append(bids, bid)
	}
	return bids, errors.WithStack(err)
}

func GetRedeemableReveals(q Querier, accountID string, name string, network *chain.Network, renewHeight int) ([]*RedeemableReveal, error) {
	rows, err := q.Query(`
SELECT name_history.tx_hash, name_history.out_idx FROM name_history
JOIN transactions ON (transactions.hash = name_history.tx_hash AND transactions.account_id = name_history.account_id)
LEFT OUTER JOIN name_history AS child 
ON child.parent_tx_hash = name_history.tx_hash AND child.parent_out_idx = name_history.out_idx
WHERE name_history.account_id = ? 
AND name_history.name = ? 
AND transactions.block_height > ?
AND child.parent_tx_hash IS NULL
AND name_history.type = 'REVEAL'
`,
		accountID,
		name,
		renewHeight-network.BiddingPeriod,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error querying redeemable reveals")
	}

	defer rows.Close()
	var revs []*RedeemableReveal
	for rows.Next() {
		rev := new(RedeemableReveal)
		err := rows.Scan(
			&rev.TxHash,
			&rev.OutIdx,
		)
		if err != nil {
			return nil, errors.Wrap(err, "error scanning redeemable reveals")
		}
		revs = append(revs, rev)
	}
	return revs, errors.Wrap(rows.Err(), "error iterating redeemable reveals")
}

func GetNameHistory(q Querier, network *chain.Network, accountID, name string, count, offset int) ([]*RichNameHistoryEntry, error) {
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
		if err := fillRichTransaction(q, accountID, network, entry.Transaction, rawTx); err != nil {
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
JOIN transactions ON (transactions.hash = name_history.tx_hash AND transactions.account_id = name_history.account_id)
LEFT OUTER JOIN name_history AS child 
ON child.parent_tx_hash = name_history.tx_hash AND child.parent_out_idx = name_history.out_idx
WHERE name_history.account_id = ? 
AND child.parent_tx_hash IS NULL
AND name_history.type = 'REVEAL'
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
