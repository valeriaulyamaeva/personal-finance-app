package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"github.com/shopspring/decimal"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"github.com/valeriaulyamaeva/personal-finance-app/utils"
	"log"
	"net/http"
	"strconv"
	"sync"
)

func ScheduleBudgetRenewal(pool *pgxpool.Pool) {
	c := cron.New()
	c.AddFunc("@monthly", func() {
		if err := database.UpdateExpiredBudgets(pool); err != nil {
			log.Printf("Ошибка обновления просроченных бюджетов: %v", err)
		}
	})
	c.Start()
}

func ScheduleTransactionArchival(pool *pgxpool.Pool) {
	c := cron.New()
	_, err := c.AddFunc("@daily", func() {
		err := database.MoveTransactionsToHistory(pool)
		if err != nil {
			log.Printf("Ошибка при переносе транзакций в архив: %v", err)
		} else {
			log.Println("Архивирование транзакций завершено успешно.")
		}
	})
	if err != nil {
		log.Fatalf("Ошибка настройки CRON-задачи для архивирования транзакций: %v", err)
	}
	c.Start()
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем исходный домен из заголовка
		origin := c.Request.Header.Get("Origin")

		// Должны быть проверены только разрешенные домены
		allowedOrigins := []string{"http://localhost:3000", "http://localhost:3001"} // можно добавить новые домены

		// Если Origin совпадает с разрешенным, устанавливаем Access-Control-Allow-Origin
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		// Разрешаем отправку cookies и аутентификацию
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

		// Браузеры посылают preflight запросы типа OPTIONS, и нужно на них ответить
		if c.Request.Method == "OPTIONS" {
			c.Writer.Header().Set("Access-Control-Max-Age", "86400") // Кэширование CORS на 1 день
			c.AbortWithStatus(http.StatusOK)
			return
		}

		// Продолжаем обработку запроса
		c.Next()
	}
}

