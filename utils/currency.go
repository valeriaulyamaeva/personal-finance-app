package utils

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"
)

type CurrencyRate struct {
	Code  string  `json:"Cur_Abbreviation"`
	Scale int     `json:"Cur_Scale"`
	Rate  float64 `json:"Cur_OfficialRate"`
}

var (
	cachedRates  = make(map[string]CurrencyRate)
	cacheMutex   = sync.Mutex{}
	lastFetch    time.Time
	cacheTimeout = 1 * time.Hour
	apiURL       = "https://www.nbrb.by/api/exrates/rates?periodicity=0"
)

// GetCurrencyRate fetches and caches currency rates from the National Bank API
func GetCurrencyRate(currencyCode string) (float64, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Use cached data if it's still valid
	if time.Since(lastFetch) < cacheTimeout {
		if rate, ok := cachedRates[currencyCode]; ok {
			log.Printf("Using cached rate for currency: %s", currencyCode)
			return rate.Rate / float64(rate.Scale), nil
		}
		log.Printf("Currency %s not found in cache, refreshing rates", currencyCode)
	}

	// Fetch and update the cache
	err := fetchExchangeRates()
	if err != nil {
		log.Printf("Failed to fetch exchange rates: %v", err)
		return 0, err
	}

	rate, ok := cachedRates[currencyCode]
	if !ok {
		log.Printf("Currency %s not found after fetching rates", currencyCode)
		return 0, errors.New("currency not found")
	}

	return rate.Rate / float64(rate.Scale), nil
}

// fetchExchangeRates fetches rates from the National Bank API and updates the cache
func fetchExchangeRates() error {
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		log.Printf("Error fetching rates from API: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("API returned non-OK status: %d", resp.StatusCode)
		return errors.New("API returned an error")
	}

	var rates []CurrencyRate
	if err := json.NewDecoder(resp.Body).Decode(&rates); err != nil {
		log.Printf("Error decoding API response: %v", err)
		return err
	}

	// Validate and update cache
	newCache := make(map[string]CurrencyRate)
	for _, rate := range rates {
		if rate.Code != "" && rate.Scale > 0 && rate.Rate > 0 {
			newCache[rate.Code] = rate
		} else {
			log.Printf("Invalid rate data: %+v", rate)
		}
	}

	if len(newCache) > 0 {
		cachedRates = newCache
		lastFetch = time.Now()
		log.Println("Exchange rates cache updated successfully")
	} else {
		log.Println("No valid data to update cache")
	}

	return nil
}

// ConvertCurrency converts an amount from one currency to another
func ConvertCurrency(amount float64, fromCurrency, toCurrency string) (float64, error) {
	// Get rate for the 'from' currency
	fromRate, err := GetCurrencyRate(fromCurrency)
	if err != nil {
		return 0, err
	}

	// Get rate for the 'to' currency
	toRate, err := GetCurrencyRate(toCurrency)
	if err != nil {
		return 0, err
	}

	// Convert the amount
	convertedAmount := amount * (toRate / fromRate)

	return convertedAmount, nil
}
