package database_test

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"testing"
	"time"
)

func TestCreateBudget(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	budget := &models.Budget{
		UserID:     1,
		CategoryID: 1,
		Amount:     500.00,
		Period:     "monthly",
		StartDate:  time.Now(),
		EndDate:    time.Now().AddDate(0, 1, 0),
	}

	err = database.CreateBudget(conn, budget)
	if err != nil {
		t.Fatalf("ошибка создания бюджета: %v", err)
	}

	t.Logf("ID бюджета после создания: %d", budget.ID)

	createdBudget, err := database.GetBudgetByID(conn, budget.ID)
	if err != nil {
		t.Fatalf("ошибка получения бюджета по ID: %v", err)
	}

	if createdBudget.Amount != budget.Amount || createdBudget.Period != budget.Period {
		t.Errorf("данные бюджета не совпадают: получили %+v, хотели %+v", createdBudget, budget)
	}
}

func TestUpdateBudget(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	budget := &models.Budget{
		UserID:     1,
		CategoryID: 1,
		Amount:     600.00,
		Period:     "monthly",
		StartDate:  time.Now(),
		EndDate:    time.Now().AddDate(0, 1, 0),
	}
	err = database.CreateBudget(conn, budget)
	if err != nil {
		t.Fatalf("ошибка создания бюджета: %v", err)
	}

	// Обновляем данные бюджета
	budget.Amount = 700.00
	budget.Period = "yearly"
	err = database.UpdateBudget(conn, budget)
	if err != nil {
		t.Fatalf("ошибка обновления бюджета: %v", err)
	}

	// Проверяем обновление
	updatedBudget, err := database.GetBudgetByID(conn, budget.ID)
	if err != nil {
		t.Fatalf("не смогли получить обновленный бюджет по ID: %v", err)
	}

	if updatedBudget.Amount != budget.Amount || updatedBudget.Period != budget.Period {
		t.Errorf("данные бюджета не совпадают после обновления: получили %+v, хотели %+v", updatedBudget, budget)
	}
}

func TestDeleteBudget(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	budget := &models.Budget{
		UserID:     1,
		CategoryID: 1,
		Amount:     800.00,
		Period:     "monthly",
		StartDate:  time.Now(),
		EndDate:    time.Now().AddDate(0, 1, 0), // Дата окончания через месяц
	}
	err = database.CreateBudget(conn, budget)
	if err != nil {
		t.Fatalf("ошибка создания бюджета: %v", err)
	}

	err = database.DeleteBudget(conn, budget.ID)
	if err != nil {
		t.Fatalf("ошибка удаления бюджета: %v", err)
	}

	// Проверяем, что бюджет удален
	_, err = database.GetBudgetByID(conn, budget.ID)
	if err == nil {
		t.Errorf("ошибка удаления бюджета по ID, бюджет все еще существует")
	}
}
