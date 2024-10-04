package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	_ "github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/utils"
	"log"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("ошибка загрузки .env файла: %v", err)
	}

	conn, err := pgx.Connect(context.Background(), "postgres://postgres:root@localhost:5432/finance_db")

	if err != nil {
		log.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	utils.GenerateTestTransactions(conn, 10)
	utils.GenerateTestBudgets(conn, 10)
	utils.GenerateTestUsers(conn, 10)
	utils.GenerateTestCategories(conn, 10)
	utils.GenerateTestPaymentReminders(conn, 5)
}
