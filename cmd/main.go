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
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
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

func ScheduleDailyReminderNotifications(pool *pgxpool.Pool) {
	c := cron.New()

	// Запускаем задачу каждую ночь
	c.AddFunc("0 0 * * *", func() {
		log.Println("Запуск ежедневной проверки уведомлений...")

		// Текущее время
		now := time.Now()

		// Получаем все напоминания, которые должны быть отправлены
		query := `SELECT id, user_id, description, amount, due_date FROM payment_reminders WHERE due_date >= $1`
		rows, err := pool.Query(context.Background(), query, now)
		if err != nil {
			log.Printf("Ошибка при запросе напоминаний: %v", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var reminder models.PaymentReminder
			if err := rows.Scan(&reminder.ID, &reminder.UserID, &reminder.Description, &reminder.Amount, &reminder.DueDate); err != nil {
				log.Printf("Ошибка при чтении напоминания: %v", err)
				continue
			}

			// Используем новую функцию для планирования уведомлений
			if err := database.ScheduleSingleNotification(pool, &reminder); err != nil {
				log.Printf("Ошибка при планировании уведомлений для напоминания ID %d: %v", reminder.ID, err)
			}
		}
	})

	c.Start()
}

func convertAllDataToNewCurrency(pool *pgxpool.Pool, userID int, oldCurrency string, newCurrency string) error {
	// Получаем курсы валют для конвертации
	fromRate, errFromRate := utils.GetCurrencyRate(oldCurrency)
	toRate, errToRate := utils.GetCurrencyRate(newCurrency)

	if errFromRate != nil || errToRate != nil {
		log.Printf("Ошибка при получении курсов валют: %v, %v", errFromRate, errToRate)
		return fmt.Errorf("ошибка при получении курсов валют: %v, %v", errFromRate, errToRate)
	}

	conversionRate := toRate / fromRate
	log.Printf("Конвертационный курс: %.4f", conversionRate)

	// Конвертируем и обновляем валюту для всех бюджетов
	budgets, err := database.GetBudgetsByUserID(pool, userID)
	if err != nil {
		log.Printf("Ошибка при получении бюджета для пользователя с ID %d: %v", userID, err)
		return fmt.Errorf("ошибка при получении бюджета для пользователя с ID %d: %v", userID, err)
	}

	for _, budget := range budgets {
		if budget.Currency != newCurrency {
			convertedAmount := budget.Amount * conversionRate
			budget.Amount = convertedAmount
			budget.Currency = newCurrency
			if err := database.UpdateBudget(pool, &budget); err != nil {
				log.Printf("Ошибка при обновлении бюджета с ID %d: %v", budget.ID, err)
				return fmt.Errorf("ошибка при обновлении бюджета с ID %d: %v", budget.ID, err)
			}
		}
	}

	// Конвертируем и обновляем валюту для всех транзакций
	transactions, err := database.GetTransactionsByUserID(pool, userID)
	if err != nil {
		log.Printf("Ошибка при получении транзакций для пользователя с ID %d: %v", userID, err)
		return fmt.Errorf("ошибка при получении транзакций для пользователя с ID %d: %v", userID, err)
	}

	for _, transaction := range transactions {
		if transaction.Currency != newCurrency {
			convertedAmount := transaction.Amount * conversionRate
			transaction.Amount = convertedAmount
			transaction.Currency = newCurrency
			if err := database.UpdateTransaction(pool, &transaction); err != nil {
				log.Printf("Ошибка при обновлении транзакции с ID %d: %v", transaction.ID, err)
				return fmt.Errorf("ошибка при обновлении транзакции с ID %d: %v", transaction.ID, err)
			}
		}
	}

	// Конвертируем и обновляем валюту для цели
	goal, err := database.GetGoalByID(pool, userID)
	if err != nil {
		log.Printf("Ошибка при получении цели для пользователя с ID %d: %v", userID, err)
		return fmt.Errorf("ошибка при получении цели для пользователя с ID %d: %v", userID, err)
	}

	if goal.Currency != newCurrency {
		convertedAmount := goal.Amount * conversionRate
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
	ScheduleDailyReminderNotifications(pool)

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

		user.Password = "" // Убираем пароль из ответа для безопасности

		// Возвращаем is_admin для проверки роли
		c.JSON(http.StatusOK, gin.H{
			"message":  "Авторизация успешна",
			"user":     user,
			"is_admin": user.IsAdmin,
		})
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

	r.DELETE("/notifications/by-notification-id/:id", func(c *gin.Context) {
		notificationID := c.Param("id")

		// Преобразуем ID в int
		id, err := strconv.Atoi(notificationID)
		if err != nil {
			log.Printf("Ошибка преобразования ID: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор уведомления"})
			return
		}
		log.Printf("Получен запрос на удаление уведомления с ID: %d", id)

		// Пытаемся удалить уведомление
		err = database.DeleteNotificationByNotificationID(pool, id)
		if err != nil {
			log.Printf("Ошибка при удалении уведомления с ID %d: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления уведомления"})
			return
		}

		log.Printf("Уведомление с ID %d успешно удалено", id)
		c.JSON(http.StatusOK, gin.H{"message": "Уведомление успешно удалено"})
	})

	r.POST("/payment_reminders", func(c *gin.Context) {
		var reminder models.PaymentReminder
		if err := c.ShouldBindJSON(&reminder); err != nil {
			// Логируем ошибку валидации данных
			log.Printf("Failed to bind JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ввод"})
			return
		}

		if err := database.CreatePaymentReminder(pool, &reminder); err != nil {
			// Логируем ошибку при создании напоминания
			log.Printf("Failed to create payment reminder: %v", err)
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

		// Получаем дату для фильтрации, если указана
		dateFilter := c.DefaultQuery("date", "")
		var reminders []models.PaymentReminder
		if dateFilter == "" {
			reminders, err = database.GetPaymentRemindersByUserID(pool, userID)
		} else {
			// Преобразуем строку даты в формат time.Time
			date, err := time.Parse("2006-01-02", dateFilter)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректная дата"})
				return
			}
			// Получаем напоминания для указанной даты
			reminders, err = database.GetPaymentRemindersByUserIDAndDate(pool, userID, date)
		}

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

		log.Printf("Получение напоминаний для пользователя с ID: %d", id)

		reminder, err := database.GetPaymentRemindersByUserID(pool, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		if len(reminder) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "У вас нет напоминаний"})
			return
		}

		c.JSON(http.StatusOK, reminder)
	})

	r.PUT("/payment_reminders/:id", func(c *gin.Context) {
		var reminder models.PaymentReminder
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			// Логируем ошибку при получении ID
			log.Printf("Invalid reminder ID: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор напоминания"})
			return
		}

		if err := c.ShouldBindJSON(&reminder); err != nil {
			// Логируем ошибку валидации данных
			log.Printf("Failed to bind JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ввод"})
			return
		}
		reminder.ID = id

		if err := database.UpdatePaymentReminder(pool, &reminder); err != nil {
			// Логируем ошибку при обновлении напоминания
			log.Printf("Failed to update payment reminder: %v", err)
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

			// Конвертируем данные в новой валюте
			if err := convertAllDataToNewCurrency(pool, userID, currentSettings.Currency, settings.Currency); err != nil {
				log.Printf("Ошибка при конвертации данных для пользователя с ID %d: %v", userID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка при конвертации данных: %v", err)})
				return
			}

			// Обновляем валюту в таблице настроек пользователя
			currentSettings.Currency = settings.Currency
			if err := database.UpdateUserSettings(pool, currentSettings); err != nil {
				log.Printf("Ошибка при обновлении настроек пользователя с ID %d: %v", userID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении настроек пользователя"})
				return
			}
		}

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

	r.GET("/users", func(c *gin.Context) {
		users, err := database.GetAllUsers(pool)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении списка пользователей"})
			return
		}
		c.JSON(http.StatusOK, users)
	})

	r.PUT("/users/:id", func(c *gin.Context) {
		var user models.User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
			return
		}

		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
			return
		}
		user.ID = userID

		if err := database.UpdateUser(pool, &user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении пользователя"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Пользователь успешно обновлен"})
	})

	r.DELETE("/users/:id", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			log.Printf("Invalid user ID: %v", c.Param("id"))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
			return
		}

		log.Printf("Attempting to delete user with ID: %d", userID)
		if err := database.DeleteUser(pool, userID); err != nil {
			log.Printf("Error deleting user with ID %d: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении пользователя"})
			return
		}

		log.Printf("User with ID %d successfully deleted", userID)
		c.JSON(http.StatusOK, gin.H{"message": "Пользователь успешно удален"})
	})

	r.POST("/users", func(c *gin.Context) {
		var newUser models.User

		// Парсим тело запроса в структуру `newUser`
		if err := c.ShouldBindJSON(&newUser); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
			return
		}

		// Хэшируем пароль перед сохранением
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка хэширования пароля"})
			return
		}
		newUser.Password = string(hashedPassword)

		// Создаем пользователя через функцию CreateUser
		err = database.CreateUser(pool, &newUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка создания пользователя: %v", err)})
			return
		}

		// Возвращаем успешный ответ с данными нового пользователя
		c.JSON(http.StatusCreated, gin.H{
			"message": "Пользователь успешно создан",
			"user": gin.H{
				"id":    newUser.ID,
				"name":  newUser.Name,
				"email": newUser.Email,
			},
		})
	})

	r.GET("/admin/user_stats", database.GetUserStats(pool))

	r.GET("/admin/registrations_by_month", database.GetRegistrationsByMonth(pool))

	r.GET("/admin/user_roles", database.GetUserRoles(pool))

	r.POST("/family_accounts", func(c *gin.Context) {
		var request struct {
			Nickname    string `json:"nickname"`
			OwnerUserID int    `json:"owner_user_id"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
			return
		}

		familyID, err := database.CreateFamilyAccount(pool, request.Nickname, request.OwnerUserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка создания семейного аккаунта: %v", err)})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Семейный аккаунт успешно создан",
			"family_account": gin.H{
				"id":       familyID,
				"nickname": request.Nickname,
			},
		})
	})

	r.POST("/family_accounts/join", func(c *gin.Context) {
		var request struct {
			UserID   int    `json:"user_id"`
			Nickname string `json:"nickname"`
			Role     string `json:"role"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
			return
		}

		if err := database.JoinFamilyAccount(pool, request.UserID, request.Nickname, request.Role); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка присоединения к семейному аккаунту: %v", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Пользователь успешно присоединился к семейному аккаунту"})
	})

	r.GET("/family_accounts/:id/members", func(c *gin.Context) {
		familyAccountID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID семейного аккаунта"})
			return
		}

		members, err := database.GetFamilyMembers(pool, familyAccountID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка получения участников семейного аккаунта: %v", err)})
			return
		}

		c.JSON(http.StatusOK, members)
	})

	r.GET("/users/:id/family_account", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
			return
		}

		familyAccountID, err := database.GetFamilyAccountByUser(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка получения семейного аккаунта: %v", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{"family_account_id": familyAccountID})
	})

	r.GET("/users/:id/family_account/check", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
			return
		}

		familyAccountID, err := database.GetFamilyAccountByUser(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка получения семейного аккаунта: %v", err)})
			return
		}

		// Получаем ID владельца семейного аккаунта
		ownerID, err := database.GetFamilyAccountOwnerID(pool, familyAccountID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка получения ID владельца: %v", err)})
			return
		}

		// Перенаправляем пользователя на страницу создателя
		if userID == ownerID {
			c.JSON(http.StatusOK, gin.H{"message": "Вы являетесь владельцем семейного аккаунта", "redirect_to": "/owner_page"})
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "Вы состоите в семейном аккаунте", "redirect_to": fmt.Sprintf("/family_account/%d", familyAccountID)})
		}
	})

	if err := r.Run("localhost:8080"); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}

	select {}
}
