package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type CategorySeedingService interface {
	SeedDefaultCategories(ctx context.Context, userID uuid.UUID) error
}

type categorySeedingService struct {
	categoryRepo repository.CategoryRepository
}

func NewCategorySeedingService(categoryRepo repository.CategoryRepository) CategorySeedingService {
	return &categorySeedingService{categoryRepo: categoryRepo}
}

// DefaultCategories contains the default categories for new users
var DefaultCategories = []struct {
	Name  string
	Type  models.CategoryType
	Icon  string
	Color string
}{
	// Expense categories
	{"Makanan & Minuman", models.CategoryTypeExpense, "🍔", "#FF6B6B"},
	{"Transportasi", models.CategoryTypeExpense, "🚗", "#4ECDC4"},
	{"Belanja", models.CategoryTypeExpense, "🛍️", "#45B7D1"},
	{"Hiburan", models.CategoryTypeExpense, "🎬", "#96CEB4"},
	{"Kesehatan", models.CategoryTypeExpense, "🏥", "#FFEAA7"},
	{"Pendidikan", models.CategoryTypeExpense, "📚", "#DDA0DD"},
	{"Tagihan & Utilitas", models.CategoryTypeExpense, "💡", "#98D8C8"},
	{"Lainnya", models.CategoryTypeExpense, "📦", "#F7DC6F"},
	
	// Income categories
	{"Gaji", models.CategoryTypeIncome, "💰", "#2ECC71"},
	{"Bonus", models.CategoryTypeIncome, "🎁", "#3498DB"},
	{"Investasi", models.CategoryTypeIncome, "📈", "#9B59B6"},
	{"Lainnya", models.CategoryTypeIncome, "💵", "#1ABC9C"},
}

func (s *categorySeedingService) SeedDefaultCategories(ctx context.Context, userID uuid.UUID) error {
	for _, cat := range DefaultCategories {
		category := &models.Category{
			ID:       uuid.New(),
			UserID:   &userID,
			Name:     cat.Name,
			Type:     cat.Type,
			Icon:     &cat.Icon,
			Color:    &cat.Color,
			IsSystem: false,
		}

		err := s.categoryRepo.Create(ctx, category)
		if err != nil {
			return fmt.Errorf("failed to create category %s: %w", cat.Name, err)
		}
	}

	return nil
}
