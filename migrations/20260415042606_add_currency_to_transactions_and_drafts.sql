-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN currency VARCHAR(10) DEFAULT 'IDR' NOT NULL;
ALTER TABLE draft_transactions ADD COLUMN currency VARCHAR(10) DEFAULT 'IDR' NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions DROP COLUMN IF EXISTS currency;
ALTER TABLE draft_transactions DROP COLUMN IF EXISTS currency;
-- +goose StatementEnd
