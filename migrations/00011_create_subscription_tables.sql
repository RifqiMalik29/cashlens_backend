-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS subscription_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID REFERENCES users(id) ON DELETE CASCADE,
    event_type    VARCHAR(30) NOT NULL,
    plan          VARCHAR(20),
    price_paid    DECIMAL(10, 2),
    external_invoice_id VARCHAR(255),
    cancel_reason VARCHAR(100),
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_subscription_events_user ON subscription_events(user_id, event_type);
CREATE INDEX idx_subscription_events_invoice ON subscription_events(external_invoice_id);

-- Track pending invoices to map Xendit webhook → user_id
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

CREATE INDEX idx_pending_invoices_external ON pending_invoices(external_invoice_id);
CREATE INDEX idx_pending_invoices_user ON pending_invoices(user_id, status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS pending_invoices;
DROP TABLE IF EXISTS subscription_events;
-- +goose StatementEnd
