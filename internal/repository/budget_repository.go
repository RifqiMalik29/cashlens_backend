package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

type BudgetRepository interface {
	Create(ctx context.Context, budget *models.Budget) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Budget, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Budget, error)
	Update(ctx context.Context, budget *models.Budget) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByCategoryAndPeriod(ctx context.Context, userID, categoryID uuid.UUID, periodType models.BudgetPeriod, startDate, endDate time.Time) (*models.Budget, error)
}

type budgetRepository struct {
	db *pgxpool.Pool
}

func NewBudgetRepository(db *pgxpool.Pool) BudgetRepository {
	return &budgetRepository{db: db}
}

func (r *budgetRepository) Create(ctx context.Context, budget *models.Budget) error {
	query := `INSERT INTO budgets (id, user_id, category_id, amount, period_type, start_date, end_date, alert_threshold, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Exec(ctx, query, budget.ID, budget.UserID, budget.CategoryID, budget.Amount, budget.PeriodType, budget.StartDate, budget.EndDate, budget.AlertThreshold, budget.CreatedAt, budget.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create budget: %w", err)
	}

	return nil
}

func (r *budgetRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Budget, error) {
	budget := &models.Budget{}
	query := `SELECT id, user_id, category_id, amount, period_type, start_date, end_date, alert_threshold, created_at, updated_at FROM budgets WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(&budget.ID, &budget.UserID, &budget.CategoryID, &budget.Amount, &budget.PeriodType, &budget.StartDate, &budget.EndDate, &budget.AlertThreshold, &budget.CreatedAt, &budget.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("budget not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	return budget, nil
}

func (r *budgetRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Budget, error) {
	query := `SELECT b.id, b.user_id, b.category_id, b.amount, b.period_type, b.start_date, b.end_date, b.alert_threshold, b.created_at, b.updated_at, c.id, c.user_id, c.name, c.type, c.icon, c.color, c.is_system, c.created_at, c.updated_at FROM budgets b JOIN categories c ON b.category_id = c.id WHERE b.user_id = $1 ORDER BY b.created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list budgets: %w", err)
	}
	defer rows.Close()

	budgets := []*models.Budget{}

	for rows.Next() {
		b := &models.Budget{}
		b.Category = &models.Category{}

		err := rows.Scan(&b.ID, &b.UserID, &b.CategoryID, &b.Amount, &b.PeriodType, &b.StartDate, &b.EndDate, &b.AlertThreshold, &b.CreatedAt, &b.UpdatedAt, &b.Category.ID, &b.Category.UserID, &b.Category.Name, &b.Category.Type, &b.Category.Icon, &b.Category.Color, &b.Category.IsSystem, &b.Category.CreatedAt, &b.Category.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget: %w", err)
		}

		budgets = append(budgets, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during iteration: %w", err)
	}

	return budgets, nil
}

func (r *budgetRepository) Update(ctx context.Context, budget *models.Budget) error {
	query := `UPDATE budgets SET category_id = COALESCE($2, category_id), amount = COALESCE($3, amount), period_type = COALESCE($4, period_type), start_date = COALESCE($5, start_date), end_date = COALESCE($6, end_date), alert_threshold = COALESCE($7, alert_threshold), updated_at = $8 WHERE id = $1`

	_, err := r.db.Exec(ctx, query, budget.ID, budget.CategoryID, budget.Amount, budget.PeriodType, budget.StartDate, budget.EndDate, budget.AlertThreshold, budget.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}

	return nil
}

func (r *budgetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM budgets WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}

	return nil
}

func (r *budgetRepository) GetByCategoryAndPeriod(ctx context.Context, userID, categoryID uuid.UUID, periodType models.BudgetPeriod, startDate, endDate time.Time) (*models.Budget, error) {
	budget := &models.Budget{}
	query := `SELECT id, user_id, category_id, amount, period_type, start_date, end_date, alert_threshold, created_at, updated_at FROM budgets WHERE user_id = $1 AND category_id = $2 AND period_type = $3 AND start_date <= $4 AND end_date >= $5 LIMIT 1`

	err := r.db.QueryRow(ctx, query, userID, categoryID, periodType, startDate, endDate).Scan(&budget.ID, &budget.UserID, &budget.CategoryID, &budget.Amount, &budget.PeriodType, &budget.StartDate, &budget.EndDate, &budget.AlertThreshold, &budget.CreatedAt, &budget.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget by category and period: %w", err)
	}

	return budget, nil
}
