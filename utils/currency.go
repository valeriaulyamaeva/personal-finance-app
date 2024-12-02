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
	Code string  `json:"currency"`
	Rate float64 `json:"rate"`
}

var (
	cachedRates  = make(map[string]CurrencyRate)
	cacheMutex   = sync.Mutex{}
	lastFetch    time.Time
	cacheTimeout = 1 * time.Hour
	apiURL       = "https://openexchangerates.org/api/latest.json"
	apiKey       = "YOUR_API_KEY" // Ваш API ключ
)

// GetCurrencyRate fetches and caches currency rates from Open Exchange Rates API
func GetCurrencyRate(currencyCode string) (float64, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Use cached data if it's still valid
	if time.Since(lastFetch) < cacheTimeout {
		if rate, ok := cachedRates[currencyCode]; ok {
			log.Printf("Using cached rate for currency: %s", currencyCode)
			return rate.Rate, nil
		}
		log.Printf("Currency %s not found in cache, refreshing rates", currencyCode)
	}

	// Fetch and update the cache
	if err := fetchExchangeRates(); err != nil {
		log.Printf("Failed to fetch exchange rates: %v", err)
		return 0, err
	}

	rate, ok := cachedRates[currencyCode]
	if !ok {
		log.Printf("Currency %s not found after fetching rates", currencyCode)
		return 0, errors.New("currency not found")
	}

	return rate.Rate, nil
}

// fetchExchangeRates fetches rates from Open Exchange Rates API and updates the cache
func fetchExchangeRates() error {
	client := http.Client{Timeout: 10 * time.Second}
	url := apiURL + "?app_id=" + apiKey
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Error fetching rates from API: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("API returned non-OK status: %d", resp.StatusCode)
		return errors.New("API returned an error")
	}

	var response struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Error decoding API response: %v", err)
		return err
	}

	// Validate and update cache
	newCache := make(map[string]CurrencyRate)
	for code, rate := range response.Rates {
		if rate > 0 {
			newCache[code] = CurrencyRate{Code: code, Rate: rate}
		} else {
			log.Printf("Invalid rate for currency: %s", code)
		}
	}

	// Only update the cache if we have valid data
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
	// Fetch rates concurrently
	fromRateChan := make(chan float64)
	toRateChan := make(chan float64)
	errChan := make(chan error, 2)

	// Fetch rate for 'from' currency
	go func() {
		fromRate, err := GetCurrencyRate(fromCurrency)
		if err != nil {
			errChan <- err
			return
		}
		fromRateChan <- fromRate
	}()

	// Fetch rate for 'to' currency
	go func() {
		toRate, err := GetCurrencyRate(toCurrency)
		if err != nil {
			errChan <- err
			return
		}
		toRateChan <- toRate
	}()

	// Wait for both responses
	var fromRate, toRate float64
	select {
	case fromRate = <-fromRateChan:
	case err := <-errChan:
		return 0, err
	}

	select {
	case toRate = <-toRateChan:
	case err := <-errChan:
		return 0, err
	}

	// Convert the amount
	convertedAmount := amount * (toRate / fromRate)
	return convertedAmount, nil
}
