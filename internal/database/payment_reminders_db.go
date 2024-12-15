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

func CreatePaymentReminder(pool *pgxpool.Pool, reminder *models.PaymentReminder) error {
	// Логируем дату перед вставкой в БД
	log.Printf("Дата перед вставкой в БД: %v", reminder.DueDate)

	// Проверка на валидность даты
	if reminder.DueDate.IsZero() || reminder.DueDate.Before(time.Now().Truncate(24*time.Hour)) {
		return fmt.Errorf("некорректная или прошедшая дата напоминания")
	}

	// Проверка на валидность суммы
	if reminder.Amount <= 0 {
		return fmt.Errorf("некорректная сумма напоминания")
	}

	// Вставляем напоминание в базу данных
	query := `
        INSERT INTO payment_reminders (user_id, description, amount, due_date) 
        VALUES ($1, $2, $3, $4) 
        RETURNING id`
	err := pool.QueryRow(context.Background(), query,
		reminder.UserID,
		reminder.Description,
		reminder.Amount,
		reminder.DueDate).Scan(&reminder.ID)

	if err != nil {
		return fmt.Errorf("ошибка добавления напоминания: %v", err)
	}

	// Запланировать одно уведомление в день события
	err = ScheduleSingleNotification(pool, reminder)
	if err != nil {
		return fmt.Errorf("ошибка при планировании уведомлений: %v", err)
	}

	return nil
}

// Функция планирования одного уведомления
func ScheduleSingleNotification(pool *pgxpool.Pool, reminder *models.PaymentReminder) error {
	// Уведомление планируется в день события (due_date)
	notificationDate := reminder.DueDate
	message := fmt.Sprintf("Напоминание: нужно заплатить %.2f за %s до %v", reminder.Amount, reminder.Description, notificationDate)

	notification := models.Notification{
		UserID:   reminder.UserID,
		Message:  message,
		IsRead:   false,
		DateWhen: notificationDate,
	}

	// Проверка, чтобы уведомление не было на прошедшую дату
	if notification.DateWhen.Before(time.Now()) {
		log.Printf("Дата уведомления для напоминания ID %d уже прошла: %v", reminder.ID, notification.DateWhen)
		return nil // Если дата прошла, уведомление не отправляем
	}

	// Если уведомление еще не прошло, создаем его
	if err := CreateNotification(pool, &notification); err != nil {
		log.Printf("Ошибка при создании уведомления для напоминания ID %d: %v", reminder.ID, err)
		return fmt.Errorf("ошибка при создании уведомления: %w", err)
	}

	return nil
}

func GetPaymentReminderByID(pool *pgxpool.Pool, reminderID int) (*models.PaymentReminder, error) {
	query := `
		SELECT id, user_id, description, amount, due_date 
		FROM payment_reminders 
		WHERE id = $1`

	reminder := &models.PaymentReminder{}
	err := pool.QueryRow(context.Background(), query, reminderID).Scan(
		&reminder.ID,
		&reminder.UserID,
		&reminder.Description,
		&reminder.Amount,
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
	query := `SELECT id, user_id, description, amount, due_date FROM payment_reminders WHERE user_id = $1`
	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения напоминаний: %v", err)
	}
	defer rows.Close()

	var reminders []models.PaymentReminder
	for rows.Next() {
		var reminder models.PaymentReminder
		if err := rows.Scan(&reminder.ID, &reminder.UserID, &reminder.Description, &reminder.Amount, &reminder.DueDate); err != nil {
			return nil, err
		}
		reminders = append(reminders, reminder)
	}
	return reminders, nil
}

func UpdatePaymentReminder(pool *pgxpool.Pool, reminder *models.PaymentReminder) error {
	query := `
		UPDATE payment_reminders 
		SET description = $1, amount = $2, due_date = $3 
		WHERE id = $4`

	_, err := pool.Exec(context.Background(), query,
		reminder.Description,
		reminder.Amount, // Обрабатываем amount
		reminder.DueDate,
		reminder.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления напоминания: %v", err)
	}
	return nil
}

func DeletePaymentReminder(pool *pgxpool.Pool, reminderID int) error {
	log.Printf("Попытка удалить напоминание с ID %d", reminderID)

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

	log.Printf("Напоминание с ID %d успешно удалено", reminderID)
	return nil
}

func GetPaymentRemindersByUserIDAndDate(pool *pgxpool.Pool, userID int, date time.Time) ([]models.PaymentReminder, error) {
	query := `
		SELECT id, user_id, description, amount, due_date 
		FROM payment_reminders 
		WHERE user_id = $1 AND due_date >= $2`
	rows, err := pool.Query(context.Background(), query, userID, date)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения напоминаний: %v", err)
	}
	defer rows.Close()

	var reminders []models.PaymentReminder
	for rows.Next() {
		var reminder models.PaymentReminder
		if err := rows.Scan(&reminder.ID, &reminder.UserID, &reminder.Description, &reminder.Amount, &reminder.DueDate); err != nil {
			return nil, err
		}
		reminders = append(reminders, reminder)
	}
	return reminders, nil
}
