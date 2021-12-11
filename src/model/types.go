package model

import "time"

type PriceInfo struct {
	Level    string    `json:"level"`
	Total    float64   `json:"total"`
	Energy   float64   `json:"energy"`
	Tax      float64   `json:"tax"`
	Currency string    `json:"currency"`
	StartsAt time.Time `json:"startsAt"`
}
