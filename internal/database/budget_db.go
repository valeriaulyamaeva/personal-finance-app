package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"time"
)

func CreateBudget(pool *pgxpool.Pool, budget *models.Budget) error {
	// Проверяем, существует ли пользователь с таким user_id
	var userExists bool
	checkUserQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`
	err := pool.QueryRow(context.Background(), checkUserQuery, budget.UserID).Scan(&userExists)
	if err != nil {
		return fmt.Errorf("ошибка при проверке пользователя: %v", err)
	}
	if !userExists {
		return fmt.Errorf("пользователь с ID %d не существует", budget.UserID)
	}

	// Если пользователь существует, продолжаем вставку бюджета
	query := `
		INSERT INTO budgets (user_id, category_id, amount, remaining_amount, period, start_date, end_date) 
		VALUES ($1, $2, $3, $3, $4, $5, $6) 
		RETURNING id`
	err = pool.QueryRow(context.Background(), query,
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

func GetBudgetByID(pool *pgxpool.Pool, budgetID int) (*models.Budget, error) {
	query := `
		SELECT id, user_id, category_id, amount, period, start_date, end_date 
		FROM budgets 
		WHERE id = $1`

	budget := &models.Budget{}
	err := pool.QueryRow(context.Background(), query, budgetID).Scan(
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

func GetAllBudgets(pool *pgxpool.Pool) ([]models.Budget, error) {
	query := `SELECT id, user_id, category_id, amount, period, start_date, end_date FROM budgets`
	rows, err := pool.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении бюджетов: %v", err)
	}
	defer rows.Close()

	var budgets []models.Budget
	for rows.Next() {
		var budget models.Budget
		if err := rows.Scan(&budget.ID, &budget.UserID, &budget.CategoryID, &budget.Amount, &budget.Period, &budget.StartDate, &budget.EndDate); err != nil {
			return nil, err
		}
		budgets = append(budgets, budget)
	}
	return budgets, nil
}

func GetBudgetsByUserID(pool *pgxpool.Pool, userID int) ([]models.Budget, error) {
	query := `SELECT id, user_id, category_id, amount, period, start_date, end_date FROM budgets WHERE user_id = $1`

	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении бюджетов: %v", err)
	}
	defer rows.Close()

	var budgets []models.Budget
	for rows.Next() {
		var budget models.Budget
		if err := rows.Scan(&budget.ID, &budget.UserID, &budget.CategoryID, &budget.Amount, &budget.Period, &budget.StartDate, &budget.EndDate); err != nil {
			return nil, err
		}
		budgets = append(budgets, budget)
	}

	return budgets, nil
}

func UpdateBudget(pool *pgxpool.Pool, budget *models.Budget) error {
	query := `
		UPDATE budgets 
		SET category_id = $1, amount = $2, period = $3, start_date = $4, end_date = $5 
		WHERE id = $6`

	_, err := pool.Exec(context.Background(), query,
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

func DeleteBudget(pool *pgxpool.Pool, budgetID int) error {
	query := `
		DELETE FROM budgets 
		WHERE id = $1`

	result, err := pool.Exec(context.Background(), query, budgetID)
	if err != nil {
		return fmt.Errorf("ошибка удаления бюджета: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("бюджет с ID %d не найден", budgetID)
	}
	return nil
}

func DeductFromBudget(pool *pgxpool.Pool, categoryID int, amount float64, transactionDate time.Time) error {
	query := `
		UPDATE budgets 
		SET remaining_amount = remaining_amount - $1
		WHERE category_id = $2 
		AND remaining_amount >= $1
		AND $3 BETWEEN start_date AND end_date
		RETURNING remaining_amount
	`

	var remainingAmount float64
	err := pool.QueryRow(context.Background(), query, amount, categoryID, transactionDate).Scan(&remainingAmount)
	if err != nil {
		return fmt.Errorf("ошибка при вычитании из бюджета: %v", err)
	}

	if remainingAmount < 0 {
		return fmt.Errorf("превышен бюджет для категории")
	}

	return nil
}

func RenewBudgetPeriod(budget *models.Budget) {
	switch budget.Period {
	case "monthly":
		budget.StartDate = budget.EndDate.AddDate(0, 1, 0)
		budget.EndDate = budget.EndDate.AddDate(0, 1, 0)
	case "yearly":
		budget.StartDate = budget.EndDate.AddDate(1, 0, 0)
		budget.EndDate = budget.EndDate.AddDate(1, 0, 0)
	}
}

func UpdateExpiredBudgets(pool *pgxpool.Pool) error {
	query := `SELECT id, user_id, category_id, amount, remaining_amount, period, start_date, end_date FROM budgets WHERE end_date < CURRENT_DATE`
	rows, err := pool.Query(context.Background(), query)
	if err != nil {
		return fmt.Errorf("ошибка при получении истекших бюджетов: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var budget models.Budget
		if err := rows.Scan(&budget.ID, &budget.UserID, &budget.CategoryID, &budget.Amount, &budget.RemainingAmount, &budget.Period, &budget.StartDate, &budget.EndDate); err != nil {
			return fmt.Errorf("ошибка при сканировании бюджета: %v", err)
		}

		RenewBudgetPeriod(&budget)
		budget.RemainingAmount = budget.Amount

		updateQuery := `
			UPDATE budgets 
			SET start_date = $1, end_date = $2, remaining_amount = $3 
			WHERE id = $4
		`
		_, err := pool.Exec(context.Background(), updateQuery, budget.StartDate, budget.EndDate, budget.RemainingAmount, budget.ID)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении бюджета: %v", err)
		}
	}
	return nil
}
