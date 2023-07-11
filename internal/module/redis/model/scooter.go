package model

import "github.com/redis/go-redis/v9"

type RedisScooter struct {
	Scooter      *redis.GeoLocation
	Availability bool
}

func NewRedisScooter(location *redis.GeoLocation, coords *redis.GeoPos, availability bool) *RedisScooter {
	return &RedisScooter{
		Scooter: &redis.GeoLocation{
			Name:      location.Name,
			Longitude: coords.Longitude,
			Latitude:  coords.Latitude,
		},
		Availability: availability,
	}
}
