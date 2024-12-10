package models

import (
	"time"
)

type Goal struct {
	ID            int       `json:"id" db:"id"`
	UserID        int       `json:"user_id" db:"user_id"`
	Amount        float64   `json:"amount" db:"amount"`
	CurrentAmount float64   `json:"current_amount" db:"current_amount"`
	TargetDate    time.Time `json:"target_date" db:"target_date"`
	Name          string    `json:"name" db:"name"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	Status        string    `json:"status" db:"status"`
	Currency      string    `json:"currency" db:"currency"`
}

func (g *Goal) RemainingAmount() float64 {
	return g.Amount - g.CurrentAmount
}
