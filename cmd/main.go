package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"log"
	"net/http"
	"strconv"
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
		origin := c.Request.Header.Get("Origin")
		if origin == "http://localhost:3000" || origin == "http://localhost:3001" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}

		c.Next()
	}
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

		if err := c.ShouldBindJSON(&transaction); err != nil {
			log.Printf("Ошибка привязки JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ввод", "details": err.Error()})
			return
		}

		log.Printf("Полученные данные для создания транзакции: %+v", transaction)

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
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil || userID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный или отсутствующий идентификатор пользователя"})
			return
		}

		settings, err := database.GetUserSettingsByID(pool, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Настройки пользователя не найдены"})
			return
		}

		c.JSON(http.StatusOK, settings)
	})

	r.PUT("/usersettings/:id", func(c *gin.Context) {
		var settings models.UserSettings
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil || userID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор пользователя"})
			return
		}
		settings.UserID = userID

		if err := c.ShouldBindJSON(&settings); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := database.UpdateUserSettings(pool, &settings); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Настройки пользователя успешно обновлены"})
	})

	r.GET("/convert", func(c *gin.Context) {
		from := c.DefaultQuery("from", "")        // Валюта для конвертации (from)
		to := c.DefaultQuery("to", "")            // Валюта в которую конвертировать (to)
		amountStr := c.DefaultQuery("amount", "") // Сумма для конвертации (amount)

		if from == "" || to == "" || amountStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Отсутствуют параметры 'from', 'to' или 'amount'"})
			return
		}

		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверное значение 'amount'"})
			return
		}

		// Получение актуальных курсов валют
		rates, err := getCachedRates() // Ссылка на функцию для получения актуальных курсов
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения курсов валют"})
			return
		}

		fromRate, fromExists := rates[from]
		toRate, toExists := rates[to]
		if !fromExists || !toExists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Валюта не найдена"})
			return
		}

		result := amount * (toRate / fromRate)
		response := map[string]interface{}{
			"from":   from,
			"to":     to,
			"amount": amount,
			"result": result,
		}

		c.JSON(http.StatusOK, response)
	})

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}
