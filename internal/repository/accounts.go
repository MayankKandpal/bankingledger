package repository

import (
	"database/sql"

	"github.com/MayankKandpal/bankingledger/internal/models"
)

const systemAccountID = "00000000-0000-0000-0000-000000000000"

func CreateAccount(db *sql.DB, name, mobile string) (models.Account, error) {
	query := `
		INSERT INTO accounts (name, mobile)
		VALUES ($1, NULLIF($2, ''))
		RETURNING id, name, mobile, balance, created_at`

	var a models.Account
	var mob sql.NullString

	err := db.QueryRow(query, name, mobile).Scan(
		&a.ID, &a.Name, &mob, &a.Balance, &a.CreatedAt,
	)
	if err != nil {
		return models.Account{}, err
	}
	if mob.Valid {
		a.Mobile = &mob.String
	}
	return a, nil
}

func ListAccounts(db *sql.DB) ([]models.Account, error) {
	query := `
		SELECT id, name, mobile, balance, created_at
		FROM accounts
		WHERE id != $1
		ORDER BY created_at ASC`

	rows, err := db.Query(query, systemAccountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var a models.Account
		var mob sql.NullString
		if err := rows.Scan(&a.ID, &a.Name, &mob, &a.Balance, &a.CreatedAt); err != nil {
			return nil, err
		}
		if mob.Valid {
			a.Mobile = &mob.String
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}
