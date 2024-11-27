package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"golang.org/x/crypto/bcrypt"
)

func validateUserData(user *models.User) error {
	if user.Name == "" || user.Email == "" || user.Password == "" {
		log.Printf("Ошибка валидации данных: %+v\n", user)
		return errors.New("все поля обязательны для заполнения")
	}

	emailRegex := `^[^\s@]+@[^\s@]+\.[^\s@]+$`
	matched, _ := regexp.MatchString(emailRegex, user.Email)
	if !matched {
		log.Printf("Некорректный email: %s\n", user.Email)
		return errors.New("некорректный формат email")
	}

	return nil
}

func RegisterUser(pool *pgxpool.Pool, user *models.User) error {
	tx, err := pool.Begin(context.Background())
	if err != nil {
		return errors.New("ошибка запуска транзакции")
	}
	defer tx.Rollback(context.Background())

	// Insert user
	query := `
		INSERT INTO users (email, password, name, is_admin)
		VALUES ($1, $2, $3, $4)
		RETURNING id`
	err = tx.QueryRow(context.Background(), query, user.Email, user.Password, user.Name, false).Scan(&user.ID)
	if err != nil {
		return errors.New("ошибка добавления пользователя в базу данных")
	}

	settingsQuery := `
		INSERT INTO usersettings (user_id, two_factor_enabled, theme, notification_volume, auto_updates, weekly_reports)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.Exec(context.Background(), settingsQuery, user.ID, false, "light", 50, false, false)
	if err != nil {
		return errors.New("ошибка создания дэфолтных настроек")
	}

	if err := tx.Commit(context.Background()); err != nil {
		return errors.New("ошибка коммита транзакции")
	}

	return nil
}

func AuthenticateUser(pool *pgxpool.Pool, email, password string) (*models.User, error) {
	var user models.User
	query := `SELECT id, email, password, name FROM users WHERE email = $1`

	err := pool.QueryRow(context.Background(), query, email).Scan(&user.ID, &user.Email, &user.Password, &user.Name)
	if err != nil {
		dummyHash, _ := bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))

		if err.Error() == "no rows in result set" {
			log.Printf("Ошибка аутентификации: пользователь с email %s не найден.\n", email)
			return nil, errors.New("пользователь не найден")
		}
		log.Printf("Ошибка запроса аутентификации: %v\n", err)
		return nil, fmt.Errorf("ошибка при аутентификации: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		log.Printf("Ошибка аутентификации: неверный пароль для email %s.\n", email)
		return nil, errors.New("неверный пароль")
	}

	user.Password = ""
	return &user, nil
}
