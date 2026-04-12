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

type BudgetService interface {
	Create(ctx context.Context, userID uuid.UUID, req models.CreateBudgetRequest) (*models.Budget, error)
	Get(ctx context.Context, id, userID uuid.UUID) (*models.Budget, error)
	List(ctx context.Context, userID uuid.UUID) ([]*models.Budget, error)
	Update(ctx context.Context, id, userID uuid.UUID, req models.UpdateBudgetRequest) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type budgetService struct {
	budgetRepo repository.BudgetRepository
}

func NewBudgetService(budgetRepo repository.BudgetRepository) BudgetService {
	return &budgetService{budgetRepo: budgetRepo}
}

func (s *budgetService) Create(ctx context.Context, userID uuid.UUID, req models.CreateBudgetRequest) (*models.Budget, error) {
	if req.Amount <= 0 {
		return nil, errors.NewBadRequest("Budget amount must be greater than 0")
	}

	if req.PeriodType != models.BudgetPeriodWeekly && req.PeriodType != models.BudgetPeriodMonthly && req.PeriodType != models.BudgetPeriodYearly {
		return nil, errors.NewBadRequest("Invalid period type. Must be weekly, monthly, or yearly")
	}

	if req.EndDate.Before(req.StartDate) {
		return nil, errors.NewBadRequest("End date must be after start date")
	}

	if req.AlertThreshold != nil && (*req.AlertThreshold < 0 || *req.AlertThreshold > 100) {
		return nil, errors.NewBadRequest("Alert threshold must be between 0 and 100")
	}

	budget := &models.Budget{
		ID:             uuid.New(),
		UserID:         userID,
		CategoryID:     req.CategoryID,
		Amount:         req.Amount,
		PeriodType:     req.PeriodType,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		AlertThreshold: req.AlertThreshold,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := s.budgetRepo.Create(ctx, budget)
	if err != nil {
		return nil, fmt.Errorf("failed to create budget: %w", err)
	}

	return budget, nil
}

func (s *budgetService) Get(ctx context.Context, id, userID uuid.UUID) (*models.Budget, error) {
	budget, err := s.budgetRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	if budget.UserID != userID {
		return nil, errors.NewForbidden("Forbidden access")
	}

	return budget, nil
}

func (s *budgetService) List(ctx context.Context, userID uuid.UUID) ([]*models.Budget, error) {
	budgets, err := s.budgetRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list budgets: %w", err)
	}

	return budgets, nil
}

func (s *budgetService) Update(ctx context.Context, id, userID uuid.UUID, req models.UpdateBudgetRequest) error {
	budget, err := s.budgetRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get budget: %w", err)
	}

	if budget.UserID != userID {
		return errors.NewForbidden("Forbidden access")
	}

	if req.CategoryID != nil {
		budget.CategoryID = *req.CategoryID
	}

	if req.Amount != nil && *req.Amount > 0 {
		budget.Amount = *req.Amount
	}

	if req.PeriodType != nil {
		if *req.PeriodType != models.BudgetPeriodWeekly && *req.PeriodType != models.BudgetPeriodMonthly && *req.PeriodType != models.BudgetPeriodYearly {
			return errors.NewBadRequest("Invalid period type. Must be weekly, monthly, or yearly")
		}
		budget.PeriodType = *req.PeriodType
	}

	if req.StartDate != nil {
		budget.StartDate = *req.StartDate
	}

	if req.EndDate != nil {
		if req.EndDate.Before(budget.StartDate) {
			return errors.NewBadRequest("End date must be after start date")
		}
		budget.EndDate = *req.EndDate
	}

	if req.AlertThreshold != nil && (*req.AlertThreshold < 0 || *req.AlertThreshold > 100) {
		return errors.NewBadRequest("Alert threshold must be between 0 and 100")
	} else if req.AlertThreshold != nil {
		budget.AlertThreshold = req.AlertThreshold
	}

	budget.UpdatedAt = time.Now()

	err = s.budgetRepo.Update(ctx, budget)
	if err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}

	return nil
}

func (s *budgetService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	budget, err := s.budgetRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get budget: %w", err)
	}

	if budget.UserID != userID {
		return errors.NewForbidden("Forbidden access")
	}

	err = s.budgetRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}

	return nil
}
