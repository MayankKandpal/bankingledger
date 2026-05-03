package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/shopspring/decimal"

	"github.com/MayankKandpal/bankingledger/internal/repository"
	"github.com/MayankKandpal/bankingledger/internal/service"
)

type TransferHandler struct {
	DB *sql.DB
}

type createTransferRequest struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        string `json:"amount"` // string to avoid float precision issues
}

func (h *TransferHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.FromAccountID == "" || req.ToAccountID == "" {
		writeError(w, "from_account_id and to_account_id are required", http.StatusBadRequest)
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		writeError(w, "invalid amount", http.StatusBadRequest)
		return
	}

	svc := &service.TransferService{DB: h.DB}
	transfer, err := svc.Execute(service.TransferInput{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        amount,
	})

	if err != nil {
		switch {
		case errors.Is(err, service.ErrInsufficientFunds):
			// transfer was recorded as FAILED — return it with 422
			writeJSON(w, http.StatusUnprocessableEntity, transfer)
		case errors.Is(err, service.ErrSameAccount), errors.Is(err, service.ErrInvalidAmount):
			writeError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, service.ErrAccountNotFound):
			writeError(w, err.Error(), http.StatusNotFound)
		default:
			writeError(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusCreated, transfer)
}

func (h *TransferHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r, 100, 500)
	transfers, err := repository.ListTransfers(h.DB, limit, offset)
	if err != nil {
		writeError(w, "failed to list transfers", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, transfers)
}

func (h *TransferHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	transfer, err := repository.GetTransferByID(h.DB, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, "transfer not found", http.StatusNotFound)
			return
		}
		writeError(w, "failed to get transfer", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, transfer)
}

func (h *TransferHandler) Reverse(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	svc := &service.ReversalService{DB: h.DB}
	reversal, err := svc.Execute(id)

	if err != nil {
		switch {
		case errors.Is(err, service.ErrTransferNotFound):
			writeError(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, service.ErrCannotReverseFailed):
			writeError(w, err.Error(), http.StatusUnprocessableEntity)
		default:
			writeError(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusOK, reversal)
}
