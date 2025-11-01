package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email                    string    `gorm:"uniqueIndex" json:"email"`
	Password                 string    `json:"-"`
	UUID                     string    `gorm:"uniqueIndex" json:"uuid"`
	TariffID                 int       `json:"tariff_id"`
	CreatedAt                time.Time `json:"created_at"`
	IsBanned                 bool      `json:"is_banned"`
	TelegramID               int64     `json:"telegram_id"`
	TariffExpiresAt          time.Time `json:"tariff_expires_at"`
	UsedTraffic              int64     `json:"used_traffic"`
	PasswordResetToken       string    `json:"-"`
	PasswordResetExpiresAt   time.Time `json:"-"`
	PasswordResetRequestedAt time.Time `json:"-"`
	TrialEndsAt              time.Time `json:"trial_ends_at"`
	// Refresh token hash (SHA256 hex) for issued refresh token
	RefreshTokenHash      string    `json:"-"`
	RefreshTokenExpiresAt time.Time `json:"-"`
	Tariff                Tariff
}
