package utils

import (
	"context"
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
	cachedRates  = sync.Map{}
	lastFetch    time.Time
	cacheTimeout = 1 * time.Hour
	apiURL       = "https://v6.exchangerate-api.com/v6/e8c2f4afec9e1abf33fd661d/latest/"
)

func GetCurrencyRate(currencyCode string) (float64, error) {
	// Check if rate is in cache and it's still valid
	if rate, ok := cachedRates.Load(currencyCode); ok {
		if time.Since(lastFetch) < cacheTimeout {
			log.Printf("Using cached rate for currency: %s", currencyCode)
			return rate.(CurrencyRate).Rate, nil
		}
	}

	if time.Since(lastFetch) >= cacheTimeout {
		if err := fetchExchangeRates(); err != nil {
			log.Printf("Failed to fetch exchange rates: %v", err)
			// Use cached data if available and still valid
			if rate, ok := cachedRates.Load(currencyCode); ok {
				log.Printf("Using cached rate for currency: %s (after failed fetch)", currencyCode)
				return rate.(CurrencyRate).Rate, nil
			}
			return 0, err
		}
	}

	// Retry fetching the currency rate from cache
	if rate, ok := cachedRates.Load(currencyCode); ok {
		return rate.(CurrencyRate).Rate, nil
	}

	return 0, errors.New("currency not found")
}

func fetchExchangeRates() error {
	client := http.Client{Timeout: 10 * time.Second}
	url := apiURL + "USD" // Base currency is set to USD for better compatibility

	var lastErr error
	for i := 0; i < 3; i++ {
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			log.Printf("Error fetching rates (attempt %d): %v", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = errors.New("API returned non-OK status")
			log.Printf("API returned non-OK status: %d (attempt %d)", resp.StatusCode, i+1)
			time.Sleep(2 * time.Second)
			continue
		}

		var response struct {
			ConversionRates map[string]float64 `json:"conversion_rates"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			lastErr = err
			log.Printf("Error decoding API response (attempt %d): %v", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Only update the cache if valid data is found
		if len(response.ConversionRates) > 0 {
			for code, rate := range response.ConversionRates {
				if rate > 0 {
					cachedRates.Store(code, CurrencyRate{Code: code, Rate: rate})
				} else {
					log.Printf("Invalid rate for currency: %s", code)
				}
			}
			lastFetch = time.Now()
			log.Println("Exchange rates cache updated successfully")
			return nil
		}

		lastErr = errors.New("no valid data to update cache")
		log.Println(lastErr)
		time.Sleep(2 * time.Second)
	}

	return lastErr
}

func ConvertCurrency(amount float64, fromCurrency, toCurrency string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	fromRateChan := make(chan float64)
	toRateChan := make(chan float64)
	errChan := make(chan error, 2)

	// Fetch rates for 'from' and 'to' currency concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		fromRate, err := GetCurrencyRate(fromCurrency)
		if err != nil {
			errChan <- err
			return
		}
		fromRateChan <- fromRate
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		toRate, err := GetCurrencyRate(toCurrency)
		if err != nil {
			errChan <- err
			return
		}
		toRateChan <- toRate
	}()

	wg.Wait()

	var fromRate, toRate float64
	select {
	case fromRate = <-fromRateChan:
	case err := <-errChan:
		return 0, err
	case <-ctx.Done():
		return 0, errors.New("timeout reached while fetching rates")
	}

	select {
	case toRate = <-toRateChan:
	case err := <-errChan:
		return 0, err
	case <-ctx.Done():
		return 0, errors.New("timeout reached while fetching rates")
	}

	// Validate rates are valid
	if fromRate == 0 || toRate == 0 {
		return 0, errors.New("invalid currency rates")
	}

	// Convert amount based on the rates
	convertedAmount := amount * (toRate / fromRate)
	return convertedAmount, nil
}
