package database_test

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"testing"
	"time"
)

func TestCreateUser(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	user := &models.User{
		Name:     "Vicky Crend",
		Email:    fmt.Sprintf("vickycred.%d@example.com", time.Now().UnixNano()),
		Password: "987",
		IsAdmin:  false, // Устанавливаем значение по умолчанию для is_admin
	}

	err = database.CreateUser(conn, user)
	if err != nil {
		t.Fatalf("ошибка создания пользователя: %v", err)
	}

	t.Logf("ID пользователя после создания: %d", user.ID)

	createdUser, err := database.GetUserByID(conn, user.ID)
	if err != nil {
		t.Fatalf("ошибка получения пользователя по ID: %v", err)
	}

	if createdUser.Name != user.Name || createdUser.Email != user.Email || createdUser.IsAdmin != user.IsAdmin {
		t.Errorf("данные пользователя не совпадают: получили %+v, хотели %+v", createdUser, user)
	}
}

func TestUpdateUser(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("не удалось подгрузить .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	user := &models.User{
		Name:     "Vale Olesik",
		Email:    "vale.smith1111@example.com",
		Password: "123",
		IsAdmin:  false, // Устанавливаем значение по умолчанию для is_admin
	}
	err = database.CreateUser(conn, user)
	if err != nil {
		t.Fatalf("ошибка создания пользователя: %v", err)
	}

	// Обновляем данные пользователя
	user.Name = "Vale New"
	user.Email = "valenewemail.updated@example.com"
	user.IsAdmin = true // Изменяем статус на администратор
	err = database.UpdateUser(conn, user)
	if err != nil {
		t.Fatalf("ошибка обновления пользователя: %v", err)
	}

	// Проверяем обновление
	updatedUser, err := database.GetUserByID(conn, user.ID)
	if err != nil {
		t.Fatalf("не смогли получить обновленного пользователя по ID: %v", err)
	}

	if updatedUser.Name != user.Name || updatedUser.Email != user.Email || updatedUser.IsAdmin != user.IsAdmin {
		t.Errorf("данные пользователя не совпадают после обновления: получили %+v, хотели %+v", updatedUser, user)
	}
}

func TestDeleteUser(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("не удалось подгрузить .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	user := &models.User{
		Name:     "Emily Davisness",
		Email:    "emily.davi1111s@example.com",
		Password: "yetanothersecurepassword",
		IsAdmin:  false, // Устанавливаем значение по умолчанию для is_admin
	}
	err = database.CreateUser(conn, user)
	if err != nil {
		t.Fatalf("ошибка создания пользователя: %v", err)
	}

	err = database.DeleteUser(conn, user.ID)
	if err != nil {
		t.Fatalf("ошибка удаления пользователя: %v", err)
	}

	// Проверяем, что пользователь удален
	_, err = database.GetUserByID(conn, user.ID)
	if err == nil {
		t.Errorf("ошибка удаления пользователя по ID, пользователь все еще существует")
	}
}
