package models

import "time"

type PaymentReminder struct {
	ID          int       `json:"id" db:"id"`
	UserID      int       `json:"user_id" db:"user_id"`
	Description string    `json:"description" db:"description"`
	Amount      float64   `json:"amount" db:"amount"`
	DueDate     time.Time `json:"due_date" db:"due_date"`
	Note        string    `json:"note" db:"note"`
}
