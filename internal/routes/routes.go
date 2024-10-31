package routes

import (
	"github.com/gorilla/mux"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/handlers"
)

func SetupRouter() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/api/users", handlers.CreateUserHandler).Methods("POST")

	// Здесь можно добавить другие маршруты...

	return r
}
