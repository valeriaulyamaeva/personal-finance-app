package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"os"
)

func ConnectDB() (*pgx.Conn, error) {
	// Загрузить переменные из .env
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("Error loading .env file")
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), 5432, os.Getenv("DB_NAME"))

	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
