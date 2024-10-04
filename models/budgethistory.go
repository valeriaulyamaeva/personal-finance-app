package models

import "time"

type BudgetsHistory struct {
	ID        int       `json:"id" db:"id"`
	BudgetID  int       `json:"budget_id" db:"budget_id"`
	OpDate    time.Time `json:"op_date" db:"op_date"`
	OldAmount float64   `json:"old_amount" db:"old_amount"`
	NewAmount float64   `json:"new_amount" db:"new_amount"`
	UserName  string    `json:"user_name" db:"user_name"`
}
