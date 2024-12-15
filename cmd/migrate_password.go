package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
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
	if err := MigrateUsersToNotifications(pool); err != nil {
		log.Fatalf("Ошибка при миграции паролей: %v", err)
	}

	log.Println("Миграция паролей завершена успешно.")
}

// Функция для хеширования паролей пользователей
func MigrateUsersToNotifications(pool *pgxpool.Pool) error {
	query := `
		SELECT id 
		FROM users`
	rows, err := pool.Query(context.Background(), query)
	if err != nil {
		return fmt.Errorf("ошибка при извлечении пользователей: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			return fmt.Errorf("ошибка при чтении пользователя: %v", err)
		}

		notificationQuery := `
			INSERT INTO notifications (user_id, message, is_read) 
			VALUES ($1, $2, $3)`
		_, err = pool.Exec(context.Background(), notificationQuery, userID, "Добро пожаловать в приложение!", false)
		if err != nil {
			return fmt.Errorf("ошибка при создании уведомления для пользователя %d: %v", userID, err)
		}
	}

	return nil
}
