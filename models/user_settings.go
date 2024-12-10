package models

type UserSettings struct {
	ID                 int    `json:"id"`
	UserID             int    `json:"user_id"`
	Currency           string `json:"currency"`
	Theme              string `json:"theme"`
	NotificationVolume int    `json:"notification_volume"`
	AutoUpdates        bool   `json:"auto_updates"`
	WeeklyReports      bool   `json:"weekly_reports"`
}
