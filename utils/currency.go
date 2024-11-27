package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

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
func GetCachedRates() (map[string]float64, error) {
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
