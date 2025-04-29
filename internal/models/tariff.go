package models

import "gorm.io/gorm"

type Tariff struct {
	gorm.Model
	Name         string
	Description  string
	Price        float64
	TrafficLimit int64 // in bytes
}
