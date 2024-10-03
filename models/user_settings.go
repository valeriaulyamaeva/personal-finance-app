package models

type UserSettings struct {
	ID               int  `json:"id" db:"id"`
	UserID           int  `json:"user_id" db:"user_id"`
	TwoFactorEnabled bool `json:"two_factor_enabled" db:"two_factor_enabled"`
}
