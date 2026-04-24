-- +goose Up
DROP TABLE IF EXISTS pending_invoices;

-- +goose Down
CREATE TABLE IF NOT EXISTS pending_invoices (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID REFERENCES users(id) ON DELETE CASCADE,
    external_invoice_id VARCHAR(255) UNIQUE NOT NULL,
    plan                VARCHAR(20) NOT NULL,
    amount              DECIMAL(10, 2) NOT NULL,
    status              VARCHAR(20) DEFAULT 'pending', -- pending, paid, expired, failed
    expires_at          TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pending_invoices_external ON pending_invoices(external_invoice_id);
CREATE INDEX IF NOT EXISTS idx_pending_invoices_user ON pending_invoices(user_id, status);
