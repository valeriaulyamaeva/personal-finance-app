package database_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func TestCreateCategory(t *testing.T) {
	conn, err := database.ConnectDB()
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка подключения к бд: %v", err)
	}
	defer conn.Close(context.Background())

	category := &models.Category{
		UserID: 1,
		Name:   fmt.Sprintf("Test Category %d", time.Now().UnixNano()),
		Type:   "expense",
	}

	err = database.CreateCategory(conn, category)
	if err != nil {
		t.Fatalf("ошибка создания категории: %v", err)
	}

	t.Logf("ID категории после создания: %d", category.ID)

	createdCategory, err := database.GetCategoryByID(conn, category.ID)
	if err != nil {
		t.Fatalf("ошибка получения категории по id: %v", err)
	}

	if createdCategory.Name != category.Name || createdCategory.Type != category.Type {
		t.Errorf("данные категории не совпадают: получили %+v, хотели %+v", createdCategory, category)
	}
}
func TestUpdateCategory(t *testing.T) {
	conn, err := database.ConnectDB()
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	defer conn.Close(context.Background())

	// Создаем новую категорию для обновления
	category := &models.Category{
		UserID: 1,
		Name:   "Category to Update",
		Type:   "expense",
	}
	err = database.CreateCategory(conn, category)
	if err != nil {
		t.Fatalf("ошибка создания категории: %v", err)
	}

	// Обновляем категорию
	category.Name = "Updated Category"
	category.Type = "income"
	err = database.UpdateCategory(conn, category)
	if err != nil {
		t.Fatalf("ошибка обновления категории: %v", err)
	}

	// Получаем обновленную категорию для проверки
	updatedCategory, err := database.GetCategoryByID(conn, category.ID)
	if err != nil {
		t.Fatalf("ошибка получения обновленной категории: %v", err)
	}

	if updatedCategory.Name != category.Name || updatedCategory.Type != category.Type {
		t.Errorf("данные обновленной категории не совпадают: получили %+v, хотели %+v", updatedCategory, category)
	}
}
func TestDeleteCategory(t *testing.T) {
	conn, err := database.ConnectDB()
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	defer conn.Close(context.Background())

	// Создаем новую категорию для удаления
	category := &models.Category{
		UserID: 1,
		Name:   "Category to Delete",
		Type:   "expense",
	}
	err = database.CreateCategory(conn, category)
	if err != nil {
		t.Fatalf("ошибка создания категории: %v", err)
	}

	// Удаляем категорию
	err = database.DeleteCategory(conn, category.ID)
	if err != nil {
		t.Fatalf("ошибка удаления категории: %v", err)
	}

	// Проверяем, что категория была удалена
	_, err = database.GetCategoryByID(conn, category.ID)
	if err == nil {
		t.Errorf("категория с ID %d все еще существует после удаления", category.ID)
	}
}
