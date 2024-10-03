package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func CreateCategory(conn *pgx.Conn, category *models.Category) error {
	query := `
		INSERT INTO categories (user_id, name, type) VALUES ($1, $2, $3) RETURNING id`

	fmt.Printf("Inserting category with UserID: %d, Name: %s, Type: %s\n", category.UserID, category.Name, category.Type)

	err := conn.QueryRow(context.Background(), query, category.UserID, category.Name, category.Type).Scan(&category.ID)
	if err != nil {
		fmt.Printf("Error details: %v\n", err)
		return fmt.Errorf("ошибка при добавлении категории: %v", err)
	}

	fmt.Printf("Inserted category with ID: %d\n", category.ID)
	return nil
}

func GetCategoryByID(conn *pgx.Conn, categoryID int) (*models.Category, error) {

	query := `
		SELECT id, user_id, name, type 
		FROM categories 
		WHERE id = $1`

	category := &models.Category{}

	err := conn.QueryRow(context.Background(), query, categoryID).Scan(
		&category.ID,
		&category.UserID,
		&category.Name,
		&category.Type,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("категория с ID %d не найдена", categoryID)
		}
		return nil, fmt.Errorf("ошибка при получении категории: %v", err)
	}

	return category, nil
}

func UpdateCategory(conn *pgx.Conn, category *models.Category) error {
	query := `
		UPDATE categories 
		SET name = $1, type = $2 
		WHERE id = $3`

	fmt.Printf("Updating category with ID: %d, New Name: %s, New Type: %s\n", category.ID, category.Name, category.Type)

	_, err := conn.Exec(context.Background(), query, category.Name, category.Type, category.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления категории: %v", err)
	}

	fmt.Printf("Updated category with ID: %d\n", category.ID)
	return nil
}

func DeleteCategory(conn *pgx.Conn, categoryID int) error {
	query := `
		DELETE FROM categories 
		WHERE id = $1`

	fmt.Printf("Deleting category with ID: %d\n", categoryID)

	result, err := conn.Exec(context.Background(), query, categoryID)
	if err != nil {
		return fmt.Errorf("ошибка удаления категории: %v", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("категория с ID %d не найдена", categoryID)
	}

	fmt.Printf("Deleted category with ID: %d\n", categoryID)
	return nil
}
