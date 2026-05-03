package service

import (
	"database/sql"
	"errors"

	"github.com/MayankKandpal/bankingledger/internal/models"
	"github.com/MayankKandpal/bankingledger/internal/repository"
	"github.com/shopspring/decimal"
)

const systemAccountID = "00000000-0000-0000-0000-000000000000"

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrSameAccount       = errors.New("source and destination accounts must be different")
	ErrInvalidAmount     = errors.New("amount must be greater than zero")
	ErrAccountNotFound   = errors.New("account not found")
)

type TransferService struct {
	DB *sql.DB
}

type TransferInput struct {
	FromAccountID string
	ToAccountID   string
	Amount        decimal.Decimal
}

func (s *TransferService) Execute(input TransferInput) (models.Transfer, error) {
	// input validation
	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return models.Transfer{}, ErrInvalidAmount
	}
	if input.FromAccountID == input.ToAccountID {
		return models.Transfer{}, ErrSameAccount
	}

	// sort account IDs — always lock lower UUID first to prevent deadlocks
	first, second := input.FromAccountID, input.ToAccountID
	if first > second {
		first, second = second, first
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return models.Transfer{}, err
	}
	defer tx.Rollback() // no-op if already committed

	// lock both rows atomically — no other transaction can read or write these rows now
	firstAcc, err := repository.GetAccountForUpdate(tx, first)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Transfer{}, ErrAccountNotFound
		}
		return models.Transfer{}, err
	}

	secondAcc, err := repository.GetAccountForUpdate(tx, second)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Transfer{}, ErrAccountNotFound
		}
		return models.Transfer{}, err
	}

	// determine which locked account is the source
	var fromBalance decimal.Decimal
	if firstAcc.ID == input.FromAccountID {
		fromBalance = firstAcc.Balance
	} else {
		fromBalance = secondAcc.Balance
	}

	// SYSTEM account represents external funds — its balance is allowed to go negative
	if input.FromAccountID != systemAccountID && fromBalance.LessThan(input.Amount) {
		reason := "INSUFFICIENT_FUNDS"
		transfer, err := repository.InsertTransfer(tx, input.FromAccountID, input.ToAccountID, input.Amount, "FAILED", &reason)
		if err != nil {
			return models.Transfer{}, err
		}
		if err := repository.InsertAuditLog(tx, &transfer.ID, "TRANSFER", input.FromAccountID, input.ToAccountID, input.Amount, "FAILURE", &reason); err != nil {
			return models.Transfer{}, err
		}
		// commit to persist the failed attempt record, then surface the error to the caller
		if err := tx.Commit(); err != nil {
			return models.Transfer{}, err
		}
		return transfer, ErrInsufficientFunds
	}

	// insert transfer first to get its ID, which ledger entries reference
	transfer, err := repository.InsertTransfer(tx, input.FromAccountID, input.ToAccountID, input.Amount, "COMPLETED", nil)
	if err != nil {
		return models.Transfer{}, err
	}
	if err := repository.InsertLedgerEntry(tx, transfer.ID, input.FromAccountID, input.Amount.Neg()); err != nil {
		return models.Transfer{}, err
	}
	if err := repository.InsertLedgerEntry(tx, transfer.ID, input.ToAccountID, input.Amount); err != nil {
		return models.Transfer{}, err
	}
	if err := repository.UpdateAccountBalance(tx, input.FromAccountID, input.Amount.Neg()); err != nil {
		return models.Transfer{}, err
	}
	if err := repository.UpdateAccountBalance(tx, input.ToAccountID, input.Amount); err != nil {
		return models.Transfer{}, err
	}
	if err := repository.InsertAuditLog(tx, &transfer.ID, "TRANSFER", input.FromAccountID, input.ToAccountID, input.Amount, "SUCCESS", nil); err != nil {
		return models.Transfer{}, err
	}

	return transfer, tx.Commit()
}
