package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type CategoryService interface {
	Create(ctx context.Context, userID uuid.UUID, req models.CreateCategoryRequest) (*models.Category, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Category, error)
	ListUserCategories(ctx context.Context, userID uuid.UUID) ([]*models.Category, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type categoryService struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryService(categoryRepo repository.CategoryRepository) CategoryService {
	return &categoryService{categoryRepo: categoryRepo}
}

// TODO: Implement methods (placeholder)
func (s *categoryService) Create(ctx context.Context, userID uuid.UUID, req models.CreateCategoryRequest) (*models.Category, error) {
	return nil, nil
}

func (s *categoryService) Get(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	return nil, nil
}

func (s *categoryService) ListUserCategories(ctx context.Context, userID uuid.UUID) ([]*models.Category, error) {
	return nil, nil
}

func (s *categoryService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return nil
}
