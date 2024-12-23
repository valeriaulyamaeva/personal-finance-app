package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"log"
	"time"
)

func CreateTransaction(pool *pgxpool.Pool, transaction *models.Transaction) error {
	// Начинаем с создания транзакции
	query := `
		INSERT INTO transactions (user_id, category_id, amount, description, transaction_date, type, goal_id) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) 
		RETURNING id`

	err := pool.QueryRow(context.Background(), query,
		transaction.UserID,
		transaction.CategoryID,
		transaction.Amount,
		transaction.Description,
		transaction.Date,
		transaction.Type,
		transaction.GoalID).Scan(&transaction.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении транзакции: %v", err)
	}

	// Если транзакция привязана к цели, обновляем баланс этой цели
	if transaction.Type == "goal" && transaction.GoalID != nil {
		// Обновляем баланс цели (расход или доход)
		err := updateGoalBalance(pool, *transaction.GoalID, transaction.Amount, transaction.Type)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении баланса цели: %v", err)
		}
	}

	return nil
}

func GetTransactionByID(pool *pgxpool.Pool, transactionID int) (*models.Transaction, error) {
	query := `
		SELECT id, user_id, category_id, amount, description, transaction_date, type
		FROM transactions 
		WHERE id = $1`

	transaction := &models.Transaction{}
	err := pool.QueryRow(context.Background(), query, transactionID).Scan(
		&transaction.ID,
		&transaction.UserID,
		&transaction.CategoryID,
		&transaction.Amount,
		&transaction.Description,
		&transaction.Date,
		&transaction.Type,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("транзакция с ID %d не найдена", transactionID)
		}
		return nil, fmt.Errorf("ошибка при получении транзакции: %v", err)
	}

	return transaction, nil
}

