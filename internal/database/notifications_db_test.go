package database_test

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"testing"
)

func TestCreateNotification(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	notification := &models.Notification{
		UserID:  1,
		Message: "Reminder to pay bills",
		IsRead:  false,
	}

	err = database.CreateNotification(conn, notification)
	if err != nil {
		t.Fatalf("ошибка создания уведомления: %v", err)
	}

	t.Logf("ID уведомления после создания: %d", notification.ID)

	createdNotification, err := database.GetNotificationByID(conn, notification.ID)
	if err != nil {
		t.Fatalf("ошибка получения уведомления по ID: %v", err)
	}

	if createdNotification.Message != notification.Message || createdNotification.IsRead != notification.IsRead {
		t.Errorf("данные уведомления не совпадают: получили %+v, хотели %+v", createdNotification, notification)
	}
}

func TestUpdateNotification(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	notification := &models.Notification{
		UserID:  1,
		Message: "Test notification",
		IsRead:  false,
	}
	err = database.CreateNotification(conn, notification)
	if err != nil {
		t.Fatalf("ошибка создания уведомления: %v", err)
	}

	// Обновляем данные уведомления
	notification.Message = "Updated notification"
	notification.IsRead = true
	err = database.UpdateNotification(conn, notification)
	if err != nil {
		t.Fatalf("ошибка обновления уведомления: %v", err)
	}

	// Проверяем обновление
	updatedNotification, err := database.GetNotificationByID(conn, notification.ID)
	if err != nil {
		t.Fatalf("не смогли получить обновленное уведомление по ID: %v", err)
	}

	if updatedNotification.Message != notification.Message || updatedNotification.IsRead != notification.IsRead {
		t.Errorf("данные уведомления не совпадают после обновления: получили %+v, хотели %+v", updatedNotification, notification)
	}
}

func TestDeleteNotification(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	notification := &models.Notification{
		UserID:  1,
		Message: "Notification to delete",
		IsRead:  false,
	}
	err = database.CreateNotification(conn, notification)
	if err != nil {
		t.Fatalf("ошибка создания уведомления: %v", err)
	}

	err = database.DeleteNotification(conn, notification.ID)
	if err != nil {
		t.Fatalf("ошибка удаления уведомления: %v", err)
	}

	// Проверяем, что уведомление удалено
	_, err = database.GetNotificationByID(conn, notification.ID)
	if err == nil {
		t.Errorf("ошибка удаления уведомления по ID, уведомление все еще существует")
	}
}
