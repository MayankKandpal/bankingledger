package repository

import (
	"database/sql"

	"github.com/lib/pq"
	"github.com/MayankKandpal/bankingledger/internal/models"
	"github.com/shopspring/decimal"
)

// IsUniqueViolation returns true when a Postgres unique constraint is hit.
// Used to detect concurrent duplicate reversals at the DB level.
func IsUniqueViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return false
}

func GetAccountForUpdate(tx *sql.Tx, id string) (models.Account, error) {
	var a models.Account
	var mob sql.NullString
	err := tx.QueryRow(`
		SELECT id, name, mobile, balance, created_at
		FROM accounts
		WHERE id = $1
		FOR UPDATE`,
		id,
	).Scan(&a.ID, &a.Name, &mob, &a.Balance, &a.CreatedAt)
	if mob.Valid {
		a.Mobile = &mob.String
	}
	return a, err
}

func InsertTransfer(tx *sql.Tx, fromID, toID string, amount decimal.Decimal, status string, failureReason *string, fee *decimal.Decimal) (models.Transfer, error) {
	var t models.Transfer
	var fr, rb sql.NullString
	var feeNull decimal.NullDecimal
	err := tx.QueryRow(`
		INSERT INTO transfers (from_account_id, to_account_id, amount, status, failure_reason, fee)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, from_account_id, to_account_id, amount, status, failure_reason, reversed_by, fee, created_at`,
		fromID, toID, amount, status, failureReason, fee,
	).Scan(&t.ID, &t.FromAccountID, &t.ToAccountID, &t.Amount, &t.Status, &fr, &rb, &feeNull, &t.CreatedAt)
	if fr.Valid {
		t.FailureReason = &fr.String
	}
	if rb.Valid {
		t.ReversedBy = &rb.String
	}
	if feeNull.Valid {
		t.Fee = &feeNull.Decimal
	}
	return t, err
}

func InsertLedgerEntry(tx *sql.Tx, transferID, accountID string, amount decimal.Decimal) error {
	_, err := tx.Exec(`
		INSERT INTO ledger_entries (transfer_id, account_id, amount)
		VALUES ($1, $2, $3)`,
		transferID, accountID, amount,
	)
	return err
}

func UpdateAccountBalance(tx *sql.Tx, id string, delta decimal.Decimal) error {
	_, err := tx.Exec(`
		UPDATE accounts SET balance = balance + $1 WHERE id = $2`,
		delta, id,
	)
	return err
}

func InsertAuditLog(tx *sql.Tx, transferID *string, operation, fromID, toID string, amount decimal.Decimal, outcome string, failureReason *string) error {
	_, err := tx.Exec(`
		INSERT INTO audit_log (transfer_id, operation, from_account_id, to_account_id, amount, outcome, failure_reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		transferID, operation, fromID, toID, amount, outcome, failureReason,
	)
	return err
}

func ListTransfers(db *sql.DB, limit, offset int) ([]models.Transfer, error) {
	rows, err := db.Query(`
		SELECT id, from_account_id, to_account_id, amount, status, failure_reason, reversed_by, fee, created_at
		FROM transfers
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []models.Transfer
	for rows.Next() {
		var t models.Transfer
		var fr, rb sql.NullString
		var feeNull decimal.NullDecimal
		if err := rows.Scan(&t.ID, &t.FromAccountID, &t.ToAccountID, &t.Amount, &t.Status, &fr, &rb, &feeNull, &t.CreatedAt); err != nil {
			return nil, err
		}
		if fr.Valid {
			t.FailureReason = &fr.String
		}
		if rb.Valid {
			t.ReversedBy = &rb.String
		}
		if feeNull.Valid {
			t.Fee = &feeNull.Decimal
		}
		transfers = append(transfers, t)
	}
	return transfers, rows.Err()
}

func GetTransferByID(db *sql.DB, id string) (models.Transfer, error) {
	var t models.Transfer
	var fr, rb sql.NullString
	var feeNull decimal.NullDecimal
	err := db.QueryRow(`
		SELECT id, from_account_id, to_account_id, amount, status, failure_reason, reversed_by, fee, created_at
		FROM transfers
		WHERE id = $1`, id,
	).Scan(&t.ID, &t.FromAccountID, &t.ToAccountID, &t.Amount, &t.Status, &fr, &rb, &feeNull, &t.CreatedAt)
	if fr.Valid {
		t.FailureReason = &fr.String
	}
	if rb.Valid {
		t.ReversedBy = &rb.String
	}
	if feeNull.Valid {
		t.Fee = &feeNull.Decimal
	}
	return t, err
}

// GetReversalByOriginalID returns the reversal transfer that points back to originalID.
// Returns sql.ErrNoRows if no reversal exists yet — used for idempotency check.
func GetReversalByOriginalID(db *sql.DB, originalID string) (models.Transfer, error) {
	var t models.Transfer
	var fr, rb sql.NullString
	var feeNull decimal.NullDecimal
	err := db.QueryRow(`
		SELECT id, from_account_id, to_account_id, amount, status, failure_reason, reversed_by, fee, created_at
		FROM transfers
		WHERE reversed_by = $1`, originalID,
	).Scan(&t.ID, &t.FromAccountID, &t.ToAccountID, &t.Amount, &t.Status, &fr, &rb, &feeNull, &t.CreatedAt)
	if fr.Valid {
		t.FailureReason = &fr.String
	}
	if rb.Valid {
		t.ReversedBy = &rb.String
	}
	if feeNull.Valid {
		t.Fee = &feeNull.Decimal
	}
	return t, err
}

// InsertReversalTransfer inserts T2 — the reversal transfer — with reversed_by pointing to T1.
// Reversals do not carry a fee (fee column remains NULL).
func InsertReversalTransfer(tx *sql.Tx, fromID, toID string, amount decimal.Decimal, reversedBy string) (models.Transfer, error) {
	var t models.Transfer
	var fr, rb sql.NullString
	var feeNull decimal.NullDecimal
	err := tx.QueryRow(`
		INSERT INTO transfers (from_account_id, to_account_id, amount, status, reversed_by)
		VALUES ($1, $2, $3, 'COMPLETED', $4)
		RETURNING id, from_account_id, to_account_id, amount, status, failure_reason, reversed_by, fee, created_at`,
		fromID, toID, amount, reversedBy,
	).Scan(&t.ID, &t.FromAccountID, &t.ToAccountID, &t.Amount, &t.Status, &fr, &rb, &feeNull, &t.CreatedAt)
	if fr.Valid {
		t.FailureReason = &fr.String
	}
	if rb.Valid {
		t.ReversedBy = &rb.String
	}
	if feeNull.Valid {
		t.Fee = &feeNull.Decimal
	}
	return t, err
}

// UpdateTransferStatus updates T1's status to REVERSED after a successful reversal.
func UpdateTransferStatus(tx *sql.Tx, id, status string) error {
	_, err := tx.Exec(`UPDATE transfers SET status = $1 WHERE id = $2`, status, id)
	return err
}
