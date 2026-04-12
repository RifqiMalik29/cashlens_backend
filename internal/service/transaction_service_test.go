package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/errors"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

// Mock Transaction Repository
type MockTransactionRepository struct {
	CreateFunc          func(ctx context.Context, tx *models.Transaction) error
	GetByIDFunc         func(ctx context.Context, id uuid.UUID) (*models.Transaction, error)
	ListByUserIDFunc    func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.TransactionWithCategory, error)
	ListByDateRangeFunc func(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*models.TransactionWithCategory, error)
	UpdateFunc          func(ctx context.Context, tx *models.Transaction) error
	DeleteFunc          func(ctx context.Context, id uuid.UUID) error
}

func (m *MockTransactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	return m.CreateFunc(ctx, tx)
}

func (m *MockTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	return m.GetByIDFunc(ctx, id)
}

func (m *MockTransactionRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.TransactionWithCategory, error) {
	return m.ListByUserIDFunc(ctx, userID, limit, offset)
}

func (m *MockTransactionRepository) ListByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*models.TransactionWithCategory, error) {
	return m.ListByDateRangeFunc(ctx, userID, startDate, endDate)
}

func (m *MockTransactionRepository) Update(ctx context.Context, tx *models.Transaction) error {
	return m.UpdateFunc(ctx, tx)
}

func (m *MockTransactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFunc(ctx, id)
}

// Mock Draft Repository
type MockDraftRepository struct {
	CreateFunc       func(ctx context.Context, draft *models.DraftTransaction) error
	GetByIDFunc      func(ctx context.Context, id uuid.UUID) (*models.DraftTransaction, error)
	ListByUserIDFunc func(ctx context.Context, userID uuid.UUID, status models.DraftStatus) ([]*models.DraftTransaction, error)
	UpdateFunc       func(ctx context.Context, draft *models.DraftTransaction) error
	DeleteFunc       func(ctx context.Context, id uuid.UUID) error
	ConfirmFunc      func(ctx context.Context, draftID uuid.UUID, txID uuid.UUID) error
}

func (m *MockDraftRepository) Create(ctx context.Context, draft *models.DraftTransaction) error {
	return m.CreateFunc(ctx, draft)
}

func (m *MockDraftRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DraftTransaction, error) {
	return m.GetByIDFunc(ctx, id)
}

func (m *MockDraftRepository) ListByUserID(ctx context.Context, userID uuid.UUID, status models.DraftStatus) ([]*models.DraftTransaction, error) {
	return m.ListByUserIDFunc(ctx, userID, status)
}

func (m *MockDraftRepository) Update(ctx context.Context, draft *models.DraftTransaction) error {
	return m.UpdateFunc(ctx, draft)
}

func (m *MockDraftRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFunc(ctx, id)
}

func (m *MockDraftRepository) Confirm(ctx context.Context, draftID uuid.UUID, txID uuid.UUID) error {
	return m.ConfirmFunc(ctx, draftID, txID)
}

