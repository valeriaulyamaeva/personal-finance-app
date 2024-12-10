package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/valeriaulyamaeva/personal-finance-app/utils"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

// Response structure for success messages
type SuccessResponse struct {
	Message string `json:"message"`
}

// ErrorResponse structure for error messages
type ErrorResponse struct {
	Error string `json:"error"`
}

// GetUserSettingsHandler retrieves user settings based on the user ID
func GetUserSettingsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil || id <= 0 {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			log.Printf("Invalid user ID in GET request: %v", id)
			return
		}

		log.Printf("Handling GET request for user_id=%d", id)

		settings, err := database.GetUserSettingsByID(pool, id)
		if err != nil {
			// Return error response with appropriate status
			http.Error(w, fmt.Sprintf("User settings not found: %v", err), http.StatusNotFound)
			log.Printf("GET /usersettings/%d failed: %v", id, err)
			return
		}

		// Send JSON response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(settings); err != nil {
			http.Error(w, "Error encoding JSON response", http.StatusInternalServerError)
			log.Printf("Error encoding JSON response for user_id=%d: %v", id, err)
		}
	}
}

// UpdateUserSettingsHandler updates user settings in the database
func UpdateUserSettingsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil || id <= 0 {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			log.Printf("Invalid user ID in PUT request: %v", id)
			return
		}

		var settings models.UserSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			log.Printf("Error decoding JSON: %v", err)
			return
		}

		// Set the user ID from the URL path
		settings.UserID = id
		log.Printf("Updating settings for user_id=%d with settings: %+v", id, settings)

		// Update user settings in the database
		if err := database.UpdateUserSettings(pool, &settings); err != nil {
			http.Error(w, fmt.Sprintf("Error updating user settings: %v", err), http.StatusInternalServerError)
			log.Printf("Error updating settings for user_id=%d: %v", id, err)
			return
		}

		// Return success message
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := SuccessResponse{Message: "User settings updated successfully"}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Error sending response", http.StatusInternalServerError)
			log.Printf("Error sending response for user_id=%d: %v", id, err)
		}
	}
}

func ConvertCurrencyHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем ID пользователя из параметра запроса
		userID, err := strconv.Atoi(c.Param("id"))
		if err != nil || userID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный или отсутствующий идентификатор пользователя"})
			return
		}

		// Получаем настройки пользователя из базы данных, передавая пул
		settings, err := database.GetUserSettingsByID(pool, userID)
		if err != nil {
			if err.Error() == "настройки пользователя с ID не найдены" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Настройки пользователя не найдены"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при извлечении настроек пользователя"})
			}
			return
		}

		// Получаем параметры запроса для конвертации
		amountParam := c.DefaultQuery("amount", "0")
		amount, err := strconv.ParseFloat(amountParam, 64)
		if err != nil || amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректная сумма"})
			return
		}

		// Получаем валюты из настроек пользователя
		fromCurrency := settings.OldCurrency // старая валюта
		toCurrency := settings.Currency      // новая валюта

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
	}
}
