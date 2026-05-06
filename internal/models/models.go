package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Account struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Mobile    *string         `json:"mobile,omitempty"`
	Balance   decimal.Decimal `json:"balance"`
	CreatedAt time.Time       `json:"created_at"`
}

type Transfer struct {
	ID            string           `json:"id"`
	FromAccountID string           `json:"from_account_id"`
	ToAccountID   string           `json:"to_account_id"`
	Amount        decimal.Decimal  `json:"amount"`
	Fee           *decimal.Decimal `json:"fee,omitempty"`
	Status        string           `json:"status"`
	FailureReason *string          `json:"failure_reason,omitempty"`
	ReversedBy    *string          `json:"reversed_by,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
}

type LedgerEntry struct {
	ID         string          `json:"id"`
	TransferID string          `json:"transfer_id"`
	AccountID  string          `json:"account_id"`
	Amount     decimal.Decimal `json:"amount"`
	CreatedAt  time.Time       `json:"created_at"`
}

type AuditLog struct {
	ID            string           `json:"id"`
	TransferID    *string          `json:"transfer_id,omitempty"`
	Operation     string           `json:"operation"`
	FromAccountID *string          `json:"from_account_id,omitempty"`
	ToAccountID   *string          `json:"to_account_id,omitempty"`
	Amount        *decimal.Decimal `json:"amount,omitempty"`
	Outcome       string           `json:"outcome"`
	FailureReason *string          `json:"failure_reason,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
}
