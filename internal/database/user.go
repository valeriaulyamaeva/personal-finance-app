package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"golang.org/x/crypto/bcrypt"
	"log"
)

// RegisterUser регистрирует нового пользователя
func RegisterUser(conn *pgxpool.Pool, user *models.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("ошибка хеширования пароля: %v", err)
	}

	query := `
		INSERT INTO Users (email, password, name) VALUES ($1, $2, $3) RETURNING id`
	err = conn.QueryRow(context.Background(), query, user.Email, hashedPassword, user.Name).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении пользователя: %v", err)
	}
	return nil
}

func AuthenticateUser(conn *pgxpool.Pool, email, password string) (*models.User, error) {
	var user models.User
	query := `SELECT id, email, password, name FROM Users WHERE email = $1`
	err := conn.QueryRow(context.Background(), query, email).Scan(&user.ID, &user.Email, &user.Password, &user.Name)
	if err != nil {
		return nil, fmt.Errorf("пользователь не найден: %v", err)
	}

	// Вывод для отладки
	log.Printf("Введенный пароль: %s", password)
	log.Printf("Хешированный пароль из базы данных: %s", user.Password)

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("неверный пароль")
	}

	return &user, nil
}

func HashAndUpdateUserPasswords(pool *pgxpool.Pool) error {
	// Получаем всех пользователей
	rows, err := pool.Query(context.Background(), "SELECT id, password FROM Users")
	if err != nil {
		return fmt.Errorf("ошибка при получении пользователей: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var password string

		if err := rows.Scan(&id, &password); err != nil {
			return fmt.Errorf("ошибка при сканировании пользователя: %v", err)
		}

		// Хешируем пароль
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("ошибка хеширования пароля: %v", err)
		}

		// Обновляем пользователя с новым хешированным паролем
		_, err = pool.Exec(context.Background(), "UPDATE Users SET password = $1 WHERE id = $2", hashedPassword, id)
		if err != nil {
			return fmt.Errorf("ошибка при обновлении пароля пользователя: %v", err)
		}
	}

	return nil
}
