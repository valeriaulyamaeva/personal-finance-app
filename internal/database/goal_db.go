package database

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal" // Для точных денежных значений
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"log"
)

// CreateGoal добавляет новую цель в базу данных
func CreateGoal(pool *pgxpool.Pool, goal *models.Goal) error {
	query := `
		INSERT INTO goals (user_id, amount, current_amount, target_date, name, created_at, status) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) 
		RETURNING id`
	err := pool.QueryRow(context.Background(), query,
		goal.UserID,
		goal.Amount,
		goal.CurrentAmount,
		goal.TargetDate,
		goal.Name,
		goal.CreatedAt,
		goal.Status).Scan(&goal.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении цели: %v", err)
	}
	return nil
}

func GetGoalByID(pool *pgxpool.Pool, userID int) (*models.Goal, error) {
	var goal models.Goal
	query := `SELECT id, user_id, amount, current_amount, target_date, name, created_at, status, currency
              FROM goals WHERE user_id = $1`

	// Логируем запрос для диагностики
	log.Printf("Запрос к базе данных: %s с параметром userID=%d", query, userID)

	row := pool.QueryRow(context.Background(), query, userID)
	err := row.Scan(
		&goal.ID,
		&goal.UserID,
		&goal.Amount,
		&goal.CurrentAmount,
		&goal.TargetDate,
		&goal.Name,
		&goal.CreatedAt,
		&goal.Status,
		&goal.Currency,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Нет записи для данного userID
			log.Printf("Цель для пользователя с ID %d не найдена", userID)
			return nil, fmt.Errorf("цель не найдена для пользователя с ID %d", userID)
		}
		log.Printf("Ошибка при выполнении запроса к базе данных для userID=%d: %v", userID, err)
		return nil, fmt.Errorf("ошибка при получении цели: %v", err)
	}

	return &goal, nil
}

// GetAllGoals извлекает все цели пользователя
func GetAllGoals(pool *pgxpool.Pool, userID int) ([]models.Goal, error) {
	query := `SELECT id, user_id, amount, current_amount, target_date, name, created_at, status FROM goals WHERE user_id = $1`
	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении целей: %v", err)
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var goal models.Goal
		if err := rows.Scan(&goal.ID, &goal.UserID, &goal.Amount, &goal.CurrentAmount, &goal.TargetDate, &goal.Name, &goal.CreatedAt, &goal.Status); err != nil {
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
		SET amount = $1, current_amount = $2, target_date = $3, name = $4, status = $5 
		WHERE id = $6`
	_, err := pool.Exec(context.Background(), query,
		goal.Amount,
		goal.CurrentAmount,
		goal.TargetDate,
		goal.Name,
		goal.Status,
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

// / UpdateGoalProgress обновляет поле current_amount для цели в таблице Goals
func UpdateGoalProgress(pool *pgxpool.Pool, goalID int, progressAmount decimal.Decimal) error {
	// Start a transaction to ensure atomicity
	tx, err := pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("ошибка при начале транзакции: %v", err)
	}
	defer tx.Rollback(context.Background())

	// Запрос для получения текущего прогресса и целевого значения
	query := `
        SELECT current_amount, amount, status 
        FROM Goals 
        WHERE id = $1;
    `
	var currentAmount, goalAmount decimal.Decimal
	var status string

	// Получаем текущие данные цели
	err = tx.QueryRow(context.Background(), query, goalID).Scan(&currentAmount, &goalAmount, &status)
	if err != nil {
		return fmt.Errorf("ошибка при получении данных цели: %v", err)
	}
	log.Printf("Цель #%d: текущий прогресс = %s, цель = %s, статус = %s", goalID, currentAmount.String(), goalAmount.String(), status)

	// Обновляем прогресс
	newAmount := currentAmount.Add(progressAmount)
	log.Printf("Новый прогресс для цели #%d: %s", goalID, newAmount.String())

	updateProgressQuery := `
        UPDATE Goals 
        SET current_amount = $1 
        WHERE id = $2
        RETURNING current_amount;
    `
	err = tx.QueryRow(context.Background(), updateProgressQuery, newAmount, goalID).Scan(&currentAmount)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении прогресса: %v", err)
	}
	log.Printf("Обновленный прогресс для цели #%d: %s", goalID, currentAmount.String())

	// Если текущая сумма достигла или превысила целевую, обновляем статус
	if currentAmount.GreaterThanOrEqual(goalAmount) && status != "achieved" {
		updateStatusQuery := `
            UPDATE Goals 
            SET status = 'achieved' 
            WHERE id = $1;
        `
		_, err := tx.Exec(context.Background(), updateStatusQuery, goalID)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении статуса цели: %v", err)
		}
		// Логируем успешное обновление статуса
		log.Printf("Цель #%d: статус изменен на 'achieved'", goalID)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("ошибка при завершении транзакции: %v", err)
	}
	log.Printf("Транзакция успешно завершена для цели #%d", goalID)

	return nil
}

// AddMoneyToGoal добавляет деньги в цель и проверяет достижение цели
func AddMoneyToGoal(pool *pgxpool.Pool, goalID int, money decimal.Decimal) error {
	// Start a transaction to ensure atomicity
	tx, err := pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("ошибка при начале транзакции: %v", err)
	}
	defer tx.Rollback(context.Background())

	// Запрос для получения текущего прогресса и целевого значения
	query := `
        SELECT current_amount, amount, status 
        FROM goals 
        WHERE id = $1;
    `
	var currentAmount, goalAmount decimal.Decimal
	var status string

	// Получаем текущие данные цели
	err = tx.QueryRow(context.Background(), query, goalID).Scan(&currentAmount, &goalAmount, &status)
	if err != nil {
		return fmt.Errorf("ошибка при получении данных цели: %v", err)
	}

	// Обновляем прогресс
	newAmount := currentAmount.Add(money)
	updateProgressQuery := `
        UPDATE goals 
        SET current_amount = $1 
        WHERE id = $2
        RETURNING current_amount;
    `
	err = tx.QueryRow(context.Background(), updateProgressQuery, newAmount, goalID).Scan(&currentAmount)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении прогресса: %v", err)
	}

	// Логируем обновленный прогресс
	log.Printf("Цель #%d: новый прогресс = %s", goalID, currentAmount.String())

	// Если текущая сумма достигла или превысила целевую, обновляем статус
	if currentAmount.GreaterThanOrEqual(goalAmount) && status != "achieved" {
		updateStatusQuery := `
            UPDATE goals 
            SET status = 'achieved' 
            WHERE id = $1;
        `
		_, err := tx.Exec(context.Background(), updateStatusQuery, goalID)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении статуса цели: %v", err)
		}
		// Логируем успешное обновление статуса
		log.Printf("Цель #%d: статус изменен на 'achieved'", goalID)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("ошибка при завершении транзакции: %v", err)
	}

	return nil
}
