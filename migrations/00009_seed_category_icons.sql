-- +goose Up
-- +goose StatementBegin
UPDATE categories SET icon = 'briefcase-outline', color = '#4CAF50' WHERE name = 'Salary' AND is_system = TRUE;
UPDATE categories SET icon = 'laptop-outline', color = '#8BC34A' WHERE name = 'Freelance' AND is_system = TRUE;
UPDATE categories SET icon = 'trending-up-outline', color = '#2196F3' WHERE name = 'Investment' AND is_system = TRUE;
UPDATE categories SET icon = 'restaurant-outline', color = '#FF5722' WHERE name = 'Food & Dining' AND is_system = TRUE;
UPDATE categories SET icon = 'car-outline', color = '#9C27B0' WHERE name = 'Transportation' AND is_system = TRUE;
UPDATE categories SET icon = 'bag-handle-outline', color = '#FF9800' WHERE name = 'Shopping' AND is_system = TRUE;
UPDATE categories SET icon = 'game-controller-outline', color = '#E91E63' WHERE name = 'Entertainment' AND is_system = TRUE;
UPDATE categories SET icon = 'receipt-outline', color = '#607D8B' WHERE name = 'Bills & Utilities' AND is_system = TRUE;
UPDATE categories SET icon = 'medkit-outline', color = '#F44336' WHERE name = 'Healthcare' AND is_system = TRUE;
UPDATE categories SET icon = 'school-outline', color = '#3F51B5' WHERE name = 'Education' AND is_system = TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE categories SET icon = NULL, color = NULL WHERE is_system = TRUE;
-- +goose StatementEnd
