package transfer

import (
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

//go:generate mockgen -source=redis_repository.go -destination=mock/redis_repository_mock.go -package=mock
type RedisRepository interface {
	GetScooters(longitude, latitude, radius float64, city string) ([]redis.GeoLocation, error)
	GetScooterAvailability(scooterUUID uuid.UUID) (bool, error)
	GetScooterLocation(scooterUUID uuid.UUID, city string) (*redis.GeoPos, error)
	UpdateScooterLocation(scooter *redis.GeoLocation, city string) error
	UpdateScooterAvailability(scooterUUID uuid.UUID, availability bool) error
}
