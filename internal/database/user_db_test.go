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
	conn, err := database.ConnectDB()
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка подключения к бд: %v", err)
	}
	defer conn.Close(context.Background())

	user := &models.User{
		Name:     "John Doe",
		Email:    fmt.Sprintf("john.doe.%d@example.com", time.Now().UnixNano()),
		Password: "securepassword",
	}

	err = database.CreateUser(conn, user)
	if err != nil {
		t.Fatalf("ошибка создания пользователя: %v", err)
	}

	t.Logf("id пользователя после создания: %d", user.ID) // Логирование ID

	createdUser, err := database.GetUserByID(conn, user.ID)
	if err != nil {
		t.Fatalf("ошибка получения пользователя по d: %v", err)
	}

	if createdUser.Name != user.Name || createdUser.Email != user.Email {
		t.Errorf("данные пользователя не совпадают: получили %+v, хотели %+v", createdUser, user)
	}
}

func TestUpdateUser(t *testing.T) {
	conn, err := database.ConnectDB()
	if err := godotenv.Load(); err != nil {
		t.Fatalf("не удалось подгрузить .env: %v", err)
	}
	defer conn.Close(context.Background())

	user := &models.User{
		Name:     "Vale Zann",
		Email:    "vale.smith@example.com",
		Password: "123",
	}
	err = database.CreateUser(conn, user)
	if err != nil {
		t.Fatalf("ошибка создания пользователя: %v", err)
	}

	user.Name = "vale new"
	user.Email = "valenew.updated@example.com"
	err = database.UpdateUser(conn, user)
	if err != nil {
		t.Fatalf("ошибка обновления пользователя: %v", err)
	}

	updatedUser, err := database.GetUserByID(conn, user.ID)
	if err != nil {
		t.Fatalf("не смогли получить обновленного пользователя по id: %v", err)
	}

	if updatedUser.Name != user.Name || updatedUser.Email != user.Email {
		t.Errorf("данные пользователя не совпадают осле обновления: получили %+v, хотели %+v", updatedUser, user)
	}
}

func TestDeleteUser(t *testing.T) {
	conn, err := database.ConnectDB()
	if err := godotenv.Load(); err != nil {
		t.Fatalf("не удалось подгрузить .env: %v", err)
	}
	defer conn.Close(context.Background())

	user := &models.User{
		Name:     "Emily Davis",
		Email:    "emily.davis@example.com",
		Password: "yetanothersecurepassword",
	}
	err = database.CreateUser(conn, user)
	if err != nil {
		t.Fatalf("ошибка создания пользователя: %v", err)
	}
	err = database.DeleteUser(conn, user.ID)
	if err != nil {
		t.Fatalf("ошибка удаления пользователя: %v", err)
	}
	_, err = database.GetUserByID(conn, user.ID)
	if err == nil {
		t.Errorf("ошибка удаления пользователя по id")
	}
}
