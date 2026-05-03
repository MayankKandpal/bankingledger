# Banking Ledger System

## How to Run

Prerequisites:
- Go 1.26 or compatible with the module toolchain
- Node.js and npm
- PostgreSQL database
- Goose migration CLI (`go install github.com/pressly/goose/v3/cmd/goose@v3.27.1`)

```bash
# 1. Clone the repo and set up environment
cp .env.example .env
# Fill in your DATABASE_URL in .env

# 2. Run migrations (export DATABASE_URL from .env first)
export $(grep -v '^#' .env | xargs)
goose -dir ./migrations postgres "$DATABASE_URL" up

# 3. Start the backend
go run ./cmd/server
# Server starts on http://localhost:8080

# 4. Start the frontend
cd frontend
npm install
npm run dev
# Opens on http://localhost:5173
```

---

## How to Run Migrations

```bash
# Export DATABASE_URL from .env first
export $(grep -v '^#' .env | xargs)

# Apply all migrations
goose -dir ./migrations postgres "$DATABASE_URL" up

# Roll back all migrations (full reset)
goose -dir ./migrations postgres "$DATABASE_URL" down-to 0

# Re-apply after reset
goose -dir ./migrations postgres "$DATABASE_URL" up
```

---

## How to Run the Concurrency Test

The server must be running before executing the loadtest.

```bash
go run ./cmd/loadtest
```

The test runs two phases:

**Phase 1:** 50 goroutines × 20 transfers = 1000 concurrent transfers between two accounts

**Phase 2:** 50 goroutines all attempt to reverse the same transfer simultaneously

Assertions checked:
- No account has a negative balance
- Total sum of both accounts equals ₹1,00,000 (money conservation on cached balance)
- `SUM(ledger_entries.amount)` per account matches `accounts.balance` (no cache drift)
- Only one reversal row exists despite 50 concurrent reversal attempts (idempotency)

**Sample output from a clean run:**

```
Account A: 42e37cf3-4733-4c7d-bcb7-f12c28546240
Account B: 70e390fa-4b9f-4ed0-bdb4-98fbfb5852c7
Starting balances: ₹50,000 each — total ₹1,00,000

─── Phase 1: 50 goroutines × 20 transfers (1000 total) ───
Transfers complete — succeeded: 1000, failed (insufficient funds): 0
Account A balance: ₹28080
Account B balance: ₹71920
Total:             ₹100000 (expected ₹100000)
✅ No negative balances
✅ Money conservation holds on cached balance
✅ Ledger ↔ balance match per account (no cache drift)

─── Phase 2: 50 goroutines reversing the same transfer ───
Target transfer: bb649603-018b-4f00-bb51-0d5bc7f80d13 (A → B ₹100)
Responses: 200 OK=50, 422 cannot-reverse=0, other=0
✅ All successful reversal responses returned the same reversal ID
✅ Exactly one reversal row in DB (unique constraint held)
✅ Ledger ↔ balance still match after reversal storm
✅ Money conservation still holds after reversal storm

✅ All assertions passed — locking, ledger consistency, and reversal idempotency are correct
```
---

## Schema Design

### Why Four Tables

The system uses four tables, each with a single responsibility:

| Table | Responsibility |
|---|---|
| `accounts` | Who exists and what is their current balance |
| `transfers` | What was attempted and what is its current state |
| `ledger_entries` | What actually moved on which account |
| `audit_log` | What happened in order, including failures, forever |

**Transfers and ledger entries are different concepts.** A transfer is a unit of
intent — "move ₹500 from A to B". A ledger entry is a unit of fact — "account A
has ₹500 less". One transfer always produces exactly two ledger entries. A failed
transfer produces zero ledger entries because no money moved — but it still
produces a transfer row and an audit row, because the attempt happened.

This distinction is the clearest proof they cannot be the same table: a failed
transfer has nowhere to go in a ledger because nothing moved.

Insufficient-funds attempts are persisted as `FAILED` transfers and audit rows.
Malformed requests and account-not-found validations are rejected before a
transfer row is created.

