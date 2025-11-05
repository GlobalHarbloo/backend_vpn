package models

import "time"

// BotSession stores temporary dialog state for a Telegram chat
type BotSession struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	ChatID    int64     `gorm:"uniqueIndex" json:"chat_id"`
	Action    string    `json:"action"`
	Step      int       `json:"step"`
	Email     string    `json:"email"`
	UpdatedAt time.Time `json:"updated_at"`
}
