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
	// Хеширование пароля
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Ошибка хеширования пароля: %v\n", err)
		return errors.New("ошибка хеширования пароля")
	}
	user.Password = string(hashedPassword) // Сохраняем хешированный пароль

	tx, err := pool.Begin(context.Background())
	if err != nil {
		return errors.New("ошибка запуска транзакции")
	}
	defer tx.Rollback(context.Background())

	// Вставка пользователя в базу данных
	query := `INSERT INTO users (email, password, name, is_admin) VALUES ($1, $2, $3, $4) RETURNING id`
	err = tx.QueryRow(context.Background(), query, user.Email, user.Password, user.Name, false).Scan(&user.ID)
	if err != nil {
		return errors.New("ошибка добавления пользователя в базу данных")
	}

	// Вставка дефолтных настроек пользователя
	settingsQuery := `INSERT INTO usersettings (user_id, theme, notification_volume, auto_updates, weekly_reports)
		VALUES ($1, $2, $3, $4, $5)`
	_, err = tx.Exec(context.Background(), settingsQuery, user.ID, "light", 50, false, false)
	if err != nil {
		return errors.New("ошибка создания дефолтных настроек")
	}

	// Коммит транзакции
	if err := tx.Commit(context.Background()); err != nil {
		return errors.New("ошибка коммита транзакции")
	}

	return nil
}

func AuthenticateUser(pool *pgxpool.Pool, email, password string) (*models.User, error) {
	var user models.User
	query := `SELECT id, email, password, name, is_admin FROM users WHERE email = $1`

	log.Printf("Authenticating user with email: %s\n", email) // Debug email

	err := pool.QueryRow(context.Background(), query, email).Scan(&user.ID, &user.Email, &user.Password, &user.Name, &user.IsAdmin)
	if err != nil {
		if err.Error() == "no rows in result set" {
			log.Printf("Authentication failed: user with email %s not found.\n", email)
			return nil, errors.New("пользователь не найден")
		}
		log.Printf("Authentication query error: %v\n", err)
		return nil, fmt.Errorf("ошибка при аутентификации: %v", err)
	}

	log.Printf("User found: ID=%d, Email=%s, Admin=%v\n", user.ID, user.Email, user.IsAdmin) // Log user details

	// Verify hashed password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		log.Printf("Authentication failed: invalid password for email %s.\n", email)
		return nil, errors.New("неверный пароль")
	}

	user.Password = "" // Clear password for security
	return &user, nil
}
