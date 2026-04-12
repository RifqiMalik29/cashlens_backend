-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_quotas (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_month      INTEGER NOT NULL,
    period_year       INTEGER NOT NULL,
    transactions_used INTEGER NOT NULL DEFAULT 0,
    scans_used        INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, period_month, period_year)
);

CREATE INDEX idx_user_quotas_user_period ON user_quotas(user_id, period_year, period_month);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_quotas;
-- +goose StatementEnd