### Why NUMERIC(18,2) and Not FLOAT

Float is a binary approximation. `0.1 + 0.2` in floating-point is
`0.30000000000000004`. In a financial system, ₹0.30 must be exactly ₹0.30.
`NUMERIC(18,2)` gives exact decimal arithmetic at the cost of slightly more
storage — always the right trade-off for money.

### The SYSTEM Account

A reserved account with a fixed ID (`00000000-0000-0000-0000-000000000000`)
represents money entering or leaving the system.

- Cash deposit: `from_account_id = SYSTEM`, `to_account_id = user`
- Cash withdrawal: `from_account_id = user`, `to_account_id = SYSTEM`

This means deposits, withdrawals, and transfers all use the same transfer and
ledger model instead of separate deposit/withdrawal tables. Money conservation
still holds — `SUM(all ledger entries)` across all accounts including SYSTEM
always equals zero.

The SYSTEM account balance is intentionally allowed to go negative. It represents
money that has entered the system from outside.

---

## Concurrency Approach

### The Race Condition

Without locking, two concurrent transfers on the same account produce a stale read:

```
T1 reads balance of A: ₹1000  (no lock)
T2 reads balance of A: ₹1000  (no lock)
T1 deducts ₹800 → writes ₹200
T2 deducts ₹700 → writes ₹300  ← wrong, should have failed
```

### Solution: SELECT FOR UPDATE

Lock both account rows before reading the balance. The lock and the read are
atomic — no other transaction can modify those rows between the read and the write.

```sql
SELECT balance FROM accounts WHERE id = $1 FOR UPDATE;
```

### Lock Ordering Prevents Deadlocks

Without consistent ordering, two transfers in opposite directions deadlock:

```
T1: A → B  — locks A, waits for B
T2: B → A  — locks B, waits for A
Both wait forever → DEADLOCK
```

Fix: always lock the lower UUID first, regardless of transfer direction.

```
T1: A → B  — lower UUID is A, locks A first, then B
T2: B → A  — lower UUID is A, locks A first — must wait for T1
T2 waits. T1 commits. T2 proceeds. No deadlock.
```

### Crash Safety

All writes — ledger entries, balance updates, transfer status, audit log — happen
inside a single transaction. If the process crashes at any point, the entire
transaction rolls back. You never end up with ledger entries and no transfer row,
or a balance update with no audit entry.

One exception by design: a transfer that fails due to insufficient funds is still
committed. The transfer row (status=FAILED) and audit row are preserved so that
the failed attempt is visible in history.

---

## Reversal and Idempotency

### The Requirement

Repeating the same reversal request must not apply the correction more than once.

### Three-Layer Defence

**Layer 1 — Status check:**
If `T1.status == REVERSED`, fetch and return the existing reversal. This keeps
repeat reversal requests idempotent for clients.

**Layer 2 — Application idempotency check:**
Before opening a transaction, query for an existing reversal row:
```sql
SELECT id FROM transfers WHERE reversed_by = $1
```
If found, return the existing reversal — do not re-apply.

**Layer 3 — Database unique constraint:**
```sql
CREATE UNIQUE INDEX unique_reversal ON transfers(reversed_by)
  WHERE reversed_by IS NOT NULL;
```
Two concurrent requests can both pass layers 1 and 2 before either has written
anything. The unique constraint fires at the INSERT itself. Only one can succeed.
The second hits a constraint violation and the correct response (the existing
reversal) is fetched and returned.

Layer 2 reduces unnecessary load. Layer 3 is the guarantee.

### What a Reversal Looks Like

Original transfer T1: Alice → Bob ₹500

```
transfers:
  T1: from=Alice, to=Bob,   amount=500, status=REVERSED, reversed_by=null
  T2: from=Bob,   to=Alice, amount=500, status=COMPLETED, reversed_by=T1.id

ledger_entries:
  row 1: account=Alice, amount=-500, transfer_id=T1  (original debit)
  row 2: account=Bob,   amount=+500, transfer_id=T1  (original credit)
  row 3: account=Bob,   amount=-500, transfer_id=T2  (reversal debit)
  row 4: account=Alice, amount=+500, transfer_id=T2  (reversal credit)

audit_log:
  row 1: operation=TRANSFER, outcome=SUCCESS
  row 2: operation=REVERSAL, outcome=SUCCESS
```

