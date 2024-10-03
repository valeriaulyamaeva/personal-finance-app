package models

import "time"

type Transaction struct {
	ID         int       `json:"id" db:"id"`
	UserID     int       `json:"user_id" db:"user_id"`
	CategoryID int       `json:"category_id" db:"category_id"`
	Amount     float64   `json:"amount" db:"amount"`
	Date       time.Time `json:"date" db:"date"`
	Note       string    `json:"note" db:"note"`
}
