package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"log"
)

func main() {
	// Подключение к базе данных
	dbURL := "postgres://postgres:root@localhost:5432/finance_db" // Замените на свой URL
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer pool.Close()

	// Запуск миграции паролей
	if err := hashPasswords(pool); err != nil {
		log.Fatalf("Ошибка при миграции паролей: %v", err)
	}

	log.Println("Миграция паролей завершена успешно.")
}

// Функция для хеширования паролей пользователей
func hashPasswords(pool *pgxpool.Pool) error {
	rows, err := pool.Query(context.Background(), "SELECT id, password FROM users")
	if err != nil {
		return fmt.Errorf("Ошибка при получении пользователей: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var password string

		if err := rows.Scan(&id, &password); err != nil {
			return fmt.Errorf("Ошибка при сканировании пользователя: %v", err)
		}

		// Пропускаем пароли, которые уже хешированы
		if len(password) == 60 {
			continue
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("Ошибка хеширования пароля: %v", err)
		}

		_, err = pool.Exec(context.Background(), "UPDATE users SET password = $1 WHERE id = $2", hashedPassword, id)
		if err != nil {
			return fmt.Errorf("Ошибка при обновлении пароля пользователя: %v", err)
		}

		log.Printf("Пароль для пользователя с ID %d успешно обновлен", id)
	}

	return nil
}
