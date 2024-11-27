package models

type UserSettings struct {
	ID                 int    `json:"id" db:"id"`
	UserID             int    `json:"user_id" db:"user_id"`
	TwoFactorEnabled   bool   `json:"two_factor_enabled" db:"two_factor_enabled"`
	Theme              string `json:"theme" db:"theme"`
	NotificationVolume int    `json:"notification_volume" db:"notification_volume"`
	AutoUpdates        bool   `json:"auto_updates" db:"auto_updates"`
	WeeklyReports      bool   `json:"weekly_reports" db:"weekly_reports"`
}
