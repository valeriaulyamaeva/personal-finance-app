package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"log"
	"net/http"
	"regexp"
	"strconv"
)

func validateEmail(email string) error {
	emailRegex := `^[^\s@]+@[^\s@]+\.[^\s@]+$`
	if !regexp.MustCompile(emailRegex).MatchString(email) {
		return fmt.Errorf("некорректный формат email")
	}
	return nil
}

// CreateUserHandler processes user registration.
func CreateUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User

		// Decode JSON
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "Некорректный JSON формат", http.StatusBadRequest)
			log.Printf("Ошибка декодирования JSON: %v", err)
			return
		}

		// Validate fields
		if user.Name == "" || user.Email == "" || user.Password == "" {
			http.Error(w, "Все поля обязательны для заполнения", http.StatusBadRequest)
			log.Printf("Ошибка валидации данных: %+v", user)
			return
		}

		// Validate email format
		if err := validateEmail(user.Email); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			log.Printf("Ошибка проверки email: %s", user.Email)
			return
		}

		// Register user in database
		if err := database.RegisterUser(pool, &user); err != nil {
			http.Error(w, "Ошибка при регистрации пользователя", http.StatusInternalServerError)
			log.Printf("Ошибка регистрации пользователя: %v", err)
			return
		}

		log.Printf("Пользователь успешно зарегистрирован: ID = %d", user.ID)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Пользователь успешно зарегистрирован",
			"user_id": user.ID,
		})
	}
}

// Валидация ID
func validateID(idStr string) (int, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("некорректный ID")
	}
	return id, nil
}

// Декодирование JSON тела
func decodeJSONBody(r *http.Request, dst interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return fmt.Errorf("некорректный JSON формат: %w", err)
	}
	return nil
}

// Обработчик получения данных пользователя
func GetUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := validateID(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user, err := database.GetUserByID(pool, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "Пользователь не найден", http.StatusNotFound)
			} else {
				http.Error(w, "Ошибка получения данных пользователя", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

// Обработчик обновления данных пользователя
func UpdateUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := validateID(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var user models.User
		if err := decodeJSONBody(r, &user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		user.ID = id

		if err := database.UpdateUser(pool, &user); err != nil {
			http.Error(w, "Не удалось обновить пользователя", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Пользователь успешно обновлен"})
	}
}

// Обработчик удаления пользователя
func DeleteUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := validateID(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := database.DeleteUser(pool, id); err != nil {
			http.Error(w, "Не удалось удалить пользователя", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Пользователь успешно удален"})
	}
}
