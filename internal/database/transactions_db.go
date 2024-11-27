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
	query := `
		INSERT INTO transactions (user_id, category_id, amount, description, transaction_date, type) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id`

	err := pool.QueryRow(context.Background(), query,
		transaction.UserID,
		transaction.CategoryID,
		transaction.Amount,
		transaction.Description,
		transaction.Date,
		transaction.Type).Scan(&transaction.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении транзакции: %v", err)
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
	query := `
		UPDATE transactions 
		SET category_id = $1, amount = $2, description = $3, transaction_date = $4, type = $5
		WHERE id = $6`

	_, err := pool.Exec(context.Background(), query,
		transaction.CategoryID,
		transaction.Amount,
		transaction.Description,
		transaction.Date,
		transaction.Type,
		transaction.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления транзакции: %v", err)
	}
	return nil
}

func DeleteTransaction(pool *pgxpool.Pool, transactionID int) error {
	query := `DELETE FROM transactions WHERE id = $1`

	result, err := pool.Exec(context.Background(), query, transactionID)
	if err != nil {
		return fmt.Errorf("ошибка удаления транзакции: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("транзакция с ID %d не найдена", transactionID)
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
