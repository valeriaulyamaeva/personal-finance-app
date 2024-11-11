package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

// Создание транзакции с учетом поля type
func CreateTransaction(pool *pgxpool.Pool, transaction *models.Transaction) error {
	query := `
		INSERT INTO transactions (user_id, category_id, amount, description, transaction_date, type) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id`

	err := pool.QueryRow(context.Background(), query,
		transaction.UserID,
		transaction.CategoryID,
		transaction.Amount,
		transaction.Description, // Исправлено на Description
		transaction.Date,
		transaction.Type).Scan(&transaction.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении транзакции: %v", err)
	}
	return nil
}

// Получение транзакции по ID с учетом type
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
		&transaction.Type, // Добавлено поле type
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

// Получение всех транзакций с учетом type
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
			&transaction.Type, // Добавлено поле type
		); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании транзакции: %v", err)
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// Обновление транзакции с учетом type
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
		transaction.Type, // Добавлено поле type
		transaction.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления транзакции: %v", err)
	}
	return nil
}

// Удаление транзакции
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
