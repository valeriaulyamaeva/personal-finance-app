package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func CreateCategory(pool *pgxpool.Pool, category *models.Category) error {
	// Log the input data for debugging purposes
	fmt.Printf("Attempting to create category with data: UserID=%d, Name=%s, Type=%s\n", category.UserID, category.Name, category.Type)

	query := `
        INSERT INTO categories (user_id, name, type) VALUES ($1, $2, $3) RETURNING id`

	err := pool.QueryRow(context.Background(), query, category.UserID, category.Name, category.Type).Scan(&category.ID)
	if err != nil {
		fmt.Printf("Ошибка при создании категории: %v\n", err)
		return fmt.Errorf("ошибка при добавлении категории: %v", err)
	}
	return nil
}

func GetCategoryByID(pool *pgxpool.Pool, categoryID int) (*models.Category, error) {
	query := `
		SELECT id, user_id, name, type 
		FROM categories 
		WHERE id = $1`

	category := &models.Category{}

	err := pool.QueryRow(context.Background(), query, categoryID).Scan(
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

func UpdateCategory(pool *pgxpool.Pool, category *models.Category) error {
	query := `
		UPDATE categories 
		SET name = $1, type = $2 
		WHERE id = $3`

	_, err := pool.Exec(context.Background(), query, category.Name, category.Type, category.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления категории: %v", err)
	}
	return nil
}

func DeleteCategory(pool *pgxpool.Pool, categoryID int) error {
	query := `DELETE FROM categories WHERE id = $1`
	result, err := pool.Exec(context.Background(), query, categoryID)

	if err != nil {
		return fmt.Errorf("ошибка при удалении категории: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("категория с ID %d не найдена", categoryID)
	}

	return nil
}

func GetAllCategories(pool *pgxpool.Pool) ([]models.Category, error) {
	query := `SELECT id, user_id, name, type FROM categories`
	rows, err := pool.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении категорий: %v", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		if err := rows.Scan(&category.ID, &category.UserID, &category.Name, &category.Type); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, nil
}
