package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"log"
)

func GetUserSettingsByID(pool *pgxpool.Pool, userID int) (*models.UserSettings, error) {
	log.Printf("Получение настроек для user_id=%d", userID)

	query := `SELECT id, user_id, theme, notification_volume, auto_updates, 
                     weekly_reports, currency
              FROM usersettings WHERE user_id = $1`

	var settings models.UserSettings
	err := pool.QueryRow(context.Background(), query, userID).Scan(
		&settings.ID, &settings.UserID, &settings.Theme,
		&settings.NotificationVolume, &settings.AutoUpdates, &settings.WeeklyReports,
		&settings.Currency,
	)
	if err != nil {
		// Detailed logging to help identify the problem
		log.Printf("Ошибка получения настроек для user_id=%d: %v", userID, err)
		// If the error is related to not finding the user, return a specific error message
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("настройки пользователя с ID %d не найдены", userID)
		}
		// Handle other types of errors (e.g., database connectivity issues)
		return nil, fmt.Errorf("не удалось получить настройки пользователя с ID %d: %w", userID, err)
	}

	log.Printf("Настройки пользователя успешно получены для user_id=%d: %+v", userID, settings)
	return &settings, nil
}

func UpdateUserSettings(pool *pgxpool.Pool, settings *models.UserSettings) error {
	log.Printf("Обновление настроек для user_id=%d", settings.UserID)

	// Using named parameters for clarity
	query := `UPDATE usersettings
              SET theme = $1, notification_volume = $2,
                  auto_updates = $3, weekly_reports = $4, currency = $5
              WHERE user_id = $6`

	// Prepare parameters considering string and bool values (instead of sql.Null* types)
	var currency interface{}
	if settings.Currency != "" {
		currency = settings.Currency
	} else {
		currency = nil
	}

	// Execute the update query
	result, err := pool.Exec(context.Background(), query,
		settings.Theme, settings.NotificationVolume,
		settings.AutoUpdates, settings.WeeklyReports, currency, settings.UserID,
	)
	if err != nil {
		// Log error if update fails
		log.Printf("Ошибка обновления настроек для user_id=%d: %v", settings.UserID, err)
		return fmt.Errorf("ошибка обновления настроек пользователя с ID %d: %w", settings.UserID, err)
	}

	// Check if any rows were affected (i.e., if the update was successful)
	if result.RowsAffected() == 0 {
		// Log if no rows were updated
		log.Printf("Не удалось обновить настройки для user_id=%d: пользователь не найден или нет изменений", settings.UserID)
		return fmt.Errorf("не удалось обновить настройки для пользователя с ID %d", settings.UserID)
	}

	log.Printf("Настройки успешно обновлены для user_id=%d, изменено строк: %d", settings.UserID, result.RowsAffected())
	return nil
}
