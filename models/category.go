package models

type Category struct {
	ID     int    `json:"id" db:"id"`
	Name   string `json:"name" db:"name"`
	UserID int    `json:"user_id" db:"user_id"`
}
