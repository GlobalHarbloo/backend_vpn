package models

import "time"

type TrafficLog struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	UsedMB    int64     `json:"used_mb"`   // Использовано мегабайт
	Timestamp time.Time `json:"timestamp"` // Время логирования
}
