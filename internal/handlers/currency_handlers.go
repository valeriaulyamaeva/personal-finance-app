package handlers

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	_ "github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type CurrencyResponse struct {
	Rates map[string]float64 `json:"rates"`
}

type CurrencyRate struct {
	CurAbbreviation string  `json:"Cur_Abbreviation"`
	CurScale        int     `json:"Cur_Scale"`
	Rate            float64 `json:"Cur_OfficialRate"`
}

// Кэш курсов валют
var (
	cachedRates  = make(map[string]float64)
	cacheMutex   = sync.Mutex{}
	lastFetch    time.Time
	cacheTimeout = 1 * time.Hour
	apiURL       = "https://www.nbrb.by/api/exrates/rates?periodicity=0"
)

// Получение курсов валют с кэшированием
func GetExchangeRatesHandler(w http.ResponseWriter, r *http.Request) {
	rates, err := getCachedRates()
	if err != nil {
		http.Error(w, "Ошибка получения курсов валют", http.StatusInternalServerError)
		log.Printf("Ошибка: %v", err)
		return
	}
	sendResponse(w, rates)
}

func getCachedRates() (map[string]float64, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if time.Since(lastFetch) < cacheTimeout {
		return cachedRates, nil
	}

	rates, err := fetchExchangeRates()
	if err != nil {
		return nil, err
	}

	cachedRates = rates
	lastFetch = time.Now()
	return rates, nil
}

// Запрос данных о валюте через API
func fetchExchangeRates() (map[string]float64, error) {
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("API вернул ошибку")
	}

	var rates []CurrencyRate
	if err := json.NewDecoder(resp.Body).Decode(&rates); err != nil {
		return nil, err
	}

	result := make(map[string]float64)
	for _, rate := range rates {
		result[rate.CurAbbreviation] = rate.Rate / float64(rate.CurScale)
	}
	return result, nil
}

// Обработка запроса на конвертацию валют
func ConvertCurrencyHandler(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	amountStr := r.URL.Query().Get("amount")

	if from == "" || to == "" || amountStr == "" {
		http.Error(w, "Отсутствуют параметры 'from', 'to' или 'amount'", http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		http.Error(w, "Неверное значение 'amount'", http.StatusBadRequest)
		return
	}

	rates, err := getCachedRates()
	if err != nil {
		http.Error(w, "Ошибка получения курсов валют", http.StatusInternalServerError)
		return
	}

	fromRate, fromExists := rates[from]
	toRate, toExists := rates[to]
	if !fromExists || !toExists {
		http.Error(w, "Валюта не найдена", http.StatusNotFound)
		return
	}

	result := amount * (toRate / fromRate)
	response := map[string]interface{}{
		"from":   from,
		"to":     to,
		"amount": amount,
		"result": result,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Утилиты
func sendResponse(w http.ResponseWriter, rates map[string]float64) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CurrencyResponse{Rates: rates})
}

// Обработка запроса на получение курса конкретной валюты
func GetCurrencyRateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r) // Извлекаем параметры из URL
	code := vars["code"]

	rates, err := getCachedRates()
	if err != nil {
		http.Error(w, "Ошибка получения курсов валют", http.StatusInternalServerError)
		return
	}

	// Проверяем, существует ли курс для данной валюты
	rate, exists := rates[code]
	if !exists {
		http.Error(w, "Валюта не найдена", http.StatusNotFound)
		return
	}

	// Формируем ответ с курсом выбранной валюты
	response := map[string]interface{}{
		"currency": code,
		"rate":     rate,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
