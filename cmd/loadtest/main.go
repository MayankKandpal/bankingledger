package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
)

const baseURL = "http://localhost:8080"

type account struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Balance decimal.Decimal `json:"balance"`
}

type transferResp struct {
	ID            string          `json:"id"`
	FromAccountID string          `json:"from_account_id"`
	ToAccountID   string          `json:"to_account_id"`
	Amount        decimal.Decimal `json:"amount"`
	Status        string          `json:"status"`
}

func createAccount(name string) (account, error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	resp, err := http.Post(baseURL+"/accounts", "application/json", bytes.NewReader(body))
	if err != nil {
		return account{}, err
	}
	defer resp.Body.Close()
	var a account
	json.NewDecoder(resp.Body).Decode(&a)
	return a, nil
}

func doTransfer(fromID, toID, amount string) bool {
	body, _ := json.Marshal(map[string]string{
		"from_account_id": fromID,
		"to_account_id":   toID,
		"amount":          amount,
	})
	resp, err := http.Post(baseURL+"/transfers", "application/json", bytes.NewReader(body))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusCreated
}

// doTransferWithID returns the created transfer, or zero value if the request fails.
func doTransferWithID(fromID, toID, amount string) (transferResp, bool) {
	body, _ := json.Marshal(map[string]string{
		"from_account_id": fromID,
		"to_account_id":   toID,
		"amount":          amount,
	})
	resp, err := http.Post(baseURL+"/transfers", "application/json", bytes.NewReader(body))
	if err != nil {
		return transferResp{}, false
	}
	defer resp.Body.Close()
	var t transferResp
	json.NewDecoder(resp.Body).Decode(&t)
	return t, resp.StatusCode == http.StatusCreated
}

// doReverse fires a reversal request and returns (status, reversal_id_if_present).
func doReverse(originalID string) (int, string) {
	resp, err := http.Post(baseURL+"/transfers/"+originalID+"/reverse", "application/json", nil)
	if err != nil {
		return 0, ""
	}
	defer resp.Body.Close()
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if id, ok := body["id"].(string); ok {
		return resp.StatusCode, id
	}
	return resp.StatusCode, ""
}

