package routes

import (
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/handlers"
)

func SetupRouter(pool *pgxpool.Pool) *mux.Router {
	r := mux.NewRouter()

	// Маршруты для пользователей
	users := r.PathPrefix("/api/users").Subrouter()
	users.HandleFunc("", handlers.CreateUserHandler(pool)).Methods("POST")
	users.HandleFunc("/{id}", handlers.GetUserHandler(pool)).Methods("GET")
	users.HandleFunc("/{id}", handlers.UpdateUserHandler(pool)).Methods("PUT")
	users.HandleFunc("/{id}", handlers.DeleteUserHandler(pool)).Methods("DELETE")

	// Группа маршрутов для транзакций
	transactions := r.PathPrefix("/api/transactions").Subrouter()
	transactions.HandleFunc("", handlers.CreateTransactionHandler(pool)).Methods("POST")
	transactions.HandleFunc("", handlers.GetTransactionsHandler(pool)).Methods("GET")
	transactions.HandleFunc("/{id}", handlers.GetTransactionHandler(pool)).Methods("GET")
	transactions.HandleFunc("/{id}", handlers.UpdateTransactionHandler(pool)).Methods("PUT")
	transactions.HandleFunc("/{id}", handlers.DeleteTransactionHandler(pool)).Methods("DELETE")

	// Группа маршрутов для бюджетов
	budgets := r.PathPrefix("/api/budgets").Subrouter()
	budgets.HandleFunc("", handlers.CreateBudgetHandler(pool)).Methods("POST")
	budgets.HandleFunc("/{id}", handlers.GetBudgetHandler(pool)).Methods("GET")
	budgets.HandleFunc("/{id}", handlers.UpdateBudgetHandler(pool)).Methods("PUT")
	budgets.HandleFunc("/{id}", handlers.DeleteBudgetHandler(pool)).Methods("DELETE")

	// Группа маршрутов для категорий
	categories := r.PathPrefix("/categories").Subrouter()
	categories.HandleFunc("", handlers.CreateCategoryHandler(pool)).Methods("POST")
	categories.HandleFunc("", handlers.GetAllCategoriesHandler(pool)).Methods("GET")
	categories.HandleFunc("/{id}", handlers.GetCategoryHandler(pool)).Methods("GET")
	categories.HandleFunc("/{id}", handlers.UpdateCategoryHandler(pool)).Methods("PUT")
	categories.HandleFunc("/{id}", handlers.DeleteCategoryHandler(pool)).Methods("DELETE")

	// Группа маршрутов для уведомлений
	notifications := r.PathPrefix("/api/notifications").Subrouter()
	notifications.HandleFunc("", handlers.GetNotificationsHandler(pool)).Methods("GET")
	notifications.HandleFunc("/{id}/read", handlers.MarkNotificationAsReadHandler(pool)).Methods("PUT")
	notifications.HandleFunc("/{id}", handlers.DeleteNotificationHandler(pool)).Methods("DELETE")

	// Группа маршрутов для напоминаний о платежах
	paymentReminders := r.PathPrefix("/api/payment_reminders").Subrouter()
	paymentReminders.HandleFunc("", handlers.CreatePaymentReminderHandler(pool)).Methods("POST")
	paymentReminders.HandleFunc("/{id}", handlers.GetPaymentReminderHandler(pool)).Methods("GET")
	paymentReminders.HandleFunc("/{id}", handlers.UpdatePaymentReminderHandler(pool)).Methods("PUT")
	paymentReminders.HandleFunc("/{id}", handlers.DeletePaymentReminderHandler(pool)).Methods("DELETE")

	userSettings := r.PathPrefix("/usersettings").Subrouter()
	userSettings.HandleFunc("/{id}", handlers.GetUserSettingsHandler(pool)).Methods("GET")
	userSettings.HandleFunc("/{id}", handlers.UpdateUserSettingsHandler(pool)).Methods("PUT")

	exchangeRates := r.PathPrefix("/exchange-rates").Subrouter()
	exchangeRates.HandleFunc("", handlers.GetExchangeRatesHandler).Methods("GET")
	exchangeRates.HandleFunc("/{code}", handlers.GetCurrencyRateHandler).Methods("GET")
	exchangeRates.HandleFunc("/convert", handlers.ConvertCurrencyHandler).Methods("GET")
	return r
}
