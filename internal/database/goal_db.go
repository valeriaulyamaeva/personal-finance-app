package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal" // Для точных денежных значений
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

// CreateGoal добавляет новую цель в базу данных
func CreateGoal(pool *pgxpool.Pool, goal *models.Goal) error {
	query := `
		INSERT INTO goals (user_id, amount, current_amount, target_date, name, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id`
	err := pool.QueryRow(context.Background(), query,
		goal.UserID,
		goal.Amount,
		goal.Current,
		goal.Deadline,
		goal.Name,
		goal.CreatedAt).Scan(&goal.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении цели: %v", err)
	}
	return nil
}

// GetGoalByID извлекает цель по ID
func GetGoalByID(pool *pgxpool.Pool, goalID int) (*models.Goal, error) {
	query := `
		SELECT id, user_id, amount, current_amount, target_date, name, created_at 
		FROM goals 
		WHERE id = $1`

	goal := &models.Goal{}
	err := pool.QueryRow(context.Background(), query, goalID).Scan(
		&goal.ID,
		&goal.UserID,
		&goal.Amount,
		&goal.Current,
		&goal.Deadline,
		&goal.Name,
		&goal.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении цели: %v", err)
	}
	return goal, nil
}

// GetAllGoals извлекает все цели пользователя
func GetAllGoals(pool *pgxpool.Pool, userID int) ([]models.Goal, error) {
	query := `SELECT id, user_id, amount, current_amount, target_date, name, created_at FROM goals WHERE user_id = $1`
	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении целей: %v", err)
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var goal models.Goal
		if err := rows.Scan(&goal.ID, &goal.UserID, &goal.Amount, &goal.Current, &goal.Deadline, &goal.Name, &goal.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}

	return goals, nil
}

// UpdateGoal обновляет информацию о цели
func UpdateGoal(pool *pgxpool.Pool, goal *models.Goal) error {
	query := `
		UPDATE goals 
		SET amount = $1, current_amount = $2, target_date = $3, name = $4 
		WHERE id = $5`
	_, err := pool.Exec(context.Background(), query,
		goal.Amount,
		goal.Current,
		goal.Deadline,
		goal.Name,
		goal.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления цели: %v", err)
	}
	return nil
}

// DeleteGoal удаляет цель по ID
func DeleteGoal(pool *pgxpool.Pool, goalID int) error {
	query := `
		DELETE FROM goals 
		WHERE id = $1`
	result, err := pool.Exec(context.Background(), query, goalID)
	if err != nil {
		return fmt.Errorf("ошибка удаления цели: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("цель с ID %d не найдена", goalID)
	}
	return nil
}

// AddProgressToGoal обновляет прогресс по цели и проверяет достижение цели
func AddProgressToGoal(pool *pgxpool.Pool, goalID int, progress decimal.Decimal) error {
	query := `
		UPDATE goals 
		SET current_amount = current_amount + $1 
		WHERE id = $2 AND current_amount + $1 <= amount
		RETURNING current_amount, amount`
	var current, amount decimal.Decimal
	err := pool.QueryRow(context.Background(), query, progress, goalID).Scan(&current, &amount)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении прогресса к цели: %v", err)
	}
	if current.GreaterThanOrEqual(amount) {
		// Обновляем статус цели, если она достигнута
		err := updateGoalStatus(pool, goalID, "achieved")
		if err != nil {
			return fmt.Errorf("не удалось обновить статус цели на 'achieved': %v", err)
		}
	}
	return nil
}

// updateGoalStatus обновляет статус цели
func updateGoalStatus(pool *pgxpool.Pool, goalID int, status string) error {
	query := `
		UPDATE goals
		SET status = $1
		WHERE id = $2`
	_, err := pool.Exec(context.Background(), query, status, goalID)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса цели: %v", err)
	}
	return nil
}
