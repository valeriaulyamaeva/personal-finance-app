package models

import "time"

type Budget struct {
	ID         int       `json:"id" db:"id"`
	UserID     int       `json:"user_id" db:"user_id"`
	CategoryID int       `json:"category_id" db:"category_id"`
	Amount     float64   `json:"amount" db:"amount"`
	Period     string    `json:"period" db:"period"`
	StartDate  time.Time `json:"start_date" db:"start_date"`
	EndDate    time.Time `json:"end_date" db:"end_date"`
}