---

## Balance Calculation

### Cached Balance

`accounts.balance` is a running total updated inside the same transaction as
every ledger write. Because both happen atomically, they are always in sync —
there is no window where one exists without the other.

### Why Keep the Ledger

Even with a cached balance, the ledger is essential:
- Full transaction history per account (balance column cannot provide this)
- Cache verification: `SUM(ledger_entries.amount) WHERE account_id = X` should
  always equal `accounts.balance`
- Money conservation check: `SUM(all ledger_entries.amount)` across all accounts
  including SYSTEM should always equal zero

The cached balance gives O(1) reads. The ledger provides the audit trail and
correctness verification.

---

## API Endpoints

```
GET  /accounts                       list all accounts (excludes SYSTEM)
POST /accounts                       create account {name, mobile?}

GET  /transfers                      list transfers (paginated)
POST /transfers                      create transfer {from_account_id, to_account_id, amount}
GET  /transfers/:id                  get single transfer
POST /transfers/:id/reverse          reverse a completed transfer

GET  /audit-log                      list audit entries (paginated)
GET  /ledger-entries?account_id=X    list ledger entries for an account
```

Amount is always sent as a JSON string (`"amount": "500.00"`) to avoid float64
precision loss at the HTTP boundary.

---

## Trade-offs

### SELECT FOR UPDATE vs SERIALIZABLE Isolation

`SERIALIZABLE` isolation would handle deadlock prevention automatically — no
manual lock ordering required. But serialization failures require retry logic
in the application layer. Under high contention, retries can cascade.

`SELECT FOR UPDATE` with lock ordering is more explicit but gives predictable,
deadlock-free behavior with no retry complexity. For a banking system where
correctness is more important than throughput, pessimistic locking is the right
default.

### Cached Balance vs Pure Derived Balance

A pure derived balance (`SELECT SUM(amount) FROM ledger_entries WHERE account_id = X`)
is always correct but becomes O(n) after millions of transactions — unacceptable
at scale.

A cached balance gives O(1) reads. Keeping both means the cache is verifiable
at any time by comparing against the ledger sum. A background reconciliation job
can flag any discrepancy.

### SYSTEM Account vs Nullable to_account_id

Making `to_account_id` nullable would require null checks throughout the codebase.
The balance derivation breaks for the "outside money" side. The reversal logic
needs special casing.

The SYSTEM account keeps the schema uniform. All operations — deposits,
withdrawals, transfers — use the same transfer/ledger shape. The only special
rule is that SYSTEM is allowed to go negative because it represents external
money entering or leaving the application.

### TEXT + CHECK vs Postgres ENUM for Status

`ENUM` types in Postgres are hard to modify — adding a new value requires a
migration and can cause deployment coordination issues. `TEXT` with a `CHECK`
constraint (`CHECK (status IN ('COMPLETED', 'FAILED', 'REVERSED'))`) enforces
valid values at the database level while remaining easy to alter.

### COMMIT on Failure vs ROLLBACK on Failure

When a transfer fails due to insufficient funds, the transaction is committed
rather than rolled back. This preserves the failed transfer row and audit entry.
A rollback would leave no trace of the attempt. The trade-off is slightly more
storage and a more complex transaction flow — worth it for a system where
auditability is a first-class requirement.

---

## Assumptions

- SYSTEM account balance is allowed to go negative (it represents external funds)
- No authentication or authorization
- Mobile number is optional free text in this assignment; no uniqueness or format
  validation is enforced
- Insufficient-funds failures are audited; malformed requests and missing-account
  validation errors are rejected before creating transfer/audit rows
- Amount is accepted as a string in the API to avoid float64 precision loss
- Account names are not required to be unique (two customers can have the same name)
