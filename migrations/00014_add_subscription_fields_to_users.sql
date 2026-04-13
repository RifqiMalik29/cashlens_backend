-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_tier VARCHAR(20) DEFAULT 'free';
ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_expires_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_founder BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS win_back_sent_at TIMESTAMPTZ;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN IF EXISTS subscription_tier;
ALTER TABLE users DROP COLUMN IF EXISTS subscription_expires_at;
ALTER TABLE users DROP COLUMN IF EXISTS is_founder;
ALTER TABLE users DROP COLUMN IF EXISTS win_back_sent_at;
-- +goose StatementEnd
