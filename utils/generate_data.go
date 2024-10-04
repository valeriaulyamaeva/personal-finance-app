package utils

import (
	"context"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/jackc/pgx/v5"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"log"
	"math/rand"
	"time"
)

func GenerateTestUsers(conn *pgx.Conn, numUsers int) {
	for i := 0; i < numUsers; i++ {
		user := &models.User{
			Email:    gofakeit.Email(),
			Password: gofakeit.Password(true, true, true, false, false, 8), // Генерация случайного пароля
			Name:     gofakeit.Name(),
		}
		_, err := conn.Exec(context.Background(), `INSERT INTO Users (email, password, name) VALUES ($1, $2, $3)`, user.Email, user.Password, user.Name)
		if err != nil {
			log.Fatalf("ошибка при добавлении пользователя: %v", err)
		}
	}
}

func GenerateTestCategories(conn *pgx.Conn, numCategories int) {
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < numCategories; i++ {
		category := &models.Category{
			UserID: rand.Intn(10) + 39,
			Name:   gofakeit.Word(),
			Type:   randomCategoryType(),
		}

		// Используем CreateCategory для вставки в базу данных
		err := database.CreateCategory(conn, category)
		if err != nil {
			log.Fatalf("ошибка при добавлении категории: %v", err)
		}
	}
}

func randomCategoryType() string {
	if rand.Intn(2) == 0 {
		return "expense"
	}
	return "income"
}

func GenerateTestTransactions(conn *pgx.Conn, numTransactions int) {
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < numTransactions; i++ {
		transaction := &models.Transaction{
			UserID:     rand.Intn(10) + 39,
			CategoryID: rand.Intn(10) + 11,
			Amount:     gofakeit.Price(0, 1000),                  // Генерация случайной суммы (0.00 до 1000.00)
			Note:       gofakeit.Sentence(5),                     // Генерация случайного описания транзакции
			Date:       time.Now().AddDate(0, 0, -rand.Intn(30)), // Случайная дата в прошлом 30 дней
		}

		err := database.CreateTransaction(conn, transaction)
		if err != nil {
			log.Fatalf("ошибка при добавлении транзакции: %v", err)
		}
	}
}

func GenerateTestBudgets(conn *pgx.Conn, numBudgets int) {
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < numBudgets; i++ {
		startDate := time.Now().AddDate(0, 0, -rand.Intn(30))
		endDate := startDate.AddDate(0, 1, 0)

		budget := &models.Budget{
			UserID:     rand.Intn(10) + 39,
			CategoryID: rand.Intn(10) + 11,
			Amount:     gofakeit.Price(0, 1000),
			Period:     randomBudgetPeriod(),
			StartDate:  startDate,
			EndDate:    endDate,
		}

		err := database.CreateBudget(conn, budget)
		if err != nil {
			log.Fatalf("ошибка при добавлении бюджета: %v", err)
		}
	}
}

func randomBudgetPeriod() string {
	if rand.Intn(2) == 0 {
		return "monthly"
	}
	return "yearly"
}

func GenerateTestPaymentReminders(conn *pgx.Conn, numReminders int) {
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < numReminders; i++ {
		reminder := &models.PaymentReminder{
			UserID:      rand.Intn(10) + 39,
			Description: gofakeit.Sentence(5),                    // Генерация случайного описания
			DueDate:     time.Now().AddDate(0, 0, rand.Intn(30)), // Случайная дата в будущем (до 30 дней)
		}

		err := database.CreatePaymentReminder(conn, reminder)
		if err != nil {
			log.Fatalf("ошибка при добавлении напоминания: %v", err)
		}
	}
}