func TestTransactionService_Create(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()
	now := time.Now()

	tests := []struct {
		name        string
		req         models.CreateTransactionRequest
		expectError bool
	}{
		{
			name: "valid transaction",
			req: models.CreateTransactionRequest{
				CategoryID:      categoryID,
				Amount:          50000,
				Description:     "Lunch",
				TransactionDate: now,
			},
			expectError: false,
		},
		{
			name: "zero amount",
			req: models.CreateTransactionRequest{
				CategoryID:      categoryID,
				Amount:          0,
				Description:     "Invalid",
				TransactionDate: now,
			},
			expectError: true,
		},
		{
			name: "zero date",
			req: models.CreateTransactionRequest{
				CategoryID:      categoryID,
				Amount:          50000,
				Description:     "Invalid",
				TransactionDate: time.Time{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockTransactionRepository{
				CreateFunc: func(ctx context.Context, tx *models.Transaction) error {
					return nil
				},
			}

			svc := NewTransactionService(mockRepo)

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

func TestTransactionService_Get(t *testing.T) {
	userID := uuid.New()
	txID := uuid.New()

	tests := []struct {
		name        string
		transaction *models.Transaction
		requestID   uuid.UUID
		expectError bool
	}{
		{
			name: "user owns transaction",
			transaction: &models.Transaction{
				ID:     txID,
				UserID: userID,
				Amount: 50000,
			},
			requestID:   txID,
			expectError: false,
		},
		{
			name: "forbidden - different user",
			transaction: &models.Transaction{
				ID:     txID,
				UserID: uuid.New(),
				Amount: 50000,
			},
			requestID:   txID,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockTransactionRepository{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
					return tt.transaction, nil
				},
			}

			svc := NewTransactionService(mockRepo)

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

			if result.ID != tt.transaction.ID {
				t.Errorf("expected transaction ID %s, got %s", tt.transaction.ID, result.ID)
			}
		})
	}
}

func TestDraftService_Create(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()
	now := time.Now()

	tests := []struct {
		name        string
		req         models.CreateDraftRequest
		expectError bool
	}{
		{
			name: "valid draft from telegram",
			req: models.CreateDraftRequest{
				CategoryID:      &categoryID,
				Amount:          &[]float64{50000}[0],
				Description:     &[]string{"Lunch"}[0],
				TransactionDate: &now,
				Source:          models.DraftSourceTelegram,
			},
			expectError: false,
		},
		{
			name: "empty source",
			req: models.CreateDraftRequest{
				Source: "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDraftRepo := &MockDraftRepository{
				CreateFunc: func(ctx context.Context, draft *models.DraftTransaction) error {
					return nil
				},
			}
			mockTxRepo := &MockTransactionRepository{}

			svc := NewDraftService(mockDraftRepo, mockTxRepo)

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

func TestDraftService_Confirm(t *testing.T) {
	userID := uuid.New()
	draftID := uuid.New()
	categoryID := uuid.New()
	now := time.Now()
	desc := "Lunch"

	tests := []struct {
		name        string
		draft       *models.DraftTransaction
		expectError bool
	}{
		{
			name: "confirm pending draft",
			draft: &models.DraftTransaction{
				ID:              draftID,
				UserID:          userID,
				CategoryID:      &categoryID,
				Amount:          &[]float64{50000}[0],
				Description:     &desc,
				TransactionDate: &now,
				Status:          models.DraftStatusPending,
			},
			expectError: false,
		},
		{
			name: "cannot confirm already confirmed draft",
			draft: &models.DraftTransaction{
				ID:     draftID,
				UserID: userID,
				Status: models.DraftStatusConfirmed,
			},
			expectError: true,
		},
		{
			name: "forbidden - different user",
			draft: &models.DraftTransaction{
				ID:     draftID,
				UserID: uuid.New(),
				Status: models.DraftStatusPending,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confirmed := false
			mockDraftRepo := &MockDraftRepository{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.DraftTransaction, error) {
					return tt.draft, nil
				},
				ConfirmFunc: func(ctx context.Context, draftID uuid.UUID, txID uuid.UUID) error {
					confirmed = true
					return nil
				},
			}
			mockTxRepo := &MockTransactionRepository{
				CreateFunc: func(ctx context.Context, tx *models.Transaction) error {
					return nil
				},
			}

			svc := NewDraftService(mockDraftRepo, mockTxRepo)

			req := models.ConfirmDraftRequest{
				CategoryID:      categoryID,
				Amount:          50000,
				Description:     "Lunch",
				TransactionDate: now,
			}

			result, err := svc.Confirm(context.Background(), draftID, userID, req)

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
				t.Errorf("expected transaction result, got nil")
			}

			if !confirmed {
				t.Errorf("expected draft to be confirmed")
			}
		})
	}
}

func TestErrors_AppError(t *testing.T) {
	tests := []struct {
		name       string
		err        *errors.AppError
		wantStatus int
	}{
		{
			name:       "bad request",
			err:        errors.NewBadRequest("invalid input"),
			wantStatus: 400,
		},
		{
			name:       "forbidden",
			err:        errors.NewForbidden("access denied"),
			wantStatus: 403,
		},
		{
			name:       "not found",
			err:        errors.NewNotFound("resource not found"),
			wantStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.StatusCode() != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, tt.err.StatusCode())
			}

			if tt.err.Error() == "" {
				t.Errorf("expected error message, got empty")
			}
		})
	}
}
