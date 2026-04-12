-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_chat_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(20) NOT NULL CHECK (platform IN ('telegram', 'whatsapp')),
    chat_id VARCHAR(100) NOT NULL,
    username VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    linked_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(chat_id, platform)
);

CREATE INDEX idx_user_chat_links_user_id ON user_chat_links(user_id);
CREATE INDEX idx_user_chat_links_chat_id ON user_chat_links(chat_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_chat_links;
-- +goose StatementEnd
