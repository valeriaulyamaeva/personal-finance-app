package database_test

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"testing"
)

func TestCreateUserSettings(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	settings := &models.UserSettings{
		UserID:           1,
		TwoFactorEnabled: false,
	}

	err = database.CreateUserSettings(conn, settings)
	if err != nil {
		t.Fatalf("ошибка создания настроек пользователя: %v", err)
	}

	t.Logf("ID настроек пользователя после создания: %d", settings.ID)

	createdSettings, err := database.GetUserSettingsByID(conn, settings.ID)
	if err != nil {
		t.Fatalf("ошибка получения настроек пользователя по ID: %v", err)
	}

	if createdSettings.UserID != settings.UserID || createdSettings.TwoFactorEnabled != settings.TwoFactorEnabled {
		t.Errorf("данные настроек пользователя не совпадают: получили %+v, хотели %+v", createdSettings, settings)
	}
}

func TestUpdateUserSettings(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	settings := &models.UserSettings{
		UserID:           1,
		TwoFactorEnabled: false,
	}
	err = database.CreateUserSettings(conn, settings)
	if err != nil {
		t.Fatalf("ошибка создания настроек пользователя: %v", err)
	}

	// Обновляем данные настроек пользователя
	settings.TwoFactorEnabled = true
	err = database.UpdateUserSettings(conn, settings)
	if err != nil {
		t.Fatalf("ошибка обновления настроек пользователя: %v", err)
	}

	// Проверяем обновление
	updatedSettings, err := database.GetUserSettingsByID(conn, settings.ID)
	if err != nil {
		t.Fatalf("не смогли получить обновленные настройки пользователя по ID: %v", err)
	}

	if updatedSettings.TwoFactorEnabled != settings.TwoFactorEnabled {
		t.Errorf("данные настроек пользователя не совпадают после обновления: получили %+v, хотели %+v", updatedSettings, settings)
	}
}

func TestDeleteUserSettings(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	settings := &models.UserSettings{
		UserID:           1,
		TwoFactorEnabled: false,
	}
	err = database.CreateUserSettings(conn, settings)
	if err != nil {
		t.Fatalf("ошибка создания настроек пользователя: %v", err)
	}

	err = database.DeleteUserSettings(conn, settings.ID)
	if err != nil {
		t.Fatalf("ошибка удаления настроек пользователя: %v", err)
	}

	// Проверяем, что настройки пользователя удалены
	_, err = database.GetUserSettingsByID(conn, settings.ID)
	if err == nil {
		t.Errorf("ошибка удаления настроек пользователя по ID, настройки все еще существуют")
	}
}
