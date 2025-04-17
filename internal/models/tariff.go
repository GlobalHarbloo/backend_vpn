package models

type Tariff struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	TrafficGB   int     `json:"traffic_gb"` // Ограничение трафика в ГБ
	PriceUSD    float64 `json:"price_usd"`
}
