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

// CreateCategoryHandler создает новую категорию
func CreateCategoryHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var category models.Category
		if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		// Логирование данных для отладки
		log.Printf("Полученные данные категории: %+v\n", category)

		// Проверяем наличие обязательного поля UserID
		if category.UserID == 0 {
			http.Error(w, "UserID is required", http.StatusBadRequest)
			return
		}

		// Вызов функции для создания категории в базе данных
		if err := database.CreateCategory(pool, &category); err != nil {
			log.Printf("Ошибка при создании категории: %v", err)
			http.Error(w, "Failed to create category", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(category)
	}
}

// GetAllCategoriesHandler получает все категории
func GetAllCategoriesHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		categories, err := database.GetAllCategories(pool)
		if err != nil {
			log.Printf("Ошибка при получении категорий: %v", err)
			http.Error(w, "Failed to fetch categories", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(categories)
	}
}

// GetCategoryHandler получает категорию по ID
func GetCategoryHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, "Invalid category ID", http.StatusBadRequest)
			return
		}
		category, err := database.GetCategoryByID(pool, id)
		if err != nil {
			log.Printf("Ошибка при получении категории: %v", err)
			http.Error(w, "Category not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(category)
	}
}

// UpdateCategoryHandler обновляет существующую категорию
func UpdateCategoryHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, "Invalid category ID", http.StatusBadRequest)
			return
		}
		var category models.Category
		if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		category.ID = id
		if err := database.UpdateCategory(pool, &category); err != nil {
			log.Printf("Ошибка при обновлении категории: %v", err)
			http.Error(w, "Failed to update category", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "Category updated successfully"})
	}
}

// DeleteCategoryHandler удаляет категорию по ID
func DeleteCategoryHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, "Invalid category ID", http.StatusBadRequest)
			return
		}
		if err := database.DeleteCategory(pool, id); err != nil {
			log.Printf("Ошибка при удалении категории: %v", err)
			http.Error(w, "Failed to delete category", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "Category deleted successfully"})
	}
}
