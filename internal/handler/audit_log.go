package handler

import (
	"database/sql"
	"net/http"

	"github.com/MayankKandpal/bankingledger/internal/repository"
)

type AuditLogHandler struct {
	DB *sql.DB
}

func (h *AuditLogHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r, 100, 500)
	entries, err := repository.ListAuditLog(h.DB, limit, offset)
	if err != nil {
		writeError(w, "failed to fetch audit log", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}
