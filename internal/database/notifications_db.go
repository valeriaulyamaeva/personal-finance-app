package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func CreateNotification(conn *pgx.Conn, notification *models.Notification) error {
	query := `
		INSERT INTO notifications (user_id, message, is_read) 
		VALUES ($1, $2, $3) 
		RETURNING id`

	err := conn.QueryRow(context.Background(), query,
		notification.UserID,
		notification.Message,
		notification.IsRead).Scan(&notification.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении уведомления: %v", err)
	}
	return nil
}

func GetNotificationByID(conn *pgx.Conn, notificationID int) (*models.Notification, error) {
	query := `
		SELECT id, user_id, message, is_read, created_at 
		FROM notifications 
		WHERE id = $1`

	notification := &models.Notification{}
	err := conn.QueryRow(context.Background(), query, notificationID).Scan(
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

func UpdateNotification(conn *pgx.Conn, notification *models.Notification) error {
	query := `
		UPDATE notifications 
		SET message = $1, is_read = $2 
		WHERE id = $3`

	_, err := conn.Exec(context.Background(), query,
		notification.Message,
		notification.IsRead,
		notification.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления уведомления: %v", err)
	}
	return nil
}

func DeleteNotification(conn *pgx.Conn, notificationID int) error {
	query := `
		DELETE FROM notifications 
		WHERE id = $1`

	result, err := conn.Exec(context.Background(), query, notificationID)
	if err != nil {
		return fmt.Errorf("ошибка удаления уведомления: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("уведомление с ID %d не найдено", notificationID)
	}
	return nil
}
