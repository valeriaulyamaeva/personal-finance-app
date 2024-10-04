package models

import "time"

type Report struct {
	ID            int       `json:"id" db:"id"`
	UserID        int       `json:"user_id" db:"user_id"`
	ReportData    string    `json:"report_data" db:"report_data"`
	GeneratedDate time.Time `json:"generated_date" db:"generated_date"`
}
