-- +goose Up
-- +goose StatementBegin
CREATE TABLE transfers (
  id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  from_account_id  UUID          NOT NULL REFERENCES accounts(id),
  to_account_id    UUID          NOT NULL REFERENCES accounts(id),
  amount           NUMERIC(18,2) NOT NULL CHECK (amount > 0),
  status           TEXT          NOT NULL CHECK (status IN ('COMPLETED', 'FAILED', 'REVERSED')),
  failure_reason   TEXT,
  reversed_by      UUID          REFERENCES transfers(id),
  created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- Only one reversal can ever exist per original transfer.
-- Partial index ignores NULLs so unreversed transfers don't conflict.
CREATE UNIQUE INDEX unique_reversal ON transfers(reversed_by)
  WHERE reversed_by IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE transfers;
-- +goose StatementEnd
