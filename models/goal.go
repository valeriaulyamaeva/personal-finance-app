package models

import "time"

type Goal struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Amount    float64   `json:"amount" db:"amount"`
	Current   float64   `json:"current" db:"current"`
	Deadline  time.Time `json:"deadline" db:"deadline"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
