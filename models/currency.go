package models

type Currency struct {
	ID     int    `json:"id" db:"id"`
	Name   string `json:"name" db:"name"`
	Symbol string `json:"symbol" db:"symbol"`
}
