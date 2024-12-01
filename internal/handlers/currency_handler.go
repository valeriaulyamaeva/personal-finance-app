package handlers

import (
	"fmt"
	"github.com/goccy/go-json"
	"github.com/gorilla/mux"
	"github.com/valeriaulyamaeva/personal-finance-app/utils"
	"net/http"
	"strconv"
)

func ConvertCurrency(amount float64, fromCurrency, toCurrency string) (float64, error) {
	// Получаем курс для валюты из
	fromRate, err := utils.GetCurrencyRate(fromCurrency)
	if err != nil {
		return 0, fmt.Errorf("failed to get rate for currency %s: %v", fromCurrency, err)
	}

	// Получаем курс для валюты в
	toRate, err := utils.GetCurrencyRate(toCurrency)
	if err != nil {
		return 0, fmt.Errorf("failed to get rate for currency %s: %v", toCurrency, err)
	}

	// Конвертируем сумму
	convertedAmount := amount * (toRate / fromRate)
	return convertedAmount, nil
}

// Обработчик для конвертации валют
func conversionHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры запроса
	vars := mux.Vars(r)
	fromCurrency := vars["fromCurrency"]
	toCurrency := vars["toCurrency"]
	amount := r.URL.Query().Get("amount")

	// Преобразуем amount в число
	amountValue, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	// Выполняем конвертацию
	convertedAmount, err := ConvertCurrency(amountValue, fromCurrency, toCurrency)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	response := map[string]interface{}{
		"original_amount":  amountValue,
		"from_currency":    fromCurrency,
		"to_currency":      toCurrency,
		"converted_amount": convertedAmount,
	}

	// Преобразуем ответ в JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
