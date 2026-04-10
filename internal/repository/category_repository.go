package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

type CategoryRepository interface {
	Create(ctx context.Context, category *models.Category) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Category, error)
	ListSystem(ctx context.Context) ([]*models.Category, error)
	Update(ctx context.Context, category *models.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type categoryRepository struct {
	db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) CategoryRepository {
	return &categoryRepository{db: db}
}

// TODO: Implement all methods (placeholder)
func (r *categoryRepository) Create(ctx context.Context, c *models.Category) error {
	query := `INSERT INTO categories (id, user_id, name, type, icon, color, is_system, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.Exec(ctx, query, c.ID, c.UserID, c.Name, c.Type, c.Icon, c.Color, c.IsSystem, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("Failed to create category: %w", err)
	}

	return nil
}

func (r *categoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	return nil, nil
}

func (r *categoryRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Category, error) {
	query := `SELECT id, user_id, name, type, icon, color, is_system, created_at, updated_at FROM categories WHERE user_id = $1 OR user_id IS NULL`

	res, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("Category not found: %w", err)
	}
	// close connection to DB
	defer res.Close()
	categories := []*models.Category{}

	for res.Next() {
		c := &models.Category{}

		err := res.Scan(&c.ID, &c.UserID, &c.Name, &c.Type, &c.Icon, &c.Color, &c.IsSystem, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("Failed to scan category: %w", err)
		}

		categories = append(categories, c)
	}

	if err := res.Err(); err != nil {
		return nil, fmt.Errorf("Error during loop : %w", err)
	}

	return categories, nil
}

func (r *categoryRepository) ListSystem(ctx context.Context) ([]*models.Category, error) {
	return nil, nil
}

func (r *categoryRepository) Update(ctx context.Context, category *models.Category) error {
	return nil
}

func (r *categoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
