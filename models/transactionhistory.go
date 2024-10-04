package models

import "time"

type TransactionHistory struct {
	ID            int       `json:"id" db:"id"`
	TransactionID int       `json:"transaction_id" db:"transaction_id"`
	OpDate        time.Time `json:"op_date" db:"op_date"`
	OpType        string    `json:"op_type" db:"op_type"`
	OldValue      float64   `json:"old_value" db:"old_value"`
	NewValue      float64   `json:"new_value" db:"new_value"`
	UserName      string    `json:"user_name" db:"user_name"`
}
