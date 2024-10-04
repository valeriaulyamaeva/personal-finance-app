package models

import "time"

type ExchangeRate struct {
	ID             int       `json:"id" db:"id"`
	FromCurrencyID int       `json:"from_currency_id" db:"from_currency_id"`
	ToCurrencyID   int       `json:"to_currency_id" db:"to_currency_id"`
	Rate           float64   `json:"rate" db:"rate"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
