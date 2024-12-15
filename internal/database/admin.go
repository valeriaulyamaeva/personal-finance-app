package database

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/context"
	"log"
	"net/http"
)

type UserStat struct {
	TotalUsers   int `json:"total_users"`
	AdminUsers   int `json:"admin_users"`
	RegularUsers int `json:"regular_users"`
}

type MonthlyRegistrations struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

func GetUserStats(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var stats UserStat
		query := `
			SELECT 
				(SELECT COUNT(*) FROM users) AS total_users,
				(SELECT COUNT(*) FROM users WHERE is_admin = true) AS admin_users,
				(SELECT COUNT(*) FROM users WHERE is_admin = false) AS regular_users
		`

		err := pool.QueryRow(context.Background(), query).Scan(&stats.TotalUsers, &stats.AdminUsers, &stats.RegularUsers)
		if err != nil {
			log.Printf("Error fetching user stats: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения статистики пользователей"})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

func GetRegistrationsByMonth(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `
			SELECT 
				TO_CHAR(created_at, 'YYYY-MM') AS month, 
				COUNT(*) 
			FROM users 
			GROUP BY month 
			ORDER BY month ASC
		`

		rows, err := pool.Query(context.Background(), query)
		if err != nil {
			log.Printf("Error fetching registrations by month: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения регистраций по месяцам"})
			return
		}
		defer rows.Close()

		var registrations []MonthlyRegistrations
		for rows.Next() {
			var reg MonthlyRegistrations
			err = rows.Scan(&reg.Month, &reg.Count)
			if err != nil {
				log.Printf("Error scanning registration data: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обработки данных регистраций"})
				return
			}
			registrations = append(registrations, reg)
		}

		if rows.Err() != nil {
			log.Printf("Row iteration error: %v", rows.Err())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обработки регистраций по месяцам"})
			return
		}

		c.JSON(http.StatusOK, registrations)
	}
}

func GetUserRoles(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `
			SELECT 
				CASE 
					WHEN is_admin = true THEN 'admin'
					ELSE 'user'
				END AS role,
				COUNT(*) 
			FROM users 
			GROUP BY role
		`

		rows, err := pool.Query(context.Background(), query)
		if err != nil {
			log.Printf("Error fetching user roles: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения ролей пользователей"})
			return
		}
		defer rows.Close()

		type UserRole struct {
			Role  string `json:"role"`
			Count int    `json:"count"`
		}

		var roles []UserRole
		for rows.Next() {
			var role UserRole
			err = rows.Scan(&role.Role, &role.Count)
			if err != nil {
				log.Printf("Error scanning user role data: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обработки данных ролей"})
				return
			}
			roles = append(roles, role)
		}

		if rows.Err() != nil {
			log.Printf("Row iteration error: %v", rows.Err())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обработки данных ролей пользователей"})
			return
		}

		c.JSON(http.StatusOK, roles)
	}
}
