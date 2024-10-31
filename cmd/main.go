package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	_ "github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	database2 "github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"log"
	"net/http"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("ошибка загрузки .env файла: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), "postgres://postgres:root@localhost:5432/finance_db")
	if err != nil {
		log.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer pool.Close() // Закрываем пул соединений

	if err := database2.HashAndUpdateUserPasswords(pool); err != nil {
		log.Fatalf("ошибка обновления паролей: %v", err)
	}

	r := gin.Default()

	r.POST("/register", func(c *gin.Context) {
		var user models.User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err := database2.RegisterUser(pool, &user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
	})

	r.POST("/login", func(c *gin.Context) {
		var credentials models.User
		if err := c.ShouldBindJSON(&credentials); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		user, err := database2.AuthenticateUser(pool, credentials.Email, credentials.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		user.Password = ""
		c.JSON(http.StatusOK, gin.H{"message": "Login successful", "user": user})
	})

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("ошибка при запуске сервера: %v", err)
	}
}
