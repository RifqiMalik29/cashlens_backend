package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

// Mock repositories for testing
type MockCategoryRepository struct {
	CreateFunc       func(ctx context.Context, category *models.Category) error
	GetByIDFunc      func(ctx context.Context, id uuid.UUID) (*models.Category, error)
	ListByUserIDFunc func(ctx context.Context, userID uuid.UUID) ([]*models.Category, error)
	ListSystemFunc   func(ctx context.Context) ([]*models.Category, error)
	UpdateFunc       func(ctx context.Context, category *models.Category) error
	DeleteFunc       func(ctx context.Context, id uuid.UUID) error
}

func (m *MockCategoryRepository) Create(ctx context.Context, category *models.Category) error {
	return m.CreateFunc(ctx, category)
}

func (m *MockCategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	return m.GetByIDFunc(ctx, id)
}

func (m *MockCategoryRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Category, error) {
	return m.ListByUserIDFunc(ctx, userID)
}

func (m *MockCategoryRepository) ListSystem(ctx context.Context) ([]*models.Category, error) {
	return m.ListSystemFunc(ctx)
}

func (m *MockCategoryRepository) Update(ctx context.Context, category *models.Category) error {
	return m.UpdateFunc(ctx, category)
}

func (m *MockCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFunc(ctx, id)
}

func TestCategoryService_Create(t *testing.T) {
	tests := []struct {
		name        string
		req         models.CreateCategoryRequest
		expectError bool
		errorType   string
	}{
		{
			name: "valid expense category",
			req: models.CreateCategoryRequest{
				Name:  "Food",
				Type:  models.CategoryTypeExpense,
				Icon:  "🍔",
				Color: "#FF0000",
			},
			expectError: false,
		},
		{
			name: "valid income category",
			req: models.CreateCategoryRequest{
				Name:  "Salary",
				Type:  models.CategoryTypeIncome,
				Icon:  "💰",
				Color: "#00FF00",
			},
			expectError: false,
		},
		{
			name: "empty name",
			req: models.CreateCategoryRequest{
				Name: "",
				Type: models.CategoryTypeExpense,
			},
			expectError: true,
			errorType:   "name cannot be empty",
		},
		{
			name: "invalid type",
			req: models.CreateCategoryRequest{
				Name: "Invalid",
				Type: "invalid_type",
			},
			expectError: true,
			errorType:   "invalid category type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockCategoryRepository{
				CreateFunc: func(ctx context.Context, category *models.Category) error {
					return nil
				},
			}

			svc := NewCategoryService(mockRepo)
			userID := uuid.New()

			result, err := svc.Create(context.Background(), userID, tt.req)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if err.Error() != tt.errorType {
					t.Errorf("expected error %q, got %q", tt.errorType, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("expected result, got nil")
				return
			}

			if result.Name != tt.req.Name {
				t.Errorf("expected name %q, got %q", tt.req.Name, result.Name)
			}

			if result.Type != tt.req.Type {
				t.Errorf("expected type %q, got %q", tt.req.Type, result.Type)
			}

			if result.UserID == nil || *result.UserID != userID {
				t.Errorf("expected user ID %s, got %v", userID, result.UserID)
			}

			if result.IsSystem {
				t.Errorf("expected user category to not be system category")
			}
		})
	}
}

