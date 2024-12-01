package models

type UserSettings struct {
	ID                 int    `json:"id"`
	UserID             int    `json:"user_id"`
	Currency           string `json:"currency"`           // Используем обычную строку
	OldCurrency        string `json:"old_currency"`       // Используем обычную строку
	Theme              string `json:"theme"`              // Используем обычную строку
	TwoFactorEnabled   bool   `json:"two_factor_enabled"` // Используем обычный bool
	NotificationVolume int    `json:"notification_volume"`
	AutoUpdates        bool   `json:"auto_updates"`
	WeeklyReports      bool   `json:"weekly_reports"`
}
