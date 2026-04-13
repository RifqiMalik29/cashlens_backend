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

type TransactionService interface {
	Create(ctx context.Context, userID uuid.UUID, req models.CreateTransactionRequest) (*models.Transaction, error)
	Get(ctx context.Context, id, userID uuid.UUID) (*models.Transaction, error)
	List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.TransactionWithCategory, error)
	ListByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*models.TransactionWithCategory, error)
	Update(ctx context.Context, id, userID uuid.UUID, req models.UpdateTransactionRequest) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type transactionService struct {
	transactionRepo repository.TransactionRepository
	quotaService    QuotaService
}

func NewTransactionService(transactionRepo repository.TransactionRepository, quotaService QuotaService) TransactionService {
	return &transactionService{
		transactionRepo: transactionRepo,
		quotaService:    quotaService,
	}
}

func (s *transactionService) Create(ctx context.Context, userID uuid.UUID, req models.CreateTransactionRequest) (*models.Transaction, error) {
	if req.Amount <= 0 {
		return nil, errors.NewBadRequest("Amount must be greater than 0")
	}

	if req.TransactionDate.IsZero() {
		return nil, errors.NewBadRequest("Transaction date is required")
	}

	// Atomic quota check + increment (prevents TOCTOU race condition)
	if err := s.quotaService.CheckAndIncrementTransactionQuota(ctx, userID); err != nil {
		return nil, err
	}

	transaction := &models.Transaction{
		ID:              uuid.New(),
		UserID:          userID,
		CategoryID:      req.CategoryID,
		Amount:          req.Amount,
		Description:     &req.Description,
		TransactionDate: req.TransactionDate,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err := s.transactionRepo.Create(ctx, transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return transaction, nil
}

func (s *transactionService) Get(ctx context.Context, id, userID uuid.UUID) (*models.Transaction, error) {
	tx, err := s.transactionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if tx.UserID != userID {
		return nil, errors.NewForbidden("Forbidden access")
	}

	return tx, nil
}

func (s *transactionService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.TransactionWithCategory, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	transactions, err := s.transactionRepo.ListByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	return transactions, nil
}

func (s *transactionService) ListByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*models.TransactionWithCategory, error) {
	transactions, err := s.transactionRepo.ListByDateRange(ctx, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions by date range: %w", err)
	}

	return transactions, nil
}

func (s *transactionService) Update(ctx context.Context, id, userID uuid.UUID, req models.UpdateTransactionRequest) error {
	tx, err := s.transactionRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	if tx.UserID != userID {
		return errors.NewForbidden("Forbidden access")
	}

	if req.CategoryID != nil {
		tx.CategoryID = *req.CategoryID
	}

	if req.Amount != nil && *req.Amount > 0 {
		tx.Amount = *req.Amount
	}

	if req.Description != nil {
		tx.Description = req.Description
	}

	if req.TransactionDate != nil {
		tx.TransactionDate = *req.TransactionDate
	}

	tx.UpdatedAt = time.Now()

	err = s.transactionRepo.Update(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

func (s *transactionService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	tx, err := s.transactionRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	if tx.UserID != userID {
		return errors.NewForbidden("Forbidden access")
	}

	err = s.transactionRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	return nil
}