func TestCategoryService_Get(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()

	tests := []struct {
		name        string
		category    *models.Category
		requestID   uuid.UUID
		expectError bool
		errorMsg    string
	}{
		{
			name: "user owns category",
			category: &models.Category{
				ID:     categoryID,
				UserID: &userID,
				Name:   "Food",
				Type:   models.CategoryTypeExpense,
			},
			requestID:   categoryID,
			expectError: false,
		},
		{
			name: "system category accessible",
			category: &models.Category{
				ID:       categoryID,
				UserID:   nil,
				Name:     "Salary",
				Type:     models.CategoryTypeIncome,
				IsSystem: true,
			},
			requestID:   categoryID,
			expectError: false,
		},
		{
			name: "forbidden - different user",
			category: &models.Category{
				ID:     categoryID,
				UserID: &userID,
				Name:   "Private",
				Type:   models.CategoryTypeExpense,
			},
			requestID:   categoryID,
			expectError: true,
			errorMsg:    "forbidden access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otherUserID := uuid.New()
			requesterID := userID
			if tt.name == "forbidden - different user" {
				requesterID = otherUserID
			}

			mockRepo := &MockCategoryRepository{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Category, error) {
					return tt.category, nil
				},
			}

			svc := NewCategoryService(mockRepo)

			result, err := svc.Get(context.Background(), tt.requestID, requesterID)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.ID != tt.category.ID {
				t.Errorf("expected category ID %s, got %s", tt.category.ID, result.ID)
			}
		})
	}
}

func TestCategoryService_Delete(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()

	tests := []struct {
		name        string
		category    *models.Category
		requestID   uuid.UUID
		expectError bool
	}{
		{
			name: "delete user category",
			category: &models.Category{
				ID:     categoryID,
				UserID: &userID,
				Name:   "Food",
				Type:   models.CategoryTypeExpense,
			},
			requestID:   categoryID,
			expectError: false,
		},
		{
			name: "cannot delete system category",
			category: &models.Category{
				ID:       categoryID,
				UserID:   nil,
				Name:     "Salary",
				Type:     models.CategoryTypeIncome,
				IsSystem: true,
			},
			requestID:   categoryID,
			expectError: true,
		},
		{
			name: "forbidden - different user",
			category: &models.Category{
				ID:     categoryID,
				UserID: &userID,
				Name:   "Private",
				Type:   models.CategoryTypeExpense,
			},
			requestID:   categoryID,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otherUserID := uuid.New()
			requesterID := userID
			if tt.name == "forbidden - different user" {
				requesterID = otherUserID
			}

			deleted := false
			mockRepo := &MockCategoryRepository{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Category, error) {
					return tt.category, nil
				},
				DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
					deleted = true
					return nil
				},
			}

			svc := NewCategoryService(mockRepo)

			err := svc.Delete(context.Background(), tt.requestID, requesterID)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if deleted {
					t.Errorf("category should not be deleted")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !deleted {
				t.Errorf("expected category to be deleted")
			}
		})
	}
}

func TestCategoryService_Update(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()

	newName := "Updated Food"
	newIcon := "🍕"

	tests := []struct {
		name        string
		category    *models.Category
		req         models.UpdateCategoryRequest
		requestID   uuid.UUID
		expectError bool
	}{
		{
			name: "update user category",
			category: &models.Category{
				ID:        categoryID,
				UserID:    &userID,
				Name:      "Food",
				Type:      models.CategoryTypeExpense,
				Icon:      &newIcon,
				UpdatedAt: time.Now(),
			},
			req: models.UpdateCategoryRequest{
				Name: &newName,
				Icon: &newIcon,
			},
			requestID:   categoryID,
			expectError: false,
		},
		{
			name: "cannot update system category",
			category: &models.Category{
				ID:        categoryID,
				UserID:    nil,
				Name:      "Salary",
				Type:      models.CategoryTypeIncome,
				IsSystem:  true,
				UpdatedAt: time.Now(),
			},
			req: models.UpdateCategoryRequest{
				Name: &newName,
			},
			requestID:   categoryID,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otherUserID := uuid.New()
			requesterID := userID
			if tt.name == "forbidden - different user" {
				requesterID = otherUserID
			}

			updated := false
			mockRepo := &MockCategoryRepository{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Category, error) {
					return tt.category, nil
				},
				UpdateFunc: func(ctx context.Context, category *models.Category) error {
					updated = true
					return nil
				},
			}

			svc := NewCategoryService(mockRepo)

			err := svc.Update(context.Background(), tt.requestID, requesterID, tt.req)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if updated {
					t.Errorf("category should not be updated")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !updated {
				t.Errorf("expected category to be updated")
			}
		})
	}
}
