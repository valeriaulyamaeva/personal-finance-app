package database_test

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"testing"
	"time"
)

func TestCreatePaymentReminder(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	reminder := &models.PaymentReminder{
		UserID:  1,
		Note:    "Pay utility bills",
		DueDate: time.Now().AddDate(0, 0, 7).Truncate(24 * time.Hour), // Обрезаем время до дня
	}

	err = database.CreatePaymentReminder(conn, reminder)
	if err != nil {
		t.Fatalf("ошибка создания напоминания: %v", err)
	}

	t.Logf("ID напоминания после создания: %d", reminder.ID)

	createdReminder, err := database.GetPaymentReminderByID(conn, reminder.ID)
	if err != nil {
		t.Fatalf("ошибка получения напоминания по ID: %v", err)
	}

	// Сравниваем только дату без времени
	if createdReminder.Note != reminder.Note || !createdReminder.DueDate.Truncate(24*time.Hour).Equal(reminder.DueDate) {
		t.Errorf("данные напоминания не совпадают: получили %+v, хотели %+v", createdReminder, reminder)
	}
}

func TestUpdatePaymentReminder(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	reminder := &models.PaymentReminder{
		UserID:  1,
		Note:    "Test reminder",
		DueDate: time.Now().AddDate(0, 0, 7).Truncate(24 * time.Hour), // Обрезаем время до дня
	}
	err = database.CreatePaymentReminder(conn, reminder)
	if err != nil {
		t.Fatalf("ошибка создания напоминания: %v", err)
	}

	// Обновляем данные напоминания
	reminder.Note = "Updated reminder"
	reminder.DueDate = time.Now().AddDate(0, 0, 14).Truncate(24 * time.Hour) // Обрезаем время до дня
	err = database.UpdatePaymentReminder(conn, reminder)
	if err != nil {
		t.Fatalf("ошибка обновления напоминания: %v", err)
	}

	// Проверяем обновление
	updatedReminder, err := database.GetPaymentReminderByID(conn, reminder.ID)
	if err != nil {
		t.Fatalf("не смогли получить обновленное напоминание по ID: %v", err)
	}

	// Сравниваем только дату без времени
	if updatedReminder.Note != reminder.Note || !updatedReminder.DueDate.Truncate(24*time.Hour).Equal(reminder.DueDate) {
		t.Errorf("данные напоминания не совпадают после обновления: получили %+v, хотели %+v", updatedReminder, reminder)
	}
}

func TestDeletePaymentReminder(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	reminder := &models.PaymentReminder{
		UserID:  1,
		Note:    "Reminder to delete",
		DueDate: time.Now().AddDate(0, 0, 7), // Дата окончания через 7 дней
	}
	err = database.CreatePaymentReminder(conn, reminder)
	if err != nil {
		t.Fatalf("ошибка создания напоминания: %v", err)
	}

	err = database.DeletePaymentReminder(conn, reminder.ID)
	if err != nil {
		t.Fatalf("ошибка удаления напоминания: %v", err)
	}

	// Проверяем, что напоминание удалено
	_, err = database.GetPaymentReminderByID(conn, reminder.ID)
	if err == nil {
		t.Errorf("ошибка удаления напоминания по ID, напоминание все еще существует")
	}
}
