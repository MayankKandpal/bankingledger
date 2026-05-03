-- +goose Up
-- +goose StatementBegin
CREATE INDEX idx_transfers_created ON transfers(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_transfers_created;
-- +goose StatementEnd
