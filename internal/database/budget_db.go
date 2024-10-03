package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func CreateBudget(conn *pgx.Conn, budget *models.Budget) error {
	query := `
		INSERT INTO budgets (user_id, category_id, amount, period, start_date, end_date) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id`

	err := conn.QueryRow(context.Background(), query,
		budget.UserID,
		budget.CategoryID,
		budget.Amount,
		budget.Period,
		budget.StartDate,
		budget.EndDate).Scan(&budget.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении бюджета: %v", err)
	}
	return nil
}

func GetBudgetByID(conn *pgx.Conn, budgetID int) (*models.Budget, error) {
	query := `
		SELECT id, user_id, category_id, amount, period, start_date, end_date 
		FROM budgets 
		WHERE id = $1`

	budget := &models.Budget{}
	err := conn.QueryRow(context.Background(), query, budgetID).Scan(
		&budget.ID,
		&budget.UserID,
		&budget.CategoryID,
		&budget.Amount,
		&budget.Period,
		&budget.StartDate,
		&budget.EndDate,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("бюджет с ID %d не найден", budgetID)
		}
		return nil, fmt.Errorf("ошибка при получении бюджета: %v", err)
	}

	return budget, nil
}

func UpdateBudget(conn *pgx.Conn, budget *models.Budget) error {
	query := `
		UPDATE budgets 
		SET category_id = $1, amount = $2, period = $3, start_date = $4, end_date = $5 
		WHERE id = $6`

	_, err := conn.Exec(context.Background(), query,
		budget.CategoryID,
		budget.Amount,
		budget.Period,
		budget.StartDate,
		budget.EndDate,
		budget.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления бюджета: %v", err)
	}
	return nil
}

func DeleteBudget(conn *pgx.Conn, budgetID int) error {
	query := `
		DELETE FROM budgets 
		WHERE id = $1`

	result, err := conn.Exec(context.Background(), query, budgetID)
	if err != nil {
		return fmt.Errorf("ошибка удаления бюджета: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("бюджет с ID %d не найден", budgetID)
	}
	return nil
}
