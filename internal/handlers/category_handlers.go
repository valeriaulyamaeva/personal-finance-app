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

func CreateCategoryHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var category models.Category
		if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
			http.Error(w, "Некорректный ввод", http.StatusBadRequest)
			return
		}

		log.Printf("Полученные данные категории: %+v\n", category)

		if category.UserID == 0 {
			http.Error(w, "Не указан UserID", http.StatusBadRequest)
			return
		}

		if err := database.CreateCategory(pool, &category); err != nil {
			log.Printf("Ошибка при создании категории: %v", err)
			http.Error(w, "Не удалось создать категорию", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(category)
	}
}

func GetAllCategoriesHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		categories, err := database.GetAllCategories(pool)
		if err != nil {
			log.Printf("Ошибка при получении категорий: %v", err)
			http.Error(w, "Не удалось получить категории", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(categories)
	}
}

func GetCategoryHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, "Некорректный ID категории", http.StatusBadRequest)
			return
		}
		category, err := database.GetCategoryByID(pool, id)
		if err != nil {
			log.Printf("Ошибка при получении категории: %v", err)
			http.Error(w, "Категория не найдена", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(category)
	}
}

func UpdateCategoryHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, "Некорректный ID категории", http.StatusBadRequest)
			return
		}
		var category models.Category
		if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
			http.Error(w, "Некорректный ввод", http.StatusBadRequest)
			return
		}
		category.ID = id
		if err := database.UpdateCategory(pool, &category); err != nil {
			log.Printf("Ошибка при обновлении категории: %v", err)
			http.Error(w, "Не удалось обновить категорию", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "Категория успешно обновлена"})
	}
}

func DeleteCategoryHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, "Некорректный ID категории", http.StatusBadRequest)
			return
		}
		if err := database.DeleteCategory(pool, id); err != nil {
			log.Printf("Ошибка при удалении категории: %v", err)
			http.Error(w, "Не удалось удалить категорию", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "Категория успешно удалена"})
	}
}
