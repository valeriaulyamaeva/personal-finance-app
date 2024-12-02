package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"log"
	"net/http"
	"strconv"
)

// CreateGoalHandler создает новую цель
func CreateGoalHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var goal models.Goal
		if err := json.NewDecoder(r.Body).Decode(&goal); err != nil {
			http.Error(w, "Некорректный формат ввода", http.StatusBadRequest)
			log.Printf("Ошибка декодирования JSON: %v", err)
			return
		}

		if goal.UserID == 0 || goal.Amount <= 0 || goal.Name == "" || goal.Deadline.IsZero() || goal.CreatedAt.IsZero() {
			http.Error(w, "Все поля должны быть заполнены и корректны", http.StatusBadRequest)
			log.Printf("Некорректные данные: %+v", goal)
			return
		}

		log.Printf("Добавление цели: %+v", goal)

		if err := database.CreateGoal(pool, &goal); err != nil {
			http.Error(w, "Не удалось создать цель", http.StatusInternalServerError)
			log.Printf("Ошибка создания цели в базе данных: %v", err)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(goal)
	}
}

// GetGoalHandler извлекает цель по ID
func GetGoalHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Некорректный ID цели", http.StatusBadRequest)
			return
		}

		goal, err := database.GetGoalByID(pool, id)
		if err != nil {
			http.Error(w, "Цель не найдена", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(goal)
	}
}

// GetAllGoalsHandler извлекает все цели пользователя
func GetAllGoalsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID, err := strconv.Atoi(vars["user_id"])
		if err != nil {
			http.Error(w, "Некорректный ID пользователя", http.StatusBadRequest)
			return
		}

		goals, err := database.GetAllGoals(pool, userID)
		if err != nil {
			http.Error(w, "Не удалось получить цели", http.StatusInternalServerError)
			log.Printf("Ошибка получения целей из базы данных: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(goals)
	}
}

// UpdateGoalHandler обновляет информацию о цели
func UpdateGoalHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Некорректный ID цели", http.StatusBadRequest)
			return
		}

		var goal models.Goal
		if err := json.NewDecoder(r.Body).Decode(&goal); err != nil {
			http.Error(w, "Некорректные данные", http.StatusBadRequest)
			return
		}
		goal.ID = id

		if err := database.UpdateGoal(pool, &goal); err != nil {
			http.Error(w, "Не удалось обновить цель", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Цель успешно обновлена"})
	}
}

// DeleteGoalHandler удаляет цель по ID
func DeleteGoalHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Некорректный ID цели", http.StatusBadRequest)
			return
		}

		if err := database.DeleteGoal(pool, id); err != nil {
			http.Error(w, "Не удалось удалить цель", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Цель успешно удалена"})
	}
}

// AddProgressToGoalHandler добавляет прогресс к цели
func AddProgressToGoalHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Некорректный ID цели", http.StatusBadRequest)
			return
		}

		var progressData struct {
			Progress float64 `json:"progress"`
		}

		if err := json.NewDecoder(r.Body).Decode(&progressData); err != nil {
			http.Error(w, "Некорректные данные", http.StatusBadRequest)
			return
		}

		// Convert float64 to decimal.Decimal
		progress := decimal.NewFromFloat(progressData.Progress)

		if err := database.AddProgressToGoal(pool, id, progress); err != nil {
			http.Error(w, "Не удалось добавить прогресс", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Прогресс успешно добавлен"})
	}
}
