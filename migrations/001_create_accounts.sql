-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE accounts (
  id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  name        TEXT          NOT NULL,
  mobile      TEXT,
  balance     NUMERIC(18,2) NOT NULL DEFAULT 0.00,
  created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

INSERT INTO accounts (id, name)
VALUES ('00000000-0000-0000-0000-000000000000', 'SYSTEM');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE accounts;
-- +goose StatementEnd
