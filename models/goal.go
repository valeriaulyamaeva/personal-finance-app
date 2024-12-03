package models

import (
	"fmt"
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
}

func (g *Goal) RemainingAmount() float64 {
	return g.Amount - g.CurrentAmount
}

// Обновляет статус цели, если она достигнута
func (g *Goal) UpdateGoalStatus() error {
	if g.CurrentAmount >= g.Amount {
		// Если цель достигнута, обновляем статус
		g.Status = "completed"
		return nil
	}
	return fmt.Errorf("цель не достигнута, еще необходимо %f", g.RemainingAmount())
}
