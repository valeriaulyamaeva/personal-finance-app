package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

// CreateUser создает нового пользователя в базе данных
func CreateUser(pool *pgxpool.Pool, user *models.User) error {
	query := `
		INSERT INTO users (name, email, password, is_admin) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id`
	user.IsAdmin = false
	err := pool.QueryRow(context.Background(), query, user.Name, user.Email, user.Password, user.IsAdmin).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении пользователя: %v", err)
	}

	settingsQuery := `
		INSERT INTO usersettings (user_id, theme, notification_volume, auto_updates, weekly_reports)
		VALUES ($1, $2, $3, $4, $5)`
	_, err = pool.Exec(context.Background(), settingsQuery, user.ID, "light", 50, false, false)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении настроек пользователя: %v", err)
	}

	// Создаем уведомление для нового пользователя
	notificationQuery := `
		INSERT INTO notifications (user_id, message, is_read) 
		VALUES ($1, $2, $3)`
	_, err = pool.Exec(context.Background(), notificationQuery, user.ID, "Добро пожаловать в приложение!", false)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении уведомления: %v", err)
	}

	return nil
}

// GetUserByID получает пользователя по ID
func GetUserByID(pool *pgxpool.Pool, id int) (*models.User, error) {
	query := `SELECT id, name, email FROM users WHERE id = $1`
	fmt.Printf("Executing query: %s with ID: %d\n", query, id)
	row := pool.QueryRow(context.Background(), query, id)

	var user models.User
	err := row.Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("пользователь не найден")
		}
		return nil, fmt.Errorf("ошибка получения пользователя по id: %v", err)
	}

	return &user, nil
}

// UpdateUser обновляет данные пользователя
func UpdateUser(pool *pgxpool.Pool, user *models.User) error {
	query := `UPDATE users SET name = $1, email = $2, password = $3 WHERE id = $4`
	_, err := pool.Exec(context.Background(), query, user.Name, user.Email, user.Password, user.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления пользователя: %v", err)
	}
	return nil
}

func DeleteUser(pool *pgxpool.Pool, id int) error {
	queryCheck := `SELECT COUNT(*) FROM users WHERE id = $1`
	var count int
	err := pool.QueryRow(context.Background(), queryCheck, id).Scan(&count)
	if err != nil {
		return fmt.Errorf("ошибка проверки пользователя: %v", err)
	}
	if count == 0 {
		return fmt.Errorf("пользователь с ID %d не найден", id)
	}

	query := `DELETE FROM users WHERE id = $1`
	_, err = pool.Exec(context.Background(), query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления пользователя: %v", err)
	}
	return nil
}

// GetAllUsers получает список всех пользователей из базы данных
func GetAllUsers(pool *pgxpool.Pool) ([]models.User, error) {
	query := `SELECT id, name, email, is_admin FROM users`
	rows, err := pool.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка пользователей: %v", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.IsAdmin); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании пользователя: %v", err)
		}
		users = append(users, user)
	}

	return users, nil
}
