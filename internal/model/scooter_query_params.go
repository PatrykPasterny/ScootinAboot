package model

type ScooterQueryParams struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Radius    float64 `json:"radius"`
	City      string  `json:"city"`
}
