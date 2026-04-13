-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS usage_logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID REFERENCES users(id) ON DELETE CASCADE,
    action     VARCHAR(50) NOT NULL,
    tokens_used INTEGER,
    cost_usd   DECIMAL(10, 8),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_usage_logs_user_id ON usage_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_action ON usage_logs(action);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS usage_logs;
-- +goose StatementEnd
