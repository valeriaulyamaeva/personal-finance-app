package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func CreatePaymentReminder(conn *pgx.Conn, reminder *models.PaymentReminder) error {
	query := `
		INSERT INTO payment_reminders (user_id, description, due_date) 
		VALUES ($1, $2, $3) 
		RETURNING id`

	err := conn.QueryRow(context.Background(), query,
		reminder.UserID,
		reminder.Description,
		reminder.DueDate).Scan(&reminder.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении напоминания: %v", err)
	}
	return nil
}

func GetPaymentReminderByID(conn *pgx.Conn, reminderID int) (*models.PaymentReminder, error) {
	query := `
		SELECT id, user_id, description, due_date 
		FROM payment_reminders 
		WHERE id = $1`

	reminder := &models.PaymentReminder{}
	err := conn.QueryRow(context.Background(), query, reminderID).Scan(
		&reminder.ID,
		&reminder.UserID,
		&reminder.Note,
		&reminder.DueDate,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("напоминание с ID %d не найдено", reminderID)
		}
		return nil, fmt.Errorf("ошибка при получении напоминания: %v", err)
	}

	return reminder, nil
}
func UpdatePaymentReminder(conn *pgx.Conn, reminder *models.PaymentReminder) error {
	query := `
		UPDATE payment_reminders 
		SET description = $1, due_date = $2 
		WHERE id = $3`

	_, err := conn.Exec(context.Background(), query,
		reminder.Note,
		reminder.DueDate,
		reminder.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления напоминания: %v", err)
	}
	return nil
}
func DeletePaymentReminder(conn *pgx.Conn, reminderID int) error {
	query := `
		DELETE FROM payment_reminders 
		WHERE id = $1`

	result, err := conn.Exec(context.Background(), query, reminderID)
	if err != nil {
		return fmt.Errorf("ошибка удаления напоминания: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("напоминание с ID %d не найдено", reminderID)
	}
	return nil
}
