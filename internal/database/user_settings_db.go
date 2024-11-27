package database

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func GetUserSettingsByID(pool *pgxpool.Pool, userID int) (*models.UserSettings, error) {
	log.Printf("Получение настроек для user_id=%d", userID)
	query := `SELECT id, user_id, two_factor_enabled, theme, notification_volume, auto_updates, weekly_reports
              FROM usersettings WHERE user_id = $1`

	var settings models.UserSettings
	err := pool.QueryRow(context.Background(), query, userID).Scan(
		&settings.ID, &settings.UserID, &settings.TwoFactorEnabled, &settings.Theme,
		&settings.NotificationVolume, &settings.AutoUpdates, &settings.WeeklyReports,
	)
	if err != nil {
		log.Printf("Ошибка получения настроек для user_id=%d: %v", userID, err)
		return nil, errors.New("настройки пользователя не найдены")
	}

	log.Printf("Настройки пользователя успешно получены для user_id=%d: %+v", userID, settings)
	return &settings, nil
}

func UpdateUserSettings(pool *pgxpool.Pool, settings *models.UserSettings) error {
	log.Printf("Обновление настроек для user_id=%d", settings.UserID)
	query := `UPDATE usersettings
              SET two_factor_enabled = $1, theme = $2, notification_volume = $3, auto_updates = $4, weekly_reports = $5
              WHERE user_id = $6`

	result, err := pool.Exec(context.Background(), query,
		settings.TwoFactorEnabled, settings.Theme, settings.NotificationVolume,
		settings.AutoUpdates, settings.WeeklyReports, settings.UserID,
	)
	if err != nil {
		log.Printf("Ошибка обновления настроек для user_id=%d: %v", settings.UserID, err)
		return errors.New("ошибка обновления настроек пользователя")
	}

	log.Printf("Настройки успешно обновлены для user_id=%d, изменено строк=%d", settings.UserID, result.RowsAffected())
	return nil
}
