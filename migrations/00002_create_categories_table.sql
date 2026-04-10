-- +goose Up
-- +goose StatementBegin
CREATE TYPE category_type AS ENUM ('income', 'expense');

CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type category_type NOT NULL,
    icon VARCHAR(50),
    color VARCHAR(50),
    is_system BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_categories_user_id ON categories(user_id);
CREATE INDEX idx_categories_type ON categories(type);
CREATE INDEX idx_categories_is_system ON categories(is_system);

-- Insert default system categories
INSERT INTO categories (name, type, is_system) VALUES
    ('Salary', 'income', TRUE),
    ('Freelance', 'income', TRUE),
    ('Investment', 'income', TRUE),
    ('Food & Dining', 'expense', TRUE),
    ('Transportation', 'expense', TRUE),
    ('Shopping', 'expense', TRUE),
    ('Entertainment', 'expense', TRUE),
    ('Bills & Utilities', 'expense', TRUE),
    ('Healthcare', 'expense', TRUE),
    ('Education', 'expense', TRUE);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS categories;
DROP TYPE IF EXISTS category_type;
-- +goose StatementEnd
