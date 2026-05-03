-- +goose Up
-- +goose StatementBegin
CREATE TABLE audit_log (
  id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  transfer_id      UUID          REFERENCES transfers(id),
  operation        TEXT          NOT NULL,
  from_account_id  UUID          REFERENCES accounts(id),
  to_account_id    UUID          REFERENCES accounts(id),
  amount           NUMERIC(18,2),
  outcome          TEXT          NOT NULL CHECK (outcome IN ('SUCCESS', 'FAILURE')),
  failure_reason   TEXT,
  created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- operation values: TRANSFER, REVERSAL, DEPOSIT, WITHDRAWAL
-- Append-only — rows are never updated

CREATE INDEX idx_audit_transfer ON audit_log(transfer_id);
CREATE INDEX idx_audit_from     ON audit_log(from_account_id);
CREATE INDEX idx_audit_created  ON audit_log(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE audit_log;
-- +goose StatementEnd
