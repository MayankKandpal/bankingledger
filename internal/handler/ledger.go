package handler

import (
	"database/sql"
	"net/http"

	"github.com/MayankKandpal/bankingledger/internal/repository"
)

type LedgerHandler struct {
	DB *sql.DB
}

func (h *LedgerHandler) List(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("account_id")
	limit, offset := parseLimitOffset(r, 100, 500)
	entries, err := repository.ListLedgerEntries(h.DB, accountID, limit, offset)
	if err != nil {
		writeError(w, "failed to fetch ledger entries", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}
