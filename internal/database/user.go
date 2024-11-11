package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"golang.org/x/crypto/bcrypt"
	"log"
)

func RegisterUser(pool *pgxpool.Pool, user *models.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("ошибка хеширования пароля: %v", err)
	}

	query := `
		INSERT INTO users (email, password, name) VALUES ($1, $2, $3) RETURNING id`
	err = pool.QueryRow(context.Background(), query, user.Email, hashedPassword, user.Name).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении пользователя: %v", err)
	}
	return nil
}

func AuthenticateUser(pool *pgxpool.Pool, email, password string) (*models.User, error) {
	var user models.User
	query := `SELECT id, email, password, name FROM users WHERE email = $1`
	err := pool.QueryRow(context.Background(), query, email).Scan(&user.ID, &user.Email, &user.Password, &user.Name)
	if err != nil {
		return nil, fmt.Errorf("пользователь не найден: %v", err)
	}

	log.Printf("Хешированный пароль из базы данных: %s", user.Password)
	log.Printf("Пароль, введенный пользователем: %s", password)

	// Проверка пароля
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		log.Printf("Пароль неверен: %v", err)
		return nil, fmt.Errorf("неверный пароль")
	}

	user.Password = ""
	return &user, nil
}
