-- +goose Up
-- +goose StatementBegin
CREATE TABLE ledger_entries (
  id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  transfer_id  UUID          NOT NULL REFERENCES transfers(id),
  account_id   UUID          NOT NULL REFERENCES accounts(id),
  amount       NUMERIC(18,2) NOT NULL,
  created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- amount: negative = debit, positive = credit
-- Every successful transfer creates exactly 2 rows; failed transfers create 0

CREATE INDEX idx_ledger_account  ON ledger_entries(account_id);
CREATE INDEX idx_ledger_transfer ON ledger_entries(transfer_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE ledger_entries;
-- +goose StatementEnd
