package handlers

import (
	"github.com/goccy/go-json"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"net/http"
	"strconv"
)

// GetNotificationsHandler retrieves all notifications for a user
func GetNotificationsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		notifications, err := database.GetNotificationsByUserID(pool, userID)
		if err != nil {
			http.Error(w, "Error retrieving notifications", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(notifications)
	}
}

// MarkNotificationAsReadHandler marks a notification as read
func MarkNotificationAsReadHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notificationID, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, "Invalid notification ID", http.StatusBadRequest)
			return
		}
		err = database.MarkNotificationAsRead(pool, notificationID)
		if err != nil {
			http.Error(w, "Error marking notification as read", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// DeleteNotificationHandler deletes a specific notification
func DeleteNotificationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notificationID, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, "Invalid notification ID", http.StatusBadRequest)
			return
		}
		err = database.DeleteNotification(pool, notificationID)
		if err != nil {
			http.Error(w, "Error deleting notification", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
