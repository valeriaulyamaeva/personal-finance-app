package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"log"
)

func CreatePaymentReminder(pool *pgxpool.Pool, reminder *models.PaymentReminder) error {
	query := `
		INSERT INTO payment_reminders (user_id, description, due_date) 
		VALUES ($1, $2, $3) 
		RETURNING id`

	err := pool.QueryRow(context.Background(), query,
		reminder.UserID,
		reminder.Description,
		reminder.DueDate).Scan(&reminder.ID)
	if err != nil {
		return fmt.Errorf("ошибка добавления напоминания: %v", err)
	}
	return nil
}

func GetPaymentReminderByID(pool *pgxpool.Pool, reminderID int) (*models.PaymentReminder, error) {
	query := `
		SELECT id, user_id, description, due_date 
		FROM payment_reminders 
		WHERE id = $1`

	reminder := &models.PaymentReminder{}
	err := pool.QueryRow(context.Background(), query, reminderID).Scan(
		&reminder.ID,
		&reminder.UserID,
		&reminder.Description,
		&reminder.DueDate,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("напоминание с ID %d не найдено", reminderID)
		}
		return nil, fmt.Errorf("ошибка получения напоминания: %v", err)
	}

	return reminder, nil
}

func GetPaymentRemindersByUserID(pool *pgxpool.Pool, userID int) ([]models.PaymentReminder, error) {
	query := `SELECT id, user_id, description, due_date FROM payment_reminders WHERE user_id = $1`
	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения напоминаний: %v", err)
	}
	defer rows.Close()

	var reminders []models.PaymentReminder
	for rows.Next() {
		var reminder models.PaymentReminder
		if err := rows.Scan(&reminder.ID, &reminder.UserID, &reminder.Description, &reminder.DueDate); err != nil {
			return nil, err
		}
		reminders = append(reminders, reminder)
	}
	return reminders, nil
}

func UpdatePaymentReminder(pool *pgxpool.Pool, reminder *models.PaymentReminder) error {
	query := `
		UPDATE payment_reminders 
		SET description = $1, due_date = $2 
		WHERE id = $3`

	_, err := pool.Exec(context.Background(), query,
		reminder.Description,
		reminder.DueDate,
		reminder.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления напоминания: %v", err)
	}
	return nil
}

func DeletePaymentReminder(pool *pgxpool.Pool, reminderID int) error {
	query := `
		DELETE FROM payment_reminders 
		WHERE id = $1`

	result, err := pool.Exec(context.Background(), query, reminderID)
	if err != nil {
		return fmt.Errorf("ошибка удаления напоминания: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("напоминание с ID %d не найдено", reminderID)
	}
	return nil
}

func ScheduleReminderNotifications(pool *pgxpool.Pool) {
	c := cron.New()
	c.AddFunc("@daily", func() {
		log.Println("Запуск проверки просроченных напоминаний")
		ctx := context.Background()
		query := `
			SELECT id, user_id, description, amount, due_date 
			FROM payment_reminders 
			WHERE due_date < CURRENT_DATE`
		rows, err := pool.Query(ctx, query)
		if err != nil {
			log.Printf("Ошибка при запросе просроченных напоминаний: %v", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var reminder models.PaymentReminder
			if err := rows.Scan(&reminder.ID, &reminder.UserID, &reminder.Description, &reminder.Amount, &reminder.DueDate); err != nil {
				log.Printf("Ошибка при чтении напоминания: %v", err)
				continue
			}

			notification := models.Notification{
				UserID:  reminder.UserID,
				Message: fmt.Sprintf("Просроченное напоминание о платеже: %s (%.2f руб.)", reminder.Description, reminder.Amount),
				IsRead:  false,
			}
			if err := CreateNotification(pool, &notification); err != nil {
				log.Printf("Ошибка при создании уведомления для напоминания ID %d: %v", reminder.ID, err)
			}
		}
	})
	c.Start()
}
