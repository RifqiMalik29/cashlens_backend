package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

// Mock Budget Repository
type MockBudgetRepository struct {
	CreateFunc        func(ctx context.Context, budget *models.Budget) error
	GetByIDFunc       func(ctx context.Context, id uuid.UUID) (*models.Budget, error)
	ListByUserIDFunc  func(ctx context.Context, userID uuid.UUID) ([]*models.Budget, error)
	UpdateFunc        func(ctx context.Context, budget *models.Budget) error
	DeleteFunc        func(ctx context.Context, id uuid.UUID) error
	GetByCategoryFunc func(ctx context.Context, userID, categoryID uuid.UUID, periodType models.BudgetPeriod, startDate, endDate time.Time) (*models.Budget, error)
}

func (m *MockBudgetRepository) Create(ctx context.Context, budget *models.Budget) error {
	return m.CreateFunc(ctx, budget)
}

func (m *MockBudgetRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Budget, error) {
	return m.GetByIDFunc(ctx, id)
}

func (m *MockBudgetRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Budget, error) {
	return m.ListByUserIDFunc(ctx, userID)
}

func (m *MockBudgetRepository) Update(ctx context.Context, budget *models.Budget) error {
	return m.UpdateFunc(ctx, budget)
}

func (m *MockBudgetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFunc(ctx, id)
}

func (m *MockBudgetRepository) GetByCategoryAndPeriod(ctx context.Context, userID, categoryID uuid.UUID, periodType models.BudgetPeriod, startDate, endDate time.Time) (*models.Budget, error) {
	return m.GetByCategoryFunc(ctx, userID, categoryID, periodType, startDate, endDate)
}

func TestBudgetService_Create(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()
	now := time.Now()
	endDate := now.AddDate(0, 1, 0)

	tests := []struct {
		name        string
		req         models.CreateBudgetRequest
		expectError bool
	}{
		{
			name: "valid monthly budget",
			req: models.CreateBudgetRequest{
				CategoryID: categoryID,
				Amount:     500000,
				PeriodType: models.BudgetPeriodMonthly,
				StartDate:  now,
				EndDate:    endDate,
			},
			expectError: false,
		},
		{
			name: "zero amount",
			req: models.CreateBudgetRequest{
				CategoryID: categoryID,
				Amount:     0,
				PeriodType: models.BudgetPeriodMonthly,
				StartDate:  now,
				EndDate:    endDate,
			},
			expectError: true,
		},
		{
			name: "invalid period type",
			req: models.CreateBudgetRequest{
				CategoryID: categoryID,
				Amount:     500000,
				PeriodType: "invalid",
				StartDate:  now,
				EndDate:    endDate,
			},
			expectError: true,
		},
		{
			name: "end before start",
			req: models.CreateBudgetRequest{
				CategoryID: categoryID,
				Amount:     500000,
				PeriodType: models.BudgetPeriodMonthly,
				StartDate:  endDate,
				EndDate:    now,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockBudgetRepository{
				CreateFunc: func(ctx context.Context, budget *models.Budget) error {
					return nil
				},
			}

			svc := NewBudgetService(mockRepo)

			result, err := svc.Create(context.Background(), userID, tt.req)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("expected result, got nil")
			}
		})
	}
}

func TestBudgetService_Get(t *testing.T) {
	userID := uuid.New()
	budgetID := uuid.New()

	tests := []struct {
		name        string
		budget      *models.Budget
		requestID   uuid.UUID
		expectError bool
	}{
		{
			name: "user owns budget",
			budget: &models.Budget{
				ID:     budgetID,
				UserID: userID,
				Amount: 500000,
			},
			requestID:   budgetID,
			expectError: false,
		},
		{
			name: "forbidden - different user",
			budget: &models.Budget{
				ID:     budgetID,
				UserID: uuid.New(),
				Amount: 500000,
			},
			requestID:   budgetID,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockBudgetRepository{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Budget, error) {
					return tt.budget, nil
				},
			}

			svc := NewBudgetService(mockRepo)

			result, err := svc.Get(context.Background(), tt.requestID, userID)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result.ID != tt.budget.ID {
				t.Errorf("expected budget ID %s, got %s", tt.budget.ID, result.ID)
			}
		})
	}
}

func TestBudgetService_Delete(t *testing.T) {
	userID := uuid.New()
	budgetID := uuid.New()

	tests := []struct {
		name        string
		budget      *models.Budget
		requestID   uuid.UUID
		expectError bool
	}{
		{
			name: "delete user budget",
			budget: &models.Budget{
				ID:     budgetID,
				UserID: userID,
				Amount: 500000,
			},
			requestID:   budgetID,
			expectError: false,
		},
		{
			name: "forbidden - different user",
			budget: &models.Budget{
				ID:     budgetID,
				UserID: uuid.New(),
				Amount: 500000,
			},
			requestID:   budgetID,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleted := false
			mockRepo := &MockBudgetRepository{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Budget, error) {
					return tt.budget, nil
				},
				DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
					deleted = true
					return nil
				},
			}

			svc := NewBudgetService(mockRepo)

			err := svc.Delete(context.Background(), tt.requestID, userID)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if deleted {
					t.Errorf("budget should not be deleted")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !deleted {
				t.Errorf("expected budget to be deleted")
			}
		})
	}
}
