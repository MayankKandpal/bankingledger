package repository

import (
	"database/sql"

	"github.com/MayankKandpal/bankingledger/internal/models"
)

func ListLedgerEntries(db *sql.DB, accountID string, limit, offset int) ([]models.LedgerEntry, error) {
	var rows *sql.Rows
	var err error

	if accountID != "" {
		rows, err = db.Query(`
			SELECT id, transfer_id, account_id, amount, created_at
			FROM ledger_entries
			WHERE account_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3`, accountID, limit, offset)
	} else {
		rows, err = db.Query(`
			SELECT id, transfer_id, account_id, amount, created_at
			FROM ledger_entries
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.LedgerEntry
	for rows.Next() {
		var e models.LedgerEntry
		if err := rows.Scan(&e.ID, &e.TransferID, &e.AccountID, &e.Amount, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
