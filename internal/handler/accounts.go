package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/MayankKandpal/bankingledger/internal/repository"
)

type AccountHandler struct {
	DB *sql.DB
}

type createAccountRequest struct {
	Name   string `json:"name"`
	Mobile string `json:"mobile"`
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		writeError(w, "name is required", http.StatusBadRequest)
		return
	}

	account, err := repository.CreateAccount(h.DB, req.Name, req.Mobile)
	if err != nil {
		writeError(w, "failed to create account", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, account)
}

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	accounts, err := repository.ListAccounts(h.DB)
	if err != nil {
		writeError(w, "failed to list accounts", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, accounts)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, msg string, status int) {
	writeJSON(w, status, map[string]string{"error": msg})
}
