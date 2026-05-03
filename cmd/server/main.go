package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"github.com/MayankKandpal/bankingledger/internal/db"
	"github.com/MayankKandpal/bankingledger/internal/handler"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	database, err := db.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	r := mux.NewRouter()

	accounts := &handler.AccountHandler{DB: database}
	r.HandleFunc("/accounts", accounts.List).Methods(http.MethodGet)
	r.HandleFunc("/accounts", accounts.Create).Methods(http.MethodPost)

	transfers := &handler.TransferHandler{DB: database}
	r.HandleFunc("/transfers", transfers.List).Methods(http.MethodGet)
	r.HandleFunc("/transfers", transfers.Create).Methods(http.MethodPost)
	r.HandleFunc("/transfers/{id}", transfers.GetByID).Methods(http.MethodGet)
	r.HandleFunc("/transfers/{id}/reverse", transfers.Reverse).Methods(http.MethodPost)

	auditLog := &handler.AuditLogHandler{DB: database}
	r.HandleFunc("/audit-log", auditLog.List).Methods(http.MethodGet)

	ledger := &handler.LedgerHandler{DB: database}
	r.HandleFunc("/ledger-entries", ledger.List).Methods(http.MethodGet)

	log.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", corsMiddleware(r)); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
