package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func CreateUser(conn *pgx.Conn, user *models.User) error {
	query := `
		INSERT INTO users (name, email, password, is_admin) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id`
	user.IsAdmin = false
	err := conn.QueryRow(context.Background(), query, user.Name, user.Email, user.Password, user.IsAdmin).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении пользователя: %v", err)
	}
	return nil
}

func GetUserByID(conn *pgx.Conn, id int) (*models.User, error) { // получение пользователя по айдишнику
	query := `SELECT id, name, email FROM users WHERE id = $1`
	fmt.Printf("Executing query: %s with ID: %d\n", query, id)
	row := conn.QueryRow(context.Background(), query, id)

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

func UpdateUser(conn *pgx.Conn, user *models.User) error { //обновление данных пользователя
	query := `UPDATE users SET name = $1, email = $2, password = $3 WHERE id = $4`
	_, err := conn.Exec(context.Background(), query, user.Name, user.Email, user.Password, user.ID)
	if err != nil {
		return fmt.Errorf("шибка обнавления пользователя: %v", err)
	}
	return nil
}

func DeleteUser(conn *pgx.Conn, id int) error { //удаление пользователя
	query := `DELETE FROM users WHERE id = $1`
	_, err := conn.Exec(context.Background(), query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления пользователя: %v", err)
	}
	return nil
}
