package models

type ExchangeRate struct {
	ID           int     `json:"id" db:"id"`
	FromCurrency string  `json:"from_currency" db:"from_currency"`
	ToCurrency   string  `json:"to_currency" db:"to_currency"`
	Rate         float64 `json:"rate" db:"rate"`
}
