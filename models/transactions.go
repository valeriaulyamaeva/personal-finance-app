package models

import "time"

type Transaction struct {
	ID          int       `json:"id" db:"id"`
	UserID      int       `json:"user_id" db:"user_id"`
	CategoryID  int       `json:"category_id" db:"category_id"`
	Amount      float64   `json:"amount" db:"amount"`
	Date        time.Time `json:"date" db:"date"`
	Type        string    `json:"type" db:"type"` // Возможные значения: "income", "expense", "goal"
	Description string    `json:"description" db:"description"`
	GoalID      *int      `json:"goal_id,omitempty" db:"goal_id"` // Привязка к цели
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	Currency    string    `json:"currency" db:"currency"`
}
