package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func CreateTransaction(conn *pgx.Conn, transaction *models.Transaction) error {
	query := `
		INSERT INTO transactions (user_id, category_id, amount, description, transaction_date) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id`

	err := conn.QueryRow(context.Background(), query,
		transaction.UserID,
		transaction.CategoryID,
		transaction.Amount,
		transaction.Note,
		transaction.Date).Scan(&transaction.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении транзакции: %v", err)
	}
	return nil
}

func GetTransactionByID(conn *pgx.Conn, transactionID int) (*models.Transaction, error) {
	query := `
		SELECT id, user_id, category_id, amount, description, transaction_date 
		FROM transactions 
		WHERE id = $1`

	transaction := &models.Transaction{}
	err := conn.QueryRow(context.Background(), query, transactionID).Scan(
		&transaction.ID,
		&transaction.UserID,
		&transaction.CategoryID,
		&transaction.Amount,
		&transaction.Note,
		&transaction.Date,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("транзакция с ID %d не найдена", transactionID)
		}
		return nil, fmt.Errorf("ошибка при получении транзакции: %v", err)
	}

	return transaction, nil
}

func UpdateTransaction(conn *pgx.Conn, transaction *models.Transaction) error {
	query := `
		UPDATE transactions 
		SET category_id = $1, amount = $2, description = $3, transaction_date = $4 
		WHERE id = $5`

	_, err := conn.Exec(context.Background(), query,
		transaction.CategoryID,
		transaction.Amount,
		transaction.Note,
		transaction.Date,
		transaction.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления транзакции: %v", err)
	}
	return nil
}

func DeleteTransaction(conn *pgx.Conn, transactionID int) error {
	query := `
		DELETE FROM transactions 
		WHERE id = $1`

	result, err := conn.Exec(context.Background(), query, transactionID)
	if err != nil {
		return fmt.Errorf("ошибка удаления транзакции: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("транзакция с ID %d не найдена", transactionID)
	}
	return nil
}
