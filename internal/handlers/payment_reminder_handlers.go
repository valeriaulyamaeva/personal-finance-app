package handlers

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"net/http"
	"strconv"
)

// CreatePaymentReminderHandler handles the creation of a new payment reminder.
func CreatePaymentReminderHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reminder models.PaymentReminder
		if err := json.NewDecoder(r.Body).Decode(&reminder); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		if err := database.CreatePaymentReminder(pool, &reminder); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(reminder)
	}
}

// GetPaymentReminderHandler retrieves a single payment reminder by ID.
func GetPaymentReminderHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}
		reminder, err := database.GetPaymentReminderByID(pool, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(reminder)
	}
}

// UpdatePaymentReminderHandler handles updating an existing payment reminder.
func UpdatePaymentReminderHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reminder models.PaymentReminder
		if err := json.NewDecoder(r.Body).Decode(&reminder); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		if err := database.UpdatePaymentReminder(pool, &reminder); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(reminder)
	}
}

// DeletePaymentReminderHandler deletes a payment reminder by ID.
func DeletePaymentReminderHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}
		if err := database.DeletePaymentReminder(pool, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Payment reminder deleted successfully"})
	}
}
