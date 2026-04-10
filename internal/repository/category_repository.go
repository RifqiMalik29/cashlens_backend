package repository

import (
	"context"

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
func (r *categoryRepository) Create(ctx context.Context, category *models.Category) error {
	return nil
}

func (r *categoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	return nil, nil
}

func (r *categoryRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Category, error) {
	return nil, nil
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
