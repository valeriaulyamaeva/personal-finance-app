package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

// In handlers/user_settings_handlers.go
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
			http.Error(w, "User settings not found", http.StatusNotFound)
			log.Printf("GET /usersettings/%d failed: %v", id, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	}
}

// UpdateUserSettingsHandler updates user settings by user ID
func UpdateUserSettingsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil || id <= 0 {
			http.Error(w, "Некорректный ID пользователя", http.StatusBadRequest)
			return
		}

		var settings models.UserSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			http.Error(w, "Некорректный формат JSON", http.StatusBadRequest)
			return
		}
		settings.UserID = id

		if err := database.UpdateUserSettings(pool, &settings); err != nil {
			http.Error(w, "Ошибка обновления настроек пользователя", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Настройки пользователя успешно обновлены"})
	}
}
