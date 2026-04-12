package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/errors"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type CategoryService interface {
	Create(ctx context.Context, userID uuid.UUID, req models.CreateCategoryRequest) (*models.Category, error)
	Get(ctx context.Context, id, userID uuid.UUID) (*models.Category, error)
	ListUserCategories(ctx context.Context, userID uuid.UUID) ([]*models.Category, error)
	Update(ctx context.Context, id, userID uuid.UUID, req models.UpdateCategoryRequest) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type categoryService struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryService(categoryRepo repository.CategoryRepository) CategoryService {
	return &categoryService{categoryRepo: categoryRepo}
}

func (s *categoryService) Create(ctx context.Context, userID uuid.UUID, req models.CreateCategoryRequest) (*models.Category, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	if req.Type != models.CategoryTypeExpense && req.Type != models.CategoryTypeIncome {
		return nil, fmt.Errorf("invalid category type")
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
		return nil, fmt.Errorf("failed to save category: %w", err)
	}

	return category, nil
}

func (s *categoryService) Get(ctx context.Context, id, userID uuid.UUID) (*models.Category, error) {
	res, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	// Allow access to system categories or user's own categories
	if !res.IsSystem && (res.UserID == nil || *res.UserID != userID) {
		return nil, fmt.Errorf("forbidden access")
	}

	return res, nil
}

func (s *categoryService) ListUserCategories(ctx context.Context, userID uuid.UUID) ([]*models.Category, error) {
	res, err := s.categoryRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	return res, nil
}

func (s *categoryService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get category: %w", err)
	}

	// Check if category is a system category
	if category.IsSystem {
		return errors.NewBadRequest("Cannot delete system category")
	}

	// Check if user owns the category
	if category.UserID == nil || *category.UserID != userID {
		return errors.NewForbidden("Forbidden Access")
	}

	err = s.categoryRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}

func (s *categoryService) Update(ctx context.Context, id, userID uuid.UUID, req models.UpdateCategoryRequest) error {
	res, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get category: %w", err)
	}

	// Check if category is a system category
	if res.IsSystem {
		return errors.NewBadRequest("Cannot update system category")
	}

	// Check if user owns the category
	if res.UserID == nil || *res.UserID != userID {
		return errors.NewForbidden("Forbidden Access")
	}

	if req.Name != nil {
		res.Name = *req.Name
	}

	if req.Type != nil && (*req.Type == models.CategoryTypeExpense || *req.Type == models.CategoryTypeIncome) {
		res.Type = *req.Type
	}

	if req.Icon != nil {
		res.Icon = req.Icon
	}

	if req.Color != nil {
		res.Color = req.Color
	}

	res.UpdatedAt = time.Now()

	err = s.categoryRepo.Update(ctx, res)
	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	return nil
}
