package model

import "github.com/redis/go-redis/v9"

type RentalScooter struct {
	*redis.GeoLocation
	City         string
	Availability bool
}

func NewRentalScooter(location *redis.GeoLocation, city string, availability bool) *RentalScooter {
	return &RentalScooter{
		GeoLocation:  location,
		City:         city,
		Availability: availability,
	}
}
