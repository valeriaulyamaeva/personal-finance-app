package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"net/http"
	"strconv"
)

func CreateTransactionHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var transaction models.Transaction
		if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
			http.Error(w, "Invalid input data", http.StatusBadRequest)
			return
		}

		// Проверка бюджета перед добавлением транзакции
		if transaction.Type == "expense" {
			err := database.DeductFromBudget(pool, transaction.CategoryID, transaction.Amount, transaction.Date)
			if err != nil {
				http.Error(w, "Failed to deduct from budget: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		// Создание транзакции
		if err := database.CreateTransaction(pool, &transaction); err != nil {
			http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(transaction)
	}
}

// Получение всех транзакций
func GetTransactionsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transactions, err := database.GetAllTransactions(pool)
		if err != nil {
			http.Error(w, "Failed to get transactions", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(transactions)
	}
}

// Получение транзакции по ID
func GetTransactionHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
			return
		}

		transaction, err := database.GetTransactionByID(pool, id)
		if err != nil {
			http.Error(w, "Transaction not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(transaction)
	}
}

// Обновление транзакции
func UpdateTransactionHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
			return
		}

		var transaction models.Transaction
		if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
			http.Error(w, "Invalid input data", http.StatusBadRequest)
			return
		}
		transaction.ID = id

		if err := database.UpdateTransaction(pool, &transaction); err != nil {
			http.Error(w, "Failed to update transaction", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"message": "Transaction updated successfully"})
	}
}

// Удаление транзакции
func DeleteTransactionHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
			return
		}

		if err := database.DeleteTransaction(pool, id); err != nil {
			http.Error(w, "Failed to delete transaction", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"message": "Transaction deleted successfully"})
	}
}