func GetTransactionsByUserID(pool *pgxpool.Pool, userID int) ([]models.Transaction, error) {
	query := `
        SELECT id, user_id, category_id, amount, description, transaction_date, type
        FROM transactions
        WHERE user_id = $1`

	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении транзакций: %v", err)
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var transaction models.Transaction
		if err := rows.Scan(
			&transaction.ID,
			&transaction.UserID,
			&transaction.CategoryID,
			&transaction.Amount,
			&transaction.Description,
			&transaction.Date,
			&transaction.Type,
		); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании транзакции: %v", err)
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func GetAllTransactions(pool *pgxpool.Pool) ([]*models.Transaction, error) {
	query := `
		SELECT id, user_id, category_id, amount, description, transaction_date, type
		FROM transactions`

	rows, err := pool.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении списка транзакций: %v", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		transaction := &models.Transaction{}
		if err := rows.Scan(
			&transaction.ID,
			&transaction.UserID,
			&transaction.CategoryID,
			&transaction.Amount,
			&transaction.Description,
			&transaction.Date,
			&transaction.Type,
		); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании транзакции: %v", err)
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func UpdateTransaction(pool *pgxpool.Pool, transaction *models.Transaction) error {
	// Получаем старую сумму для корректировки баланса цели
	var oldAmount float64
	selectQuery := `
		SELECT amount 
		FROM transactions 
		WHERE id = $1`
	err := pool.QueryRow(context.Background(), selectQuery, transaction.ID).Scan(&oldAmount)
	if err != nil {
		return fmt.Errorf("ошибка при получении старой суммы транзакции: %v", err)
	}

	// Обновляем саму транзакцию
	query := `
		UPDATE transactions 
		SET category_id = $1, amount = $2, description = $3, transaction_date = $4, type = $5
		WHERE id = $6`

	_, err = pool.Exec(context.Background(), query,
		transaction.CategoryID,
		transaction.Amount,
		transaction.Description,
		transaction.Date,
		transaction.Type,
		transaction.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления транзакции: %v", err)
	}

	// Если транзакция привязана к цели, обновляем баланс цели
	if transaction.GoalID != nil {
		// Если транзакция изменяет баланс цели, откатываем старую сумму и добавляем новую
		err := updateGoalBalance(pool, *transaction.GoalID, oldAmount, "expense")
		if err != nil {
			return fmt.Errorf("ошибка при обновлении баланса цели при изменении: %v", err)
		}
		err = updateGoalBalance(pool, *transaction.GoalID, transaction.Amount, transaction.Type)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении баланса цели при изменении: %v", err)
		}
	}

	return nil
}

func DeleteTransaction(pool *pgxpool.Pool, transactionID int) error {
	// Получаем информацию о транзакции перед удалением
	var transaction models.Transaction
	selectQuery := `
		SELECT user_id, category_id, amount, description, transaction_date, type, goal_id
		FROM transactions 
		WHERE id = $1`
	err := pool.QueryRow(context.Background(), selectQuery, transactionID).Scan(
		&transaction.UserID,
		&transaction.CategoryID,
		&transaction.Amount,
		&transaction.Description,
		&transaction.Date,
		&transaction.Type,
		&transaction.GoalID,
	)
	if err != nil {
		return fmt.Errorf("ошибка при получении транзакции для удаления: %v", err)
	}

	// Удаляем транзакцию
	query := `DELETE FROM transactions WHERE id = $1`
	result, err := pool.Exec(context.Background(), query, transactionID)
	if err != nil {
		return fmt.Errorf("ошибка удаления транзакции: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("транзакция с ID %d не найдена", transactionID)
	}

	// Если транзакция привязана к цели, обновляем баланс цели
	if transaction.GoalID != nil {
		// Если транзакция была расходом, нужно добавить сумму обратно к балансу цели
		err := updateGoalBalance(pool, *transaction.GoalID, transaction.Amount, "income")
		if err != nil {
			return fmt.Errorf("ошибка при обновлении баланса цели после удаления: %v", err)
		}
		// Если транзакция была доходом, нужно вычесть сумму из баланса цели
		err = updateGoalBalance(pool, *transaction.GoalID, transaction.Amount, "expense")
		if err != nil {
			return fmt.Errorf("ошибка при обновлении баланса цели после удаления: %v", err)
		}
	}

	return nil
}

func MoveTransactionsToHistory(pool *pgxpool.Pool) error {
	now := time.Now()
	currentMonth := int(now.Month())
	currentYear := now.Year()

	tx, err := pool.Begin(context.Background())
	if err != nil {
		log.Printf("Ошибка начала транзакции: %v", err)
		return err
	}
	defer tx.Rollback(context.Background())

	insertQuery := `
		INSERT INTO transactionhistory (
			user_id, category_id, amount, description, transaction_date, type, op_date, op_type, user_name
		)
		SELECT 
			t.user_id, 
			t.category_id, 
			t.amount, 
			t.description, 
			t.transaction_date, 
			t.type, 
			NOW() AS op_date, 
			'archived' AS op_type, 
			u.name AS user_name
		FROM transactions t
		INNER JOIN users u ON t.user_id = u.id
		WHERE EXTRACT(MONTH FROM t.transaction_date) != $1 OR EXTRACT(YEAR FROM t.transaction_date) != $2`

	res, err := tx.Exec(context.Background(), insertQuery, currentMonth, currentYear)
	if err != nil {
		log.Printf("Ошибка переноса транзакций в transactionhistory: %v", err)
		return err
	}

	insertedCount := res.RowsAffected()
	log.Printf("Перенесено транзакций: %d", insertedCount)

	// Удаление перенесённых транзакций
	deleteQuery := `
		DELETE FROM transactions
		WHERE EXTRACT(MONTH FROM transaction_date) != $1 OR EXTRACT(YEAR FROM transaction_date) != $2`

	res, err = tx.Exec(context.Background(), deleteQuery, currentMonth, currentYear)
	if err != nil {
		log.Printf("Ошибка удаления старых транзакций из transactions: %v", err)
		return err
	}

	deletedCount := res.RowsAffected()
	log.Printf("Удалено транзакций: %d", deletedCount)

	// Завершаем транзакцию
	if err := tx.Commit(context.Background()); err != nil {
		log.Printf("Ошибка фиксации транзакции: %v", err)
		return err
	}

	log.Println("Успешно перенесены транзакции в transactionhistory.")
	return nil
}

func updateGoalBalance(pool *pgxpool.Pool, goalID int, amount float64, transactionType string) error {
	var newBalance float64
	var currentAmount float64
	var targetAmount float64
	var goalStatus string

	// Получаем текущие данные о цели
	selectQuery := `SELECT current_amount, amount, status FROM goals WHERE id = $1`
	err := pool.QueryRow(context.Background(), selectQuery, goalID).Scan(&currentAmount, &targetAmount, &goalStatus)
	if err != nil {
		return fmt.Errorf("ошибка при получении данных цели: %v", err)
	}

	// Если транзакция типа "expense", уменьшаем баланс цели
	if transactionType == "expense" {
		newBalance = currentAmount - amount
	} else if transactionType == "income" {
		// Если транзакция типа "income", увеличиваем баланс цели
		newBalance = currentAmount + amount
	} else if transactionType == "goal" {
		// Handle "goal" type if needed (balance can stay same or be updated based on a goal-specific logic)
		// For example, we can log or throw an error if the logic requires further handling here.
		return nil // Skip updating balance for "goal", or handle it differently.
	} else {
		return fmt.Errorf("неизвестный тип транзакции: %s", transactionType)
	}

	// Обновляем баланс цели
	updateQuery := `UPDATE goals SET current_amount = $1 WHERE id = $2`
	_, err = pool.Exec(context.Background(), updateQuery, newBalance, goalID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении баланса цели: %v", err)
	}

	// Проверка достижения цели
	if newBalance >= targetAmount && goalStatus != "completed" {
		// Цель достигнута
		updateStatusQuery := `UPDATE goals SET status = 'completed' WHERE id = $1`
		_, err := pool.Exec(context.Background(), updateStatusQuery, goalID)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении статуса цели на 'completed': %v", err)
		}
	}

	return nil
}

// Получение валюты транзакции по ID пользователя
func GetTransactionCurrencyByUserID(pool *pgxpool.Pool, userID int) (string, error) {
	// Запрос для получения валюты транзакции
	query := "SELECT currency FROM transactions WHERE user_id = $1 LIMIT 1"
	var currency string
	err := pool.QueryRow(context.Background(), query, userID).Scan(&currency)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Если данных нет для пользователя, возвращаем ошибку
			return "", fmt.Errorf("не найдена валюта для пользователя с ID %d", userID)
		}
		// Обработка других ошибок
		return "", fmt.Errorf("ошибка при получении валюты транзакции для пользователя: %v", err)
	}
	return currency, nil
}
