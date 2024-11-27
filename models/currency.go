package models

type Currency struct {
	ID           int     `json:"id" db:"id"`                       // Уникальный идентификатор
	Code         string  `json:"code" db:"code"`                   // Код валюты (USD, EUR, RUB)
	Name         string  `json:"name" db:"name"`                   // Название валюты
	ExchangeRate float64 `json:"exchange_rate" db:"exchange_rate"` // Курс валюты
	Scale        int     `json:"scale" db:"scale"`                 // Масштаб (например, 1 или 100)
	LastUpdated  string  `json:"last_updated" db:"last_updated"`   // Дата последнего обновления
}
