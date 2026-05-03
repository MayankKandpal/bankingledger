package service

import (
	"database/sql"
	"errors"

	"github.com/MayankKandpal/bankingledger/internal/models"
	"github.com/MayankKandpal/bankingledger/internal/repository"
)

var (
	ErrTransferNotFound      = errors.New("transfer not found")
	ErrCannotReverseFailed   = errors.New("cannot reverse a failed transfer")
	ErrCannotReverseReversal = errors.New("cannot reverse a reversal transfer")
)

type ReversalService struct {
	DB *sql.DB
}

func (s *ReversalService) Execute(originalID string) (models.Transfer, error) {
	// 1. Fetch T1
	t1, err := repository.GetTransferByID(s.DB, originalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Transfer{}, ErrTransferNotFound
		}
		return models.Transfer{}, err
	}

	// A reversal is terminal — refuse to reverse a transfer that is itself a reversal.
	// This check fires before the REVERSED-status check on purpose: a reversal that has
	// itself been reversed has both reversed_by != nil AND status == REVERSED, and we
	// want to reject it explicitly rather than return a chain link as an idempotent hit.
	if t1.ReversedBy != nil {
		return models.Transfer{}, ErrCannotReverseReversal
	}

	// 2 & 3. Status guards
	if t1.Status == "FAILED" {
		return models.Transfer{}, ErrCannotReverseFailed
	}
	if t1.Status == "REVERSED" {
		// idempotent — return the existing reversal rather than an error
		return repository.GetReversalByOriginalID(s.DB, originalID)
	}

	// 4. Idempotency check — return existing reversal without re-applying
	existing, err := repository.GetReversalByOriginalID(s.DB, originalID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return models.Transfer{}, err
	}
	if err == nil {
		return existing, nil
	}

	// 5. Begin transaction
	tx, err := s.DB.Begin()
	if err != nil {
		return models.Transfer{}, err
	}
	defer tx.Rollback()

	// 6. Lock both accounts in consistent order — same rule as transfer
	first, second := t1.FromAccountID, t1.ToAccountID
	if first > second {
		first, second = second, first
	}
	if _, err := repository.GetAccountForUpdate(tx, first); err != nil {
		return models.Transfer{}, err
	}
	if _, err := repository.GetAccountForUpdate(tx, second); err != nil {
		return models.Transfer{}, err
	}

	// 7. Insert T2 — from/to are flipped, reversed_by points to T1
	// Unique constraint on reversed_by is the final safety net against concurrent reversals
	t2, err := repository.InsertReversalTransfer(tx, t1.ToAccountID, t1.FromAccountID, t1.Amount, originalID)
	if err != nil {
		if repository.IsUniqueViolation(err) {
			// A concurrent request just committed a reversal — fetch and return it
			tx.Rollback()
			return repository.GetReversalByOriginalID(s.DB, originalID)
		}
		return models.Transfer{}, err
	}

	// 8. Ledger entries with flipped signs
	// Original T1: Alice(-500), Bob(+500)
	// Reversal T2: Bob(-500), Alice(+500)
	if err := repository.InsertLedgerEntry(tx, t2.ID, t1.ToAccountID, t1.Amount.Neg()); err != nil {
		return models.Transfer{}, err
	}
	if err := repository.InsertLedgerEntry(tx, t2.ID, t1.FromAccountID, t1.Amount); err != nil {
		return models.Transfer{}, err
	}

	// 9. Restore both balances
	if err := repository.UpdateAccountBalance(tx, t1.ToAccountID, t1.Amount.Neg()); err != nil {
		return models.Transfer{}, err
	}
	if err := repository.UpdateAccountBalance(tx, t1.FromAccountID, t1.Amount); err != nil {
		return models.Transfer{}, err
	}

	// 10. Mark T1 as REVERSED
	if err := repository.UpdateTransferStatus(tx, originalID, "REVERSED"); err != nil {
		return models.Transfer{}, err
	}

	// 11. Audit log
	if err := repository.InsertAuditLog(tx, &t2.ID, "REVERSAL", t1.ToAccountID, t1.FromAccountID, t1.Amount, "SUCCESS", nil); err != nil {
		return models.Transfer{}, err
	}

	return t2, tx.Commit()
}
