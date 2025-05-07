package models

import "time"

type Payment struct {
	ID            int       `gorm:"primaryKey" json:"id"`
	UserID        int       `json:"user_id"`
	Amount        int       `json:"amount"`
	TariffID      int       `json:"tariff_id"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}
