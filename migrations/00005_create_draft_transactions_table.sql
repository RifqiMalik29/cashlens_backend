-- +goose Up
-- +goose StatementBegin
CREATE TABLE draft_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    amount DECIMAL(15, 2),
    description TEXT,
    transaction_date DATE,
    source VARCHAR(50) NOT NULL CHECK (source IN ('telegram', 'whatsapp', 'receipt_scan', 'manual')),
    raw_data JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'rejected')),
    confirmed_transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_draft_transactions_user_id ON draft_transactions(user_id);
CREATE INDEX idx_draft_transactions_status ON draft_transactions(status);
CREATE INDEX idx_draft_transactions_source ON draft_transactions(source);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS draft_transactions;
-- +goose StatementEnd
