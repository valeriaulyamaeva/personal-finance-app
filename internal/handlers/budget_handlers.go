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
			http.Error(w, "Invalid input format", http.StatusBadRequest)
			log.Printf("Ошибка декодирования JSON: %v", err)
			return
		}

		// Проверка на наличие всех необходимых полей
		if budget.UserID == 0 || budget.CategoryID == 0 || budget.Amount <= 0 || budget.Period == "" || budget.StartDate.IsZero() || budget.EndDate.IsZero() {
			http.Error(w, "All fields are required and must be valid", http.StatusBadRequest)
			log.Printf("Некорректные данные: %+v", budget)
			return
		}

		// Логируем данные перед добавлением для отладки
		log.Printf("Добавление бюджета: %+v", budget)

		if err := database.CreateBudget(pool, &budget); err != nil {
			http.Error(w, "Failed to create budget", http.StatusInternalServerError)
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
			http.Error(w, "Invalid budget ID", http.StatusBadRequest)
			return
		}

		budget, err := database.GetBudgetByID(pool, id)
		if err != nil {
			http.Error(w, "Budget not found", http.StatusNotFound)
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
			http.Error(w, "Invalid budget ID", http.StatusBadRequest)
			return
		}

		var budget models.Budget
		if err := json.NewDecoder(r.Body).Decode(&budget); err != nil {
			http.Error(w, "Invalid data", http.StatusBadRequest)
			return
		}
		budget.ID = id

		if err := database.UpdateBudget(pool, &budget); err != nil {
			http.Error(w, "Failed to update budget", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Budget updated successfully"})
	}
}

func DeleteBudgetHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Invalid budget ID", http.StatusBadRequest)
			return
		}

		if err := database.DeleteBudget(pool, id); err != nil {
			http.Error(w, "Failed to delete budget", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Budget deleted successfully"})
	}
}
