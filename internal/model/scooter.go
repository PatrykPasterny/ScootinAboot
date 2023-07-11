package model

import (
	"github.com/google/uuid"
)

type ScooterGet struct {
	UUID         uuid.UUID `json:"UUID"`
	Longitude    float64   `json:"longitude"`
	Latitude     float64   `json:"latitude"`
	Availability bool      `json:"availability"`
}

type ScooterPost struct {
	UUID         uuid.UUID `json:"UUID"`
	Longitude    float64   `json:"longitude"`
	Latitude     float64   `json:"latitude"`
	Availability bool      `json:"availability"`
	City         string    `json:"city"`
}
