-- +goose Up
-- +goose StatementBegin
-- Add email confirmation fields to users table
ALTER TABLE users ADD COLUMN is_confirmed BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN confirmation_token VARCHAR(255);
ALTER TABLE users ADD COLUMN confirmation_expires_at TIMESTAMP;

-- Create index for faster token lookup
CREATE INDEX idx_users_confirmation_token ON users(confirmation_token);

-- Mark existing users as confirmed to avoid breaking their accounts
UPDATE users SET is_confirmed = TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN IF EXISTS is_confirmed;
ALTER TABLE users DROP COLUMN IF EXISTS confirmation_token;
ALTER TABLE users DROP COLUMN IF EXISTS confirmation_expires_at;
DROP INDEX IF EXISTS idx_users_confirmation_token;
-- +goose StatementEnd
