package models

import "time"

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`         // Никогда не отдаём в JSON
	UUID      string    `json:"uuid"`      // UUID для Xray
	TariffID  int       `json:"tariff_id"` // ID тарифа
	CreatedAt time.Time `json:"created_at"`
	IsBanned  bool      `json:"is_banned"`
}
