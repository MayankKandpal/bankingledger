-- +goose Up
ALTER TABLE transfers ADD COLUMN fee NUMERIC(18,2);

-- +goose Down
ALTER TABLE transfers DROP COLUMN fee;
