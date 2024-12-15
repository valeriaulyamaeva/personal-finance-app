package database

import (
	_ "context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"github.com/valeriaulyamaeva/personal-finance-app/utils"
	"golang.org/x/net/context"
	"log"
)

// Обновление валюты в бюджетах
func UpdateCurrencyForUser(pool *pgxpool.Pool, userID int, newCurrency string) error {
	// Получаем текущие бюджеты пользователя
	budgets, err := GetBudgetsByUserID(pool, userID)
	if err != nil {
		return fmt.Errorf("ошибка при получении бюджетов пользователя: %v", err)
	}

	// Обрабатываем каждый бюджет
	for _, budget := range budgets {
		// Если валюта отличается от новой, конвертируем
		if budget.Currency != newCurrency {
			convertedAmount, err := utils.ConvertCurrency(budget.Amount, budget.Currency, newCurrency)
			if err != nil {
				log.Printf("Ошибка при конвертации суммы для бюджета с ID %d: %v", budget.ID, err)
				return err
			}

			// Обновляем сумму и валюту в бюджете
			budget.Amount = convertedAmount
			budget.Currency = newCurrency

			// Обновляем бюджет в базе данных
			if err := UpdateBudget(pool, &budget); err != nil {
				return fmt.Errorf("ошибка обновления бюджета с ID %d: %v", budget.ID, err)
			}
		}
	}

	// Аналогично обновляем транзакции и цели
	// Обновление транзакций
	if err := updateTransactionsCurrency(pool, userID, newCurrency); err != nil {
		return fmt.Errorf("ошибка при обновлении транзакций: %v", err)
	}

	// Обновление целей
	if err := updateGoalsCurrency(pool, userID, newCurrency); err != nil {
		return fmt.Errorf("ошибка при обновлении целей: %v", err)
	}

	// Обновляем валюту в настройках пользователя
	if err := updateUserSettingsCurrency(pool, userID, newCurrency); err != nil {
		return fmt.Errorf("ошибка при обновлении валюты в настройках пользователя: %v", err)
	}

	return nil
}

// Обновление валюты в транзакциях
func updateTransactionsCurrency(pool *pgxpool.Pool, userID int, newCurrency string) error {
	transactions, err := GetTransactionsByUserID(pool, userID)
	if err != nil {
		return fmt.Errorf("ошибка при получении транзакций пользователя: %v", err)
	}

	for _, transaction := range transactions {
		if transaction.Currency != newCurrency {
			convertedAmount, err := utils.ConvertCurrency(transaction.Amount, transaction.Currency, newCurrency)
			if err != nil {
				log.Printf("Ошибка при конвертации суммы для транзакции с ID %d: %v", transaction.ID, err)
				return err
			}

			// Обновляем транзакцию в базе данных
			transaction.Amount = convertedAmount
			transaction.Currency = newCurrency
			if err := UpdateTransaction(pool, &transaction); err != nil {
				return fmt.Errorf("ошибка при обновлении транзакции с ID %d: %v", transaction.ID, err)
			}
		}
	}
	return nil
}

// Обновление валюты для целей пользователя
func updateGoalsCurrency(pool *pgxpool.Pool, userID int, newCurrency string) error {
	// Получаем все цели пользователя
	goals, err := GetGoalsByUserID(pool, userID)
	if err != nil {
		log.Printf("Ошибка при получении целей пользователя с ID %d: %v", userID, err)
		return fmt.Errorf("ошибка при получении целей пользователя: %v", err)
	}

	// Обрабатываем каждую цель
	for _, goal := range goals {
		if goal.Currency != newCurrency {
			// Конвертируем сумму цели, если валюта отличается
			convertedAmount, err := utils.ConvertCurrency(goal.Amount, goal.Currency, newCurrency)
			if err != nil {
				log.Printf("Ошибка при конвертации суммы для цели с ID %d: %v", goal.ID, err)
				return err
			}

			// Логируем конвертированную сумму
			log.Printf("Цель с ID %d: сумма %.2f в валюте %s конвертирована в %.2f в валюте %s", goal.ID, goal.Amount, goal.Currency, convertedAmount, newCurrency)

			// Обновляем цель в базе данных
			goal.Amount = convertedAmount
			goal.Currency = newCurrency
			if err := UpdateGoal(pool, &goal); err != nil {
				log.Printf("Ошибка при обновлении цели с ID %d: %v", goal.ID, err)
				return fmt.Errorf("ошибка при обновлении цели с ID %d: %v", goal.ID, err)
			}
		}
	}
	return nil
}

// Функция для получения целей пользователя из базы данных
func GetGoalsByUserID(pool *pgxpool.Pool, userID int) ([]models.Goal, error) {
	// Здесь запрос к базе для получения всех целей пользователя
	var goals []models.Goal
	query := "SELECT id, user_id, amount, currency, target_date FROM goals WHERE user_id = $1"
	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var goal models.Goal
		if err := rows.Scan(&goal.ID, &goal.UserID, &goal.Amount, &goal.Currency, &goal.TargetDate); err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return goals, nil
}

func GetUserSettings(pool *pgxpool.Pool, userID int) (*models.UserSettings, error) {
	var userSettings models.UserSettings
	err := pool.QueryRow(context.Background(), "SELECT currency FROM usersettings WHERE user_id=$1", userID).Scan(&userSettings.Currency)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении настроек пользователя: %v", err)
	}
	return &userSettings, nil
}

// Обновление валюты в настройках пользователя
func updateUserSettingsCurrency(pool *pgxpool.Pool, userID int, newCurrency string) error {
	userSettings, err := GetUserSettings(pool, userID)
	if err != nil {
		return fmt.Errorf("ошибка при получении настроек пользователя: %v", err)
	}

	// Обновляем валюту в настройках пользователя, если она отличается
	if userSettings.Currency != newCurrency {
		userSettings.Currency = newCurrency

		// Обновляем настройки пользователя
		if err := UpdateUserSettings(pool, userSettings); err != nil {
			return fmt.Errorf("ошибка при обновлении настроек пользователя: %v", err)
		}
	}
	return nil
}