func convertAllDataToNewCurrency(pool *pgxpool.Pool, userID int, oldCurrency string, newCurrency string) error {
	// Получаем курсы валют для конвертации
	fromRate, errFromRate := utils.GetCurrencyRate(oldCurrency)
	toRate, errToRate := utils.GetCurrencyRate(newCurrency)

	log.Printf("Получаем курсы для валют: %s -> %s, fromRate: %v, toRate: %v", oldCurrency, newCurrency, fromRate, toRate)

	if errFromRate != nil || errToRate != nil {
		log.Printf("Ошибка при получении курсов валют: %v, %v", errFromRate, errToRate)
		return fmt.Errorf("ошибка при получении курсов валют: %v, %v", errFromRate, errToRate)
	}

	conversionRate := toRate / fromRate
	log.Printf("Конвертационный курс: %.4f", conversionRate)

	// Конвертируем бюджеты
	budgets, err := database.GetBudgetsByUserID(pool, userID)
	if err != nil {
		log.Printf("Ошибка при получении бюджета для пользователя с ID %d: %v", userID, err)
		return fmt.Errorf("ошибка при получении бюджета для пользователя с ID %d: %v", userID, err)
	}

	for _, budget := range budgets {
		if budget.Currency != newCurrency {
			convertedAmount := budget.Amount * conversionRate
			log.Printf("Конвертация бюджета с ID %d: %.2f -> %.2f %s", budget.ID, budget.Amount, convertedAmount, newCurrency)
			budget.Amount = convertedAmount
			budget.Currency = newCurrency
			if err := database.UpdateBudget(pool, &budget); err != nil {
				log.Printf("Ошибка при обновлении бюджета с ID %d: %v", budget.ID, err)
				return fmt.Errorf("ошибка при обновлении бюджета с ID %d: %v", budget.ID, err)
			}
		}
	}

	// Конвертируем транзакции
	transactions, err := database.GetTransactionsByUserID(pool, userID)
	if err != nil {
		log.Printf("Ошибка при получении транзакций для пользователя с ID %d: %v", userID, err)
		return fmt.Errorf("ошибка при получении транзакций для пользователя с ID %d: %v", userID, err)
	}

	for _, transaction := range transactions {
		if transaction.Currency != newCurrency {
			convertedAmount := transaction.Amount * conversionRate
			log.Printf("Конвертация транзакции с ID %d: %.2f -> %.2f %s", transaction.ID, transaction.Amount, convertedAmount, newCurrency)
			transaction.Amount = convertedAmount
			transaction.Currency = newCurrency
			if err := database.UpdateTransaction(pool, &transaction); err != nil {
				log.Printf("Ошибка при обновлении транзакции с ID %d: %v", transaction.ID, err)
				return fmt.Errorf("ошибка при обновлении транзакции с ID %d: %v", transaction.ID, err)
			}
		}
	}

	// Конвертируем цель
	goal, err := database.GetGoalByID(pool, userID)
	if err != nil {
		log.Printf("Ошибка при получении цели для пользователя с ID %d: %v", userID, err)
		return fmt.Errorf("ошибка при получении цели для пользователя с ID %d: %v", userID, err)
	}

	if goal.Currency != newCurrency {
		convertedAmount := goal.Amount * conversionRate
		log.Printf("Конвертация цели с ID %d: %.2f -> %.2f %s", goal.ID, goal.Amount, convertedAmount, newCurrency)
		goal.Amount = convertedAmount
		goal.Currency = newCurrency
		if err := database.UpdateGoal(pool, goal); err != nil {
			log.Printf("Ошибка при обновлении цели с ID %d: %v", goal.ID, err)
			return fmt.Errorf("ошибка при обновлении цели с ID %d: %v", goal.ID, err)
		}
	}

	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), "postgres://postgres:root@localhost:5432/finance_db")
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer pool.Close()

	r := gin.Default()
	r.Use(CORSMiddleware())

	c := cron.New()
	_, err = c.AddFunc("0 * * * *", func() {
		err := database.MoveTransactionsToHistory(pool)
		if err != nil {
			log.Printf("Ошибка переноса транзакций в архив: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Ошибка добавления задачи в cron: %v", err)
	}
	c.Start()

	ScheduleBudgetRenewal(pool)
	ScheduleTransactionArchival(pool)

	r.POST("/register", func(c *gin.Context) {
		var user models.User

		if err := c.ShouldBindJSON(&user); err != nil {
			log.Printf("Ошибка привязки JSON: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный формат данных. Проверьте введённые значения."})
			return
		}

		log.Printf("Полученные данные для регистрации: %+v\n", user)

		if err := database.RegisterUser(pool, &user); err != nil {
			log.Printf("Ошибка при регистрации пользователя: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка регистрации: %v", err)})
			return
		}

		log.Printf("Пользователь успешно зарегистрирован: ID = %d\n", user.ID)
		c.JSON(http.StatusOK, gin.H{"message": "Пользователь успешно зарегистрирован", "user_id": user.ID})
	})

	r.POST("/login", func(c *gin.Context) {
		var credentials models.User
		if err := c.ShouldBindJSON(&credentials); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка ввода данных"})
			return
		}
		user, err := database.AuthenticateUser(pool, credentials.Email, credentials.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации: неверный email или пароль"})
			return
		}
		user.Password = ""
		c.JSON(http.StatusOK, gin.H{"message": "Авторизация успешна", "user": user})
	})

	r.POST("/categories", func(c *gin.Context) {
		var category models.Category
		if err := c.ShouldBindJSON(&category); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный формат категории"})
			return
		}
		if err := database.CreateCategory(pool, &category); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании категории"})
			return
		}
		c.JSON(http.StatusCreated, category)
	})

	r.GET("/categories", func(c *gin.Context) {
		categories, err := database.GetAllCategories(pool)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении списка категорий"})
			return
		}
		c.JSON(http.StatusOK, categories)
	})

	r.PUT("/categories/:id", func(c *gin.Context) {
		var category models.Category
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор категории"})
			return
		}
		if err := c.ShouldBindJSON(&category); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный формат данных для категории"})
			return
		}
		category.ID = id
		if err := database.UpdateCategory(pool, &category); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления категории"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Категория успешно обновлена"})
	})

	r.DELETE("/categories/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор категории"})
			return
		}
		if err := database.DeleteCategory(pool, id); err != nil {
			log.Printf("Ошибка удаления категории с ID %d: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении категории", "details": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Категория успешно удалена"})
	})

	r.POST("/budgets", func(c *gin.Context) {
		var budget models.Budget
		if err := c.ShouldBindJSON(&budget); err != nil {
			log.Printf("Ошибка привязки JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ввод данных"})
			return
		}
		log.Printf("Полученные данные для создания бюджета: %+v", budget)

		if err := database.CreateBudget(pool, &budget); err != nil {
			log.Printf("Ошибка при создании бюджета: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании бюджета"})
			return
		}
		c.JSON(http.StatusCreated, budget)
	})

	r.GET("/budgets", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Query("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор пользователя"})
			return
		}
		budgets, err := database.GetBudgetsByUserID(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении списка бюджетов"})
			return
		}
		c.JSON(http.StatusOK, budgets)
	})

	r.PUT("/budgets/:id", func(c *gin.Context) {
		var budget models.Budget
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			log.Printf("Некорректный идентификатор бюджета: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор бюджета"})
			return
		}

		if err := c.ShouldBindJSON(&budget); err != nil {
			log.Printf("Ошибка привязки JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ввод данных"})
			return
		}

		budget.ID = id
		log.Printf("Обновляем бюджет с данными: %+v", budget)

		if err := database.UpdateBudget(pool, &budget); err != nil {
			log.Printf("Ошибка обновления бюджета: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении бюджета"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Бюджет успешно обновлён"})
	})

	r.DELETE("/budgets/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор бюджета"})
			return
		}
		if err := database.DeleteBudget(pool, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении бюджета"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Бюджет успешно удалён"})
	})

	r.POST("/transactions", func(c *gin.Context) {
		var transaction models.Transaction
		log.Printf("Необработанные данные транзакции: %v", c.Request.Body)

		// Привязка JSON данных к структуре
		if err := c.ShouldBindJSON(&transaction); err != nil {
			log.Printf("Ошибка привязки JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ввод", "details": err.Error()})
			return
		}

		log.Printf("Полученные данные для создания транзакции: %+v", transaction)

		// Проверка типа транзакции и наличие GoalID
		if transaction.Type == "goal" && transaction.GoalID != nil && *transaction.GoalID != 0 {
			// Преобразование float64 в decimal.Decimal
			amountDecimal := decimal.NewFromFloat(transaction.Amount)

			// Обновление прогресса цели
			if err := database.UpdateGoalProgress(pool, *transaction.GoalID, amountDecimal); err != nil {
				log.Printf("Ошибка при обновлении прогресса для цели %d: %v", *transaction.GoalID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении прогресса цели"})
				return
			}
		}

		// Создание транзакции в базе данных
		if err := database.CreateTransaction(pool, &transaction); err != nil {
			log.Printf("Ошибка при создании транзакции: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания транзакции"})
			return
		}

		c.JSON(http.StatusCreated, transaction)
	})

	r.GET("/transactions", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Query("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор пользователя"})
			return
		}
		transactions, err := database.GetTransactionsByUserID(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения транзакций"})
			return
		}
		c.JSON(http.StatusOK, transactions)
	})

	r.PUT("/transactions/:id", func(c *gin.Context) {
		var transaction models.Transaction
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор транзакции"})
			return
		}
		if err := c.ShouldBindJSON(&transaction); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ввод"})
			return
		}
		transaction.ID = id

		if err := database.UpdateTransaction(pool, &transaction); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления транзакции"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Транзакция успешно обновлена"})
	})

	r.DELETE("/transactions/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор транзакции"})
			return
		}
		if err := database.DeleteTransaction(pool, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления транзакции"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Транзакция успешно удалена"})
	})

	r.GET("/dashboard/total_balance", func(c *gin.Context) {
		userID, _ := strconv.Atoi(c.Query("user_id"))
		balance, err := database.GetTotalBalance(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"total_balance": balance})
	})

	r.GET("/dashboard/monthly_expenses", func(c *gin.Context) {
		userID, _ := strconv.Atoi(c.Query("user_id"))
		expenses, err := database.GetMonthlyExpenses(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"expenses": expenses})
	})

	r.GET("/dashboard/income_expense_summary", func(c *gin.Context) {
		userID, _ := strconv.Atoi(c.Query("user_id"))
		summary, err := database.GetIncomeExpenseSummary(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, summary)
	})

	r.GET("/dashboard/category_expenses", func(c *gin.Context) {
		userID, _ := strconv.Atoi(c.Query("user_id"))
		categoryExpenses, err := database.GetCategoryWiseExpenses(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"category_expenses": categoryExpenses})
	})

	r.POST("/notifications", func(c *gin.Context) {
		var notification models.Notification
		if err := c.ShouldBindJSON(&notification); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ввод"})
			return
		}
		if err := database.CreateNotification(pool, &notification); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания уведомления"})
			return
		}
		c.JSON(http.StatusCreated, notification)
	})

	r.GET("/notifications", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Query("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор пользователя"})
			return
		}
		notifications, err := database.GetNotificationsByUserID(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения уведомлений"})
			return
		}
		c.JSON(http.StatusOK, notifications)
	})

	r.PUT("/notifications/{id}/read", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор уведомления"})
			return
		}
		err = database.MarkNotificationAsRead(pool, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка пометки уведомления как прочитанного"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Уведомление помечено как прочитанное"})
	})

	r.DELETE("/notifications/{id}", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор уведомления"})
			return
		}
		err = database.DeleteNotification(pool, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления уведомления"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Уведомление успешно удалено"})
	})

	r.POST("/payment_reminders", func(c *gin.Context) {
		var reminder models.PaymentReminder
		if err := c.ShouldBindJSON(&reminder); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ввод"})
			return
		}
		if err := database.CreatePaymentReminder(pool, &reminder); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания напоминания о платеже"})
			return
		}
		c.JSON(http.StatusCreated, reminder)
	})

	r.GET("/payment_reminders", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Query("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор пользователя"})
			return
		}
		reminders, err := database.GetPaymentRemindersByUserID(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения напоминаний о платежах"})
			return
		}
		c.JSON(http.StatusOK, reminders)
	})

	r.GET("/payment_reminders/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор напоминания"})
			return
		}
		reminder, err := database.GetPaymentReminderByID(pool, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Напоминание о платеже не найдено"})
			return
		}
		c.JSON(http.StatusOK, reminder)
	})

	r.PUT("/payment_reminders/:id", func(c *gin.Context) {
		var reminder models.PaymentReminder
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор напоминания"})
			return
		}
		if err := c.ShouldBindJSON(&reminder); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ввод"})
			return
		}
		reminder.ID = id
		if err := database.UpdatePaymentReminder(pool, &reminder); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления напоминания о платеже"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Напоминание о платеже успешно обновлено"})
	})

	r.DELETE("/payment_reminders/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор напоминания"})
			return
		}
		if err := database.DeletePaymentReminder(pool, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления напоминания о платеже"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Напоминание о платеже успешно удалено"})
	})

	r.GET("/usersettings/:id", func(c *gin.Context) {
		// Получаем ID пользователя из параметра запроса
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil || userID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный или отсутствующий идентификатор пользователя"})
			return
		}

		// Получаем настройки пользователя из базы данных
		settings, err := database.GetUserSettingsByID(pool, userID)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Настройки пользователя не найдены"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при извлечении настроек пользователя"})
			}
			return
		}

		// Отправляем настройки пользователя в ответе
		c.JSON(http.StatusOK, settings)
	})

	r.PUT("/usersettings/:id", func(c *gin.Context) {
		// Получаем ID пользователя из параметра запроса
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil || userID <= 0 {
			log.Printf("Некорректный или отсутствующий идентификатор пользователя: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный или отсутствующий идентификатор пользователя"})
			return
		}

		// Создаем структуру для хранения обновленных настроек пользователя
		var settings models.UserSettings
		if err := c.ShouldBindJSON(&settings); err != nil {
			log.Printf("Ошибка при привязке данных: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный формат данных"})
			return
		}

		// Убедимся, что ID из URL совпадает с ID в теле запроса
		if settings.UserID != userID {
			log.Printf("ID пользователя в URL не совпадает с ID в теле запроса: %d != %d", userID, settings.UserID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID пользователя в URL не совпадает с ID в теле запроса"})
			return
		}

		// Получаем текущие настройки пользователя
		currentSettings, err := database.GetUserSettingsByID(pool, userID)
		if err != nil {
			log.Printf("Ошибка при извлечении настроек пользователя с ID %d: %v", userID, err)
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Настройки пользователя не найдены"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при извлечении настроек пользователя"})
			}
			return
		}

		// Если валюта изменяется, проводим конвертацию
		if settings.Currency != currentSettings.Currency {
			log.Printf("Валюта изменяется, текущая: %s, новая: %s", currentSettings.Currency, settings.Currency)

			// Конвертируем валюту для всех данных пользователя
			conversionErr := convertAllDataToNewCurrency(pool, userID, currentSettings.Currency, settings.Currency)
			if conversionErr != nil {
				log.Printf("Ошибка при конвертации данных для пользователя с ID %d: %v", userID, conversionErr)
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка при конвертации данных: %v", conversionErr)})
				return
			}
		}

		// Обновляем настройки пользователя
		if err := database.updateUserSettingsCurrency(pool, userID, settings.Currency); err != nil {
			log.Printf("Ошибка при обновлении настроек пользователя: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка при обновлении настроек пользователя: %v", err)})
			return
		}

		// Возвращаем успешный ответ
		c.JSON(http.StatusOK, gin.H{"message": "Настройки пользователя успешно обновлены"})
	})

	r.GET("/usersettings/:id/convert", func(c *gin.Context) {
		// Получаем ID пользователя из параметра запроса
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil || userID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный или отсутствующий идентификатор пользователя"})
			return
		}

		amountParam := c.DefaultQuery("amount", "0")
		amount, err := strconv.ParseFloat(amountParam, 64)
		if err != nil || amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректная сумма"})
			return
		}

		// Получаем валюту транзакции для пользователя
		transactionCurrency, err := database.GetTransactionCurrencyByUserID(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка при получении валюты транзакции: %v", err)})
			return
		}

		// Получаем настройки пользователя
		settings, err := database.GetUserSettingsByID(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при извлечении настроек пользователя"})
			return
		}

		// Получаем валюту из настроек пользователя
		userCurrency := settings.Currency

		// Получаем курсы валют параллельно
		var wg sync.WaitGroup
		var fromRate, toRate float64
		var errFromRate, errToRate error

		wg.Add(2)

		// Получаем курс для валюты транзакции
		go func() {
			defer wg.Done()
			fromRate, errFromRate = utils.GetCurrencyRate(transactionCurrency)
		}()

		// Получаем курс для валюты пользователя
		go func() {
			defer wg.Done()
			toRate, errToRate = utils.GetCurrencyRate(userCurrency)
		}()

		// Ожидаем завершения обеих горутин
		wg.Wait()

		// Проверка ошибок при получении курсов
		if errFromRate != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка получения курса для валюты %s: %v", transactionCurrency, errFromRate)})
			return
		}
		if errToRate != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка получения курса для валюты %s: %v", userCurrency, errToRate)})
			return
		}

		// Конвертируем сумму
		convertedAmount := amount * (toRate / fromRate)

		// Отправляем результат
		c.JSON(http.StatusOK, gin.H{
			"original_amount":  amount,
			"from_currency":    transactionCurrency,
			"to_currency":      userCurrency,
			"converted_amount": convertedAmount,
		})

		// Дополнительно конвертировать сумму бюджета, если валюта отличается от валюты пользователя
		if transactionCurrency != userCurrency {
			// Конвертировать бюджеты
			budgets, err := database.GetBudgetsByUserID(pool, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении бюджета"})
				return
			}

			for _, budget := range budgets {
				// Если валюта бюджета отличается от валюты пользователя, конвертируем бюджет
				if budget.Currency != userCurrency {
					convertedBudgetAmount := budget.Amount * (toRate / fromRate)
					updatedBudget := budget
					updatedBudget.Amount = convertedBudgetAmount

					// Обновить в базе данных
					err := database.UpdateBudget(pool, &updatedBudget)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка обновления бюджета: %v", err)})
						return
					}
				}
			}
		}
	})

	r.POST("/goals", func(c *gin.Context) {
		var goal models.Goal
		if err := c.ShouldBindJSON(&goal); err != nil {
			log.Printf("Ошибка привязки JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ввод данных"})
			return
		}
		log.Printf("Полученные данные для создания цели: %+v", goal)

		// Проверка существования пользователя
		if goal.UserID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Не указан ID пользователя"})
			return
		}

		// Создание цели
		if err := database.CreateGoal(pool, &goal); err != nil {
			log.Printf("Ошибка при создании цели: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании цели"})
			return
		}
		c.JSON(http.StatusCreated, goal)
	})

	// Получение списка целей по user_id
	r.GET("/goals", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.DefaultQuery("user_id", "0"))
		if err != nil || userID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор пользователя"})
			return
		}

		goals, err := database.GetAllGoals(pool, userID) // Используем GetAllGoals
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении списка целей"})
			return
		}
		c.JSON(http.StatusOK, goals)
	})

	// Обновление существующей цели
	r.PUT("/goals/:id", func(c *gin.Context) {
		var goal models.Goal
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			log.Printf("Некорректный идентификатор цели: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор цели"})
			return
		}

		if err := c.ShouldBindJSON(&goal); err != nil {
			log.Printf("Ошибка привязки JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ввод данных"})
			return
		}

		goal.ID = id
		log.Printf("Обновляем цель с данными: %+v", goal)

		if err := database.UpdateGoal(pool, &goal); err != nil {
			log.Printf("Ошибка обновления цели: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении цели"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Цель успешно обновлена"})
	})

	// Удаление цели
	r.DELETE("/goals/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор цели"})
			return
		}

		if err := database.DeleteGoal(pool, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении цели"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Цель успешно удалена"})
	})

	r.PATCH("/goals/:id/progress", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор цели"})
			return
		}

		var progress struct {
			Amount decimal.Decimal `json:"amount"` // Сумма прогресса в формате decimal
		}

		// Привязка данных из JSON тела запроса
		if err := c.ShouldBindJSON(&progress); err != nil {
			log.Printf("Ошибка привязки JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный формат данных прогресса"})
			return
		}

		// Проверка на положительность прогресса
		if progress.Amount.LessThanOrEqual(decimal.Zero) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Прогресс должен быть положительным числом"})
			return
		}

		// Обновление текущего прогресса цели в базе данных
		if err := database.UpdateGoalProgress(pool, id, progress.Amount); err != nil {
			log.Printf("Ошибка при обновлении прогресса: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении прогресса"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Прогресс успешно обновлен"})
	})

	if err := r.Run("localhost:8080"); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}
