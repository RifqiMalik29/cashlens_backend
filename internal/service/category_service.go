package service

import (
	"context"
	"fmt"
	"time"

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
	if req.Name == "" {
		return nil, fmt.Errorf("Name can't be empty")
	}

	if req.Type != models.CategoryTypeExpense && req.Type != models.CategoryTypeIncome {
		return nil, fmt.Errorf("Invalid category type")
	}

	category := &models.Category{
		ID:        uuid.New(),
		UserID:    &userID,
		Name:      req.Name,
		Color:     &req.Color,
		Icon:      &req.Icon,
		Type:      req.Type,
		IsSystem:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.categoryRepo.Create(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("Failed to save category: %w", err)
	}

	return category, nil
}

func (s *categoryService) Get(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	return nil, nil
}

func (s *categoryService) ListUserCategories(ctx context.Context, userID uuid.UUID) ([]*models.Category, error) {
	res, err := s.categoryRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get category: %w", err)
	}

	return res, nil
}

func (s *categoryService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return nil
}
