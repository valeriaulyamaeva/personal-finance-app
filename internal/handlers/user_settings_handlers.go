package handlers

import (
	"encoding/json"
	"fmt"
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

// ConvertCurrencyHandler performs currency conversion
func ConvertCurrencyHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract query parameters
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		amountStr := r.URL.Query().Get("amount")

		// Check if required parameters are missing
		if from == "" || to == "" || amountStr == "" {
			http.Error(w, "Missing parameters 'from', 'to', or 'amount'", http.StatusBadRequest)
			log.Printf("Missing query parameters from=%s, to=%s, amount=%s", from, to, amountStr)
			return
		}

		// Convert amount to float
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount <= 0 {
			http.Error(w, "Invalid 'amount' value", http.StatusBadRequest)
			log.Printf("Invalid amount value: %s", amountStr)
			return
		}

		// Fetch exchange rate for 'from' currency
		fromRate, err := utils.GetCurrencyRate(from)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching exchange rate for 'from' currency '%s': %v", from, err), http.StatusInternalServerError)
			log.Printf("Error fetching exchange rate for currency '%s': %v", from, err)
			return
		}

		// Fetch exchange rate for 'to' currency
		toRate, err := utils.GetCurrencyRate(to)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching exchange rate for 'to' currency '%s': %v", to, err), http.StatusInternalServerError)
			log.Printf("Error fetching exchange rate for currency '%s': %v", to, err)
			return
		}

		// Calculate the converted amount
		convertedAmount := amount * (toRate / fromRate)

		// Create response JSON
		response := map[string]interface{}{
			"from_currency": from,
			"to_currency":   to,
			"amount":        amount,
			"converted":     convertedAmount,
		}

		// Send JSON response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Error sending response", http.StatusInternalServerError)
			log.Printf("Error sending JSON response: %v", err)
		}
	}
}
