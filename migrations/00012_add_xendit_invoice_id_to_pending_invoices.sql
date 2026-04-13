-- +goose Up
-- +goose StatementBegin
ALTER TABLE pending_invoices ADD COLUMN IF NOT EXISTS xendit_invoice_id VARCHAR(255);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE pending_invoices DROP COLUMN IF EXISTS xendit_invoice_id;
-- +goose StatementEnd
