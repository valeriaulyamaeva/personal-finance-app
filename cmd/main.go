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

	type UpdateUserSettingsRequest struct {
		Currency           string `json:"currency"`
		OldCurrency        string `json:"old_currency"`
		Theme              string `json:"theme"`
		NotificationVolume int    `json:"notification_volume"`
		AutoUpdates        bool   `json:"auto_updates"`
		WeeklyReports      bool   `json:"weekly_reports"`
	}

	// Структура ответа при успешном обновлении настроек пользователя
	type UpdateUserSettingsResponse struct {
		Message  string              `json:"message"`
		Settings models.UserSettings `json:"settings"`
	}

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

	// PUT /usersettings/:id
	r.PUT("/usersettings/:id", func(c *gin.Context) {
		// Получаем ID пользователя из параметра запроса
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil || userID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный или отсутствующий идентификатор пользователя"})
			return
		}

		// Чтение данных из тела запроса
		var payload UpdateUserSettingsRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
			return
		}

		// Проверка валидности валюты
		validCurrencies := map[string]bool{
			"BYN": true, "RUB": true, "PLN": true, "KRW": true,
			"JPY": true, "USD": true, "EUR": true,
		}
		if payload.Currency != "" && !validCurrencies[payload.Currency] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неподдерживаемая валюта"})
			return
		}

		settings := &models.UserSettings{
			UserID:             userID,
			Currency:           payload.Currency,
			OldCurrency:        payload.OldCurrency,
			Theme:              payload.Theme,
			NotificationVolume: payload.NotificationVolume,
			AutoUpdates:        payload.AutoUpdates,
			WeeklyReports:      payload.WeeklyReports,
		}

		// Вызов функции для обновления настроек в базе данных
		if err := database.UpdateUserSettings(pool, settings); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления настроек пользователя"})
			return
		}

		// Отправка успешного ответа с обновленными данными
		response := UpdateUserSettingsResponse{
			Message:  "Настройки успешно обновлены",
			Settings: *settings,
		}
		c.JSON(http.StatusOK, response)
	})

	r.GET("/usersettings/:id/convert", func(c *gin.Context) {
		// Получаем ID пользователя из параметра запроса
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil || userID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный или отсутствующий идентификатор пользователя"})
			return
		}

		// Получаем настройки пользователя из базы данных, передавая пул
		settings, err := database.GetUserSettingsByID(pool, userID) // Передаем пул как аргумент
		if err != nil {
			// Обрабатываем ошибки получения настроек
			if err.Error() == "настройки пользователя с ID не найдены" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Настройки пользователя не найдены"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при извлечении настроек пользователя"})
			}
			return
		}

		// Получаем параметры запроса для конвертации
		amountParam := c.DefaultQuery("amount", "0") // Получаем параметр "amount", если не задан — ставим 0
		amount, err := strconv.ParseFloat(amountParam, 64)
		if err != nil || amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректная сумма"})
			return
		}

		// Получаем валюты из настроек пользователя
		fromCurrency := settings.OldCurrency // Ваша старая валюта
		toCurrency := settings.Currency      // Текущая валюта

		// Получаем курсы валют
		fromRate, err := utils.GetCurrencyRate(fromCurrency)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка получения курса для валюты %s: %v", fromCurrency, err)})
			return
		}

		toRate, err := utils.GetCurrencyRate(toCurrency)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка получения курса для валюты %s: %v", toCurrency, err)})
			return
		}

		// Конвертируем сумму
		convertedAmount := amount * (toRate / fromRate)

		// Отправляем ответ с результатами
		c.JSON(http.StatusOK, gin.H{
			"original_amount":  amount,
			"from_currency":    fromCurrency,
			"to_currency":      toCurrency,
			"converted_amount": convertedAmount,
		})
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

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}
