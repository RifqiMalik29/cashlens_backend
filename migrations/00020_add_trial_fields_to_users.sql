-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN IF NOT EXISTS device_id TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS trial_status VARCHAR(20) NOT NULL DEFAULT 'inactive';
ALTER TABLE users ADD COLUMN IF NOT EXISTS trial_start_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS trial_end_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_users_device_id ON users(device_id);
CREATE INDEX IF NOT EXISTS idx_users_trial_status_end ON users(trial_status, trial_end_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_trial_status_end;
DROP INDEX IF EXISTS idx_users_device_id;
ALTER TABLE users DROP COLUMN IF EXISTS trial_end_at;
ALTER TABLE users DROP COLUMN IF EXISTS trial_start_at;
ALTER TABLE users DROP COLUMN IF EXISTS trial_status;
ALTER TABLE users DROP COLUMN IF EXISTS device_id;
-- +goose StatementEnd
