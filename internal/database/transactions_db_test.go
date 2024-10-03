package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func TestCreateTransaction(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	transaction := &models.Transaction{
		UserID:     1,
		CategoryID: 1,
		Amount:     100.00,
		Note:       "Test transaction",
		Date:       time.Now(),
	}

	err = database.CreateTransaction(conn, transaction)
	if err != nil {
		t.Fatalf("ошибка создания транзакции: %v", err)
	}

	t.Logf("ID транзакции после создания: %d", transaction.ID)

	createdTransaction, err := database.GetTransactionByID(conn, transaction.ID)
	if err != nil {
		t.Fatalf("ошибка получения транзакции по ID: %v", err)
	}

	if createdTransaction.Amount != transaction.Amount || createdTransaction.Note != transaction.Note {
		t.Errorf("данные транзакции не совпадают: получили %+v, хотели %+v", createdTransaction, transaction)
	}
}

func TestUpdateTransaction(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	transaction := &models.Transaction{
		UserID:     1,
		CategoryID: 1,
		Amount:     200.00,
		Note:       "Transaction to update",
		Date:       time.Now(),
	}
	err = database.CreateTransaction(conn, transaction)
	if err != nil {
		t.Fatalf("ошибка создания транзакции: %v", err)
	}

	// Обновляем данные транзакции
	transaction.Amount = 250.00
	transaction.Note = "Updated transaction"
	err = database.UpdateTransaction(conn, transaction)
	if err != nil {
		t.Fatalf("ошибка обновления транзакции: %v", err)
	}

	// Проверяем обновление
	updatedTransaction, err := database.GetTransactionByID(conn, transaction.ID)
	if err != nil {
		t.Fatalf("не смогли получить обновленную транзакцию по ID: %v", err)
	}

	if updatedTransaction.Amount != transaction.Amount || updatedTransaction.Note != transaction.Note {
		t.Errorf("данные транзакции не совпадают после обновления: получили %+v, хотели %+v", updatedTransaction, transaction)
	}
}

func TestDeleteTransaction(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	transaction := &models.Transaction{
		UserID:     1,
		CategoryID: 1,
		Amount:     300.00,
		Note:       "Transaction to delete",
		Date:       time.Now(),
	}
	err = database.CreateTransaction(conn, transaction)
	if err != nil {
		t.Fatalf("ошибка создания транзакции: %v", err)
	}

	err = database.DeleteTransaction(conn, transaction.ID)
	if err != nil {
		t.Fatalf("ошибка удаления транзакции: %v", err)
	}

	// Проверяем, что транзакция удалена
	_, err = database.GetTransactionByID(conn, transaction.ID)
	if err == nil {
		t.Errorf("ошибка удаления транзакции по ID, транзакция все еще существует")
	}
}
