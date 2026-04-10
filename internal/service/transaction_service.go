package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type TransactionService interface {
	Create(ctx context.Context, userID uuid.UUID, req models.CreateTransactionRequest) (*models.Transaction, error)
	Get(ctx context.Context, id, userID uuid.UUID) (*models.Transaction, error)
	List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.TransactionWithCategory, error)
	ListByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*models.TransactionWithCategory, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type transactionService struct {
	transactionRepo repository.TransactionRepository
}

func NewTransactionService(transactionRepo repository.TransactionRepository) TransactionService {
	return &transactionService{transactionRepo: transactionRepo}
}

// TODO: Implement methods (placeholder)
func (s *transactionService) Create(ctx context.Context, userID uuid.UUID, req models.CreateTransactionRequest) (*models.Transaction, error) {
	return nil, nil
}

func (s *transactionService) Get(ctx context.Context, id, userID uuid.UUID) (*models.Transaction, error) {
	return nil, nil
}

func (s *transactionService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.TransactionWithCategory, error) {
	return nil, nil
}

func (s *transactionService) ListByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*models.TransactionWithCategory, error) {
	return nil, nil
}

func (s *transactionService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return nil
}
