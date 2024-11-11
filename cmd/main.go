package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"log"
	"net/http"
	"strconv"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
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
		log.Fatalf("ошибка загрузки .env файла: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), "postgres://postgres:root@localhost:5432/finance_db")
	if err != nil {
		log.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer pool.Close()

	r := gin.Default()
	r.Use(CORSMiddleware())

	// Маршрут для регистрации
	r.POST("/register", func(c *gin.Context) {
		var user models.User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := database.RegisterUser(pool, &user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
	})

	// Маршрут для авторизации
	r.POST("/login", func(c *gin.Context) {
		var credentials models.User
		if err := c.ShouldBindJSON(&credentials); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		user, err := database.AuthenticateUser(pool, credentials.Email, credentials.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		user.Password = ""
		c.JSON(http.StatusOK, gin.H{"message": "Login successful", "user": user})
	})

	// Примеры маршрутов для категорий
	r.POST("/categories", func(c *gin.Context) {
		var category models.Category
		if err := c.ShouldBindJSON(&category); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}
		if err := database.CreateCategory(pool, &category); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
			return
		}
		c.JSON(http.StatusCreated, category)
	})

	r.GET("/categories", func(c *gin.Context) {
		categories, err := database.GetAllCategories(pool)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
			return
		}
		c.JSON(http.StatusOK, categories)
	})

	r.PUT("/categories/:id", func(c *gin.Context) {
		var category models.Category
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
			return
		}
		if err := c.ShouldBindJSON(&category); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}
		category.ID = id
		if err := database.UpdateCategory(pool, &category); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Category updated successfully"})
	})

	r.DELETE("/categories/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
			return
		}

		// Попытка удаления категории
		err = database.DeleteCategory(pool, id)
		if err != nil {
			log.Printf("Ошибка удаления категории с ID %d: %v", id, err) // Логируем ошибку на сервере
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
	})

	// Маршруты для бюджетов
	r.POST("/budgets", func(c *gin.Context) {
		var budget models.Budget
		if err := c.ShouldBindJSON(&budget); err != nil {
			log.Printf("Ошибка привязки JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}
		log.Printf("Полученные данные для создания бюджета: %+v", budget)

		if err := database.CreateBudget(pool, &budget); err != nil {
			log.Printf("Ошибка при создании бюджета: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create budget"})
			return
		}
		c.JSON(http.StatusCreated, budget)
	})

	r.GET("/budgets", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Query("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		budgets, err := database.GetBudgetsByUserID(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch budgets"})
			return
		}
		c.JSON(http.StatusOK, budgets)
	})

	r.PUT("/budgets/:id", func(c *gin.Context) {
		var budget models.Budget
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			log.Printf("Invalid budget ID: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid budget ID"})
			return
		}

		if err := c.ShouldBindJSON(&budget); err != nil {
			log.Printf("Ошибка привязки JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		budget.ID = id
		log.Printf("Обновляем бюджет с данными: %+v", budget) // Проверка данных на сервере

		if err := database.UpdateBudget(pool, &budget); err != nil {
			log.Printf("Ошибка обновления бюджета: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update budget"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Budget updated successfully"})
	})

	r.DELETE("/budgets/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid budget ID"})
			return
		}
		if err := database.DeleteBudget(pool, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete budget"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Budget deleted successfully"})
	})

	// Обработчик для создания транзакции
	r.POST("/transactions", func(c *gin.Context) {
		var transaction models.Transaction
		log.Printf("Raw transaction data: %v", c.Request.Body) // Логируем тело запроса

		if err := c.ShouldBindJSON(&transaction); err != nil {
			log.Printf("Ошибка привязки JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
			return
		}

		log.Printf("Полученные данные для создания транзакции: %+v", transaction)

		if err := database.CreateTransaction(pool, &transaction); err != nil {
			log.Printf("Ошибка при создании транзакции: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
			return
		}
		c.JSON(http.StatusCreated, transaction)
	})

	r.GET("/transactions", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Query("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		transactions, err := database.GetTransactionsByUserID(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
			return
		}
		c.JSON(http.StatusOK, transactions)
	})

	r.PUT("/transactions/:id", func(c *gin.Context) {
		var transaction models.Transaction
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
			return
		}
		if err := c.ShouldBindJSON(&transaction); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}
		transaction.ID = id

		if err := database.UpdateTransaction(pool, &transaction); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Transaction updated successfully"})
	})

	r.DELETE("/transactions/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
			return
		}
		if err := database.DeleteTransaction(pool, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete transaction"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Transaction deleted successfully"})
	})

	r.GET("/dashboard/total_balance", func(c *gin.Context) {
		userID, _ := strconv.Atoi(c.Query("user_id"))
		balance, err := GetTotalBalance(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"total_balance": balance})
	})

	r.GET("/dashboard/monthly_expenses", func(c *gin.Context) {
		userID, _ := strconv.Atoi(c.Query("user_id"))
		expenses, err := GetMonthlyExpenses(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"expenses": expenses})
	})

	r.GET("/dashboard/income_expense_summary", func(c *gin.Context) {
		userID, _ := strconv.Atoi(c.Query("user_id"))
		summary, err := GetIncomeExpenseSummary(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, summary)
	})

	r.GET("/dashboard/category_expenses", func(c *gin.Context) {
		userID, _ := strconv.Atoi(c.Query("user_id"))
		categoryExpenses, err := GetCategoryWiseExpenses(pool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"category_expenses": categoryExpenses})
	})

	// Запуск сервера
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("ошибка при запуске сервера: %v", err)
	}

}
