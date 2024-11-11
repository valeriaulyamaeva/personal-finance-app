package routes

import (
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/handlers"
)

func SetupRouter(pool *pgxpool.Pool) *mux.Router {
	r := mux.NewRouter()

	// Маршруты для пользователей
	r.HandleFunc("/api/users", handlers.CreateUserHandler(pool)).Methods("POST")
	r.HandleFunc("/api/users/{id}", handlers.GetUserHandler(pool)).Methods("GET")
	r.HandleFunc("/api/users/{id}", handlers.UpdateUserHandler(pool)).Methods("PUT")
	r.HandleFunc("/api/users/{id}", handlers.DeleteUserHandler(pool)).Methods("DELETE")

	// Маршруты для транзакций
	r.HandleFunc("/api/transactions", handlers.CreateTransactionHandler(pool)).Methods("POST")
	r.HandleFunc("/api/transactions", handlers.GetTransactionsHandler(pool)).Methods("GET")
	r.HandleFunc("/api/transactions/{id}", handlers.GetTransactionHandler(pool)).Methods("GET")
	r.HandleFunc("/api/transactions/{id}", handlers.UpdateTransactionHandler(pool)).Methods("PUT")
	r.HandleFunc("/api/transactions/{id}", handlers.DeleteTransactionHandler(pool)).Methods("DELETE")

	// Маршруты для бюджетов
	r.HandleFunc("/api/budgets", handlers.CreateBudgetHandler(pool)).Methods("POST")
	r.HandleFunc("/api/budgets/{id}", handlers.GetBudgetHandler(pool)).Methods("GET")
	r.HandleFunc("/api/budgets/{id}", handlers.UpdateBudgetHandler(pool)).Methods("PUT")
	r.HandleFunc("/api/budgets/{id}", handlers.DeleteBudgetHandler(pool)).Methods("DELETE")

	// Маршруты для категорий
	r.HandleFunc("/categories", handlers.CreateCategoryHandler(pool)).Methods("POST")
	r.HandleFunc("/categories", handlers.GetAllCategoriesHandler(pool)).Methods("GET")
	r.HandleFunc("/categories/{id}", handlers.GetCategoryHandler(pool)).Methods("GET")
	r.HandleFunc("/categories/{id}", handlers.UpdateCategoryHandler(pool)).Methods("PUT")
	r.HandleFunc("/categories/{id}", handlers.DeleteCategoryHandler(pool)).Methods("DELETE")

	return r
}