// verifyLedgerBalance asserts SUM(ledger_entries.amount) per account == accounts.balance
// for the given account IDs. This is the "real" invariant — money conservation on the
// cached balance can hide a torn write where ledger and balance drift apart.
func verifyLedgerBalance(db *sql.DB, accountIDs []string) []string {
	var failures []string
	for _, id := range accountIDs {
		var ledgerSum, balance decimal.Decimal
		if err := db.QueryRow(
			`SELECT COALESCE(SUM(amount), 0) FROM ledger_entries WHERE account_id = $1`, id,
		).Scan(&ledgerSum); err != nil {
			failures = append(failures, fmt.Sprintf("ledger sum query failed for %s: %v", id, err))
			continue
		}
		if err := db.QueryRow(
			`SELECT balance FROM accounts WHERE id = $1`, id,
		).Scan(&balance); err != nil {
			failures = append(failures, fmt.Sprintf("balance query failed for %s: %v", id, err))
			continue
		}
		if !ledgerSum.Equal(balance) {
			failures = append(failures, fmt.Sprintf(
				"DRIFT on %s: ledger sum=%s, cached balance=%s", id, ledgerSum, balance,
			))
		}
	}
	return failures
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env:", err)
	}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("DB open:", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("DB ping:", err)
	}

	// ---- Setup -----------------------------------------------------------
	fmt.Println("Creating accounts...")
	a, err := createAccount("LoadTest-A")
	if err != nil {
		log.Fatal("Create account A:", err)
	}
	b, err := createAccount("LoadTest-B")
	if err != nil {
		log.Fatal("Create account B:", err)
	}

	// Seed balances directly. We also seed ledger_entries so the
	// ledger ↔ balance invariant holds from the very first verification.
	const seed = 50000
	tx, err := db.Begin()
	if err != nil {
		log.Fatal("Begin seed tx:", err)
	}
	if _, err := tx.Exec(`UPDATE accounts SET balance = $1 WHERE id = $2 OR id = $3`, seed, a.ID, b.ID); err != nil {
		tx.Rollback()
		log.Fatal("Seed balances:", err)
	}
	// SYSTEM → A and SYSTEM → B as synthetic deposit transfers, so ledger entries exist.
	const sysID = "00000000-0000-0000-0000-000000000000"
	for _, target := range []string{a.ID, b.ID} {
		var transferID string
		err := tx.QueryRow(`
			INSERT INTO transfers (from_account_id, to_account_id, amount, status)
			VALUES ($1, $2, $3, 'COMPLETED') RETURNING id`,
			sysID, target, seed,
		).Scan(&transferID)
		if err != nil {
			tx.Rollback()
			log.Fatal("Seed transfer row:", err)
		}
		if _, err := tx.Exec(`
			INSERT INTO ledger_entries (transfer_id, account_id, amount) VALUES ($1, $2, $3), ($1, $4, $5)`,
			transferID, sysID, -seed, target, seed,
		); err != nil {
			tx.Rollback()
			log.Fatal("Seed ledger:", err)
		}
		// keep SYSTEM cached balance consistent with its ledger
		if _, err := tx.Exec(`UPDATE accounts SET balance = balance - $1 WHERE id = $2`, seed, sysID); err != nil {
			tx.Rollback()
			log.Fatal("Seed system balance:", err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatal("Commit seed tx:", err)
	}

	fmt.Printf("Account A: %s\n", a.ID)
	fmt.Printf("Account B: %s\n", b.ID)
	fmt.Println("Starting balances: ₹50,000 each — total ₹1,00,000")

	overallPass := true

	// ---- Phase 1: Concurrent transfer storm -------------------------------
	fmt.Println()
	fmt.Println("─── Phase 1: 50 goroutines × 20 transfers (1000 total) ───")

	var succeeded, failed int64
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(rand.Int63()))
			for j := 0; j < 20; j++ {
				amount := decimal.NewFromInt(int64(rng.Intn(1000) + 1))
				fromID, toID := a.ID, b.ID
				if rng.Intn(2) == 0 {
					fromID, toID = b.ID, a.ID
				}
				if doTransfer(fromID, toID, amount.String()) {
					atomic.AddInt64(&succeeded, 1)
				} else {
					atomic.AddInt64(&failed, 1)
				}
			}
		}()
	}
	wg.Wait()
	fmt.Printf("Transfers complete — succeeded: %d, failed (insufficient funds): %d\n", succeeded, failed)

	// Phase 1 verification
	var balA, balB decimal.Decimal
	db.QueryRow(`SELECT balance FROM accounts WHERE id = $1`, a.ID).Scan(&balA)
	db.QueryRow(`SELECT balance FROM accounts WHERE id = $1`, b.ID).Scan(&balB)
	total := balA.Add(balB)
	expected := decimal.NewFromInt(100000)

	fmt.Printf("Account A balance: ₹%s\n", balA.String())
	fmt.Printf("Account B balance: ₹%s\n", balB.String())
	fmt.Printf("Total:             ₹%s (expected ₹%s)\n", total.String(), expected.String())

	if balA.IsNegative() || balB.IsNegative() {
		fmt.Println("❌ FAIL: negative balance detected")
		overallPass = false
	} else {
		fmt.Println("✅ No negative balances")
	}
	if !total.Equal(expected) {
		fmt.Printf("❌ FAIL: cached-balance conservation broken (₹%s ≠ ₹%s)\n", total, expected)
		overallPass = false
	} else {
		fmt.Println("✅ Money conservation holds on cached balance")
	}
	if drift := verifyLedgerBalance(db, []string{a.ID, b.ID}); len(drift) > 0 {
		for _, msg := range drift {
			fmt.Println("❌ FAIL:", msg)
		}
		overallPass = false
	} else {
		fmt.Println("✅ Ledger ↔ balance match per account (no cache drift)")
	}

	// ---- Phase 2: Concurrent reversal idempotency -------------------------
	fmt.Println()
	fmt.Println("─── Phase 2: 50 goroutines reversing the same transfer ───")

	target, ok := doTransferWithID(a.ID, b.ID, "100")
	if !ok || target.ID == "" {
		fmt.Println("❌ FAIL: could not create target transfer for reversal storm")
		overallPass = false
	} else {
		fmt.Printf("Target transfer: %s (A → B ₹100)\n", target.ID)

		type result struct {
			status int
			id     string
		}
		results := make(chan result, 50)

		var rwg sync.WaitGroup
		for i := 0; i < 50; i++ {
			rwg.Add(1)
			go func() {
				defer rwg.Done()
				status, id := doReverse(target.ID)
				results <- result{status, id}
			}()
		}
		rwg.Wait()
		close(results)

		successIDs := map[string]int{} // distinct reversal IDs returned across 200 OK responses
		var ok200, alreadyReversed, otherErr int
		for r := range results {
			switch {
			case r.status == http.StatusOK:
				ok200++
				if r.id != "" {
					successIDs[r.id]++
				}
			case r.status == http.StatusUnprocessableEntity:
				alreadyReversed++
			default:
				otherErr++
			}
		}
		fmt.Printf("Responses: 200 OK=%d, 422 cannot-reverse=%d, other=%d\n", ok200, alreadyReversed, otherErr)

		// Assertion 1: every 200 response returned the same reversal ID
		if len(successIDs) > 1 {
			fmt.Printf("❌ FAIL: %d distinct reversal IDs returned across 200 responses — should be exactly 1\n", len(successIDs))
			overallPass = false
		} else {
			fmt.Println("✅ All successful reversal responses returned the same reversal ID")
		}

		// Assertion 2: exactly one reversal row exists in the DB for this original
		var reversalCount int
		db.QueryRow(`SELECT COUNT(*) FROM transfers WHERE reversed_by = $1`, target.ID).Scan(&reversalCount)
		if reversalCount != 1 {
			fmt.Printf("❌ FAIL: %d reversal rows exist for the target — should be exactly 1\n", reversalCount)
			overallPass = false
		} else {
			fmt.Println("✅ Exactly one reversal row in DB (unique constraint held)")
		}

		// Assertion 3: ledger ↔ balance still consistent after the reversal storm
		if drift := verifyLedgerBalance(db, []string{a.ID, b.ID}); len(drift) > 0 {
			for _, msg := range drift {
				fmt.Println("❌ FAIL (post-reversal):", msg)
			}
			overallPass = false
		} else {
			fmt.Println("✅ Ledger ↔ balance still match after reversal storm")
		}

		// Money conservation across A + B should still hold (the reversal is a no-op net)
		db.QueryRow(`SELECT balance FROM accounts WHERE id = $1`, a.ID).Scan(&balA)
		db.QueryRow(`SELECT balance FROM accounts WHERE id = $1`, b.ID).Scan(&balB)
		if !balA.Add(balB).Equal(expected) {
			fmt.Printf("❌ FAIL: total ₹%s ≠ ₹%s after reversal\n", balA.Add(balB), expected)
			overallPass = false
		} else {
			fmt.Println("✅ Money conservation still holds after reversal storm")
		}
	}

	fmt.Println()
	if overallPass {
		fmt.Println("✅ All assertions passed — locking, ledger consistency, and reversal idempotency are correct")
	} else {
		fmt.Println("❌ One or more assertions failed — see output above")
		os.Exit(1)
	}
}
