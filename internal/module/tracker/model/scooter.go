package model

import "github.com/redis/go-redis/v9"

type TrackerScooter struct {
	*redis.GeoLocation
	City string
}

func NewTrackerScooter(location *redis.GeoLocation, city string) *TrackerScooter {
	return &TrackerScooter{
		GeoLocation: location,
		City:        city,
	}
}
