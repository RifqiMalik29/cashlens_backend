-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN IF NOT EXISTS reminder_3d_sent_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS reminder_1d_sent_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS expo_push_token VARCHAR(255);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN IF EXISTS reminder_3d_sent_at;
ALTER TABLE users DROP COLUMN IF EXISTS reminder_1d_sent_at;
ALTER TABLE users DROP COLUMN IF EXISTS expo_push_token;
-- +goose StatementEnd
