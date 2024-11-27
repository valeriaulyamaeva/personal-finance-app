package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"log"
	"net/http"
	"strconv"
)

func CreateBudgetHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var budget models.Budget
		if err := json.NewDecoder(r.Body).Decode(&budget); err != nil {
			http.Error(w, "Некорректный формат ввода", http.StatusBadRequest)
			log.Printf("Ошибка декодирования JSON: %v", err)
			return
		}

		if budget.UserID == 0 || budget.CategoryID == 0 || budget.Amount <= 0 || budget.Period == "" || budget.StartDate.IsZero() || budget.EndDate.IsZero() {
			http.Error(w, "Все поля должны быть заполнены и корректны", http.StatusBadRequest)
			log.Printf("Некорректные данные: %+v", budget)
			return
		}

		log.Printf("Добавление бюджета: %+v", budget)

		if err := database.CreateBudget(pool, &budget); err != nil {
			http.Error(w, "Не удалось создать бюджет", http.StatusInternalServerError)
			log.Printf("Ошибка создания бюджета в базе данных: %v", err)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(budget)
	}
}

func GetBudgetHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Некорректный ID бюджета", http.StatusBadRequest)
			return
		}

		budget, err := database.GetBudgetByID(pool, id)
		if err != nil {
			http.Error(w, "Бюджет не найден", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(budget)
	}
}

func UpdateBudgetHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Некорректный ID бюджета", http.StatusBadRequest)
			return
		}

		var budget models.Budget
		if err := json.NewDecoder(r.Body).Decode(&budget); err != nil {
			http.Error(w, "Некорректные данные", http.StatusBadRequest)
			return
		}
		budget.ID = id

		if err := database.UpdateBudget(pool, &budget); err != nil {
			http.Error(w, "Не удалось обновить бюджет", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Бюджет успешно обновлён"})
	}
}

func DeleteBudgetHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Некорректный ID бюджета", http.StatusBadRequest)
			return
		}

		if err := database.DeleteBudget(pool, id); err != nil {
			http.Error(w, "Не удалось удалить бюджет", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Бюджет успешно удалён"})
	}
}
