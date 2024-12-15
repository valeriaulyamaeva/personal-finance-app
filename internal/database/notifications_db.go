package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"log"
	_ "time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func CreateNotification(pool *pgxpool.Pool, notification *models.Notification) error {
	query := `
		INSERT INTO notifications (user_id, message, is_read, datewhen) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id`
	err := pool.QueryRow(context.Background(), query,
		notification.UserID,
		notification.Message,
		notification.IsRead,
		notification.DateWhen).Scan(&notification.ID)
	if err != nil {
		return fmt.Errorf("ошибка при создании уведомления: %v", err)
	}
	return nil
}

func GetNotificationByID(pool *pgxpool.Pool, notificationID int) (*models.Notification, error) {
	query := `
		SELECT id, user_id, message, is_read, created_at 
		FROM notifications 
		WHERE id = $1`

	notification := &models.Notification{}
	err := pool.QueryRow(context.Background(), query, notificationID).Scan(
		&notification.ID,
		&notification.UserID,
		&notification.Message,
		&notification.IsRead,
		&notification.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("уведомление с ID %d не найдено", notificationID)
		}
		return nil, fmt.Errorf("ошибка при получении уведомления: %v", err)
	}

	return notification, nil
}

func GetNotificationsByUserID(pool *pgxpool.Pool, userID int) ([]models.Notification, error) {
	query := `
		SELECT id, user_id, message, is_read, created_at 
		FROM notifications 
		WHERE user_id = $1 
		ORDER BY created_at DESC`

	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении уведомлений: %v", err)
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var notification models.Notification
		if err := rows.Scan(&notification.ID, &notification.UserID, &notification.Message, &notification.IsRead, &notification.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}

	return notifications, nil
}

func UpdateNotification(pool *pgxpool.Pool, notification *models.Notification) error {
	query := `
		UPDATE notifications 
		SET message = $1, is_read = $2 
		WHERE id = $3`

	_, err := pool.Exec(context.Background(), query,
		notification.Message,
		notification.IsRead,
		notification.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления уведомления: %v", err)
	}
	return nil
}

func MarkNotificationAsRead(pool *pgxpool.Pool, notificationID int) error {
	query := `
		UPDATE notifications 
		SET is_read = TRUE 
		WHERE id = $1`

	_, err := pool.Exec(context.Background(), query, notificationID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении статуса уведомления: %v", err)
	}
	return nil
}

func DeleteNotification(pool *pgxpool.Pool, notificationID int) error {
	query := `
		DELETE FROM notifications 
		WHERE id = $1`

	result, err := pool.Exec(context.Background(), query, notificationID)
	if err != nil {
		return fmt.Errorf("ошибка удаления уведомления: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("уведомление с ID %d не найдено", notificationID)
	}
	return nil
}

func DeleteNotificationByNotificationID(pool *pgxpool.Pool, notificationID int) error {
	query := `DELETE FROM notifications WHERE id = $1`

	// Выполнение запроса
	result, err := pool.Exec(context.Background(), query, notificationID)
	if err != nil {
		log.Printf("Ошибка выполнения запроса: %v", err)
		return fmt.Errorf("ошибка удаления уведомления: %v", err)
	}

	if result.RowsAffected() == 0 {
		log.Printf("Уведомление с ID %d не найдено", notificationID)
		return fmt.Errorf("уведомление с ID %d не найдено", notificationID)
	}

	return nil
}
