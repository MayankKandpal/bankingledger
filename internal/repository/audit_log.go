package repository

import (
	"database/sql"

	"github.com/MayankKandpal/bankingledger/internal/models"
	"github.com/shopspring/decimal"
)

func ListAuditLog(db *sql.DB, limit, offset int) ([]models.AuditLog, error) {
	rows, err := db.Query(`
		SELECT id, transfer_id, operation, from_account_id, to_account_id, amount, outcome, failure_reason, created_at
		FROM audit_log
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.AuditLog
	for rows.Next() {
		var e models.AuditLog
		var transferID, fromID, toID, failureReason sql.NullString
		var amount sql.NullString

		if err := rows.Scan(
			&e.ID, &transferID, &e.Operation, &fromID, &toID, &amount, &e.Outcome, &failureReason, &e.CreatedAt,
		); err != nil {
			return nil, err
		}

		if transferID.Valid {
			e.TransferID = &transferID.String
		}
		if fromID.Valid {
			e.FromAccountID = &fromID.String
		}
		if toID.Valid {
			e.ToAccountID = &toID.String
		}
		if failureReason.Valid {
			e.FailureReason = &failureReason.String
		}
		if amount.Valid {
			d, _ := decimal.NewFromString(amount.String)
			e.Amount = &d
		}

		entries = append(entries, e)
	}
	return entries, rows.Err()
}
