package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func CreateUserSettings(conn *pgx.Conn, settings *models.UserSettings) error {
	query := `
		INSERT INTO usersettings (user_id, two_factor_enabled) 
		VALUES ($1, $2) 
		RETURNING id`

	err := conn.QueryRow(context.Background(), query,
		settings.UserID,
		settings.TwoFactorEnabled).Scan(&settings.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении настроек пользователя: %v", err)
	}
	return nil
}

func GetUserSettingsByID(conn *pgx.Conn, settingsID int) (*models.UserSettings, error) {
	query := `
		SELECT id, user_id, two_factor_enabled 
		FROM usersettings 
		WHERE id = $1`

	settings := &models.UserSettings{}
	err := conn.QueryRow(context.Background(), query, settingsID).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.TwoFactorEnabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("настройки пользователя с ID %d не найдены", settingsID)
		}
		return nil, fmt.Errorf("ошибка при получении настроек пользователя: %v", err)
	}

	return settings, nil
}

func UpdateUserSettings(conn *pgx.Conn, settings *models.UserSettings) error {
	query := `
		UPDATE usersettings 
		SET two_factor_enabled = $1 
		WHERE id = $2`

	_, err := conn.Exec(context.Background(), query,
		settings.TwoFactorEnabled,
		settings.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления настроек пользователя: %v", err)
	}
	return nil
}

func DeleteUserSettings(conn *pgx.Conn, settingsID int) error {
	query := `
		DELETE FROM usersettings 
		WHERE id = $1`

	result, err := conn.Exec(context.Background(), query, settingsID)
	if err != nil {
		return fmt.Errorf("ошибка удаления настроек пользователя: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("настройки пользователя с ID %d не найдены", settingsID)
	}
	return nil
}
