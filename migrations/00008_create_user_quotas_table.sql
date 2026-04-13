-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_quotas (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID REFERENCES users(id) ON DELETE CASCADE,
    period_month      INTEGER NOT NULL,
    period_year       INTEGER NOT NULL,
    scans_used        INTEGER DEFAULT 0,
    transactions_used INTEGER DEFAULT 0,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, period_month, period_year)
);

CREATE INDEX idx_user_quotas_user_period ON user_quotas(user_id, period_year, period_month);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_quotas;
-- +goose StatementEnd
