package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email           string    `gorm:"uniqueIndex" json:"email"`
	Password        string    `json:"-"`                       // Никогда не отдаём в JSON
	UUID            string    `gorm:"uniqueIndex" json:"uuid"` // UUID для Xray
	TariffID        int       `json:"tariff_id"`               // ID тарифа
	CreatedAt       time.Time `json:"created_at"`
	IsBanned        bool      `json:"is_banned"`
	TelegramID      int64     `json:"telegram_id"`
	TariffExpiresAt time.Time `json:"tariff_expires_at"`
	UsedTraffic     int64     `json:"used_traffic"`
	Tariff          Tariff    // Add Tariff relation
}
