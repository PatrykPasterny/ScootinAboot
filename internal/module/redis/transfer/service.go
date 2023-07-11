package transfer

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"scootinAboot/internal/module/redis/model"
)

//go:generate mockgen -source=service.go -destination=mock/redis_service_mock.go -package=mock
type RedisService interface {
	GetScooters(longitude, latitude, radius float64, city string) ([]*model.RedisScooter, error)
	UpdateScooter(scooter *model.RedisScooter, city string) error
	UpdateScooterLocation(scooter *redis.GeoLocation, city string) error
	UpdateScooterAvailability(scooterUUID uuid.UUID, availability bool) error
}

type redisService struct {
	logger *log.Logger
	repo   RedisRepository
}

func NewRedisService(logger *log.Logger, repository RedisRepository) *redisService {
	return &redisService{
		logger: logger,
		repo:   repository,
	}
}

func (rs *redisService) GetScooters(longitude, latitude, radius float64, city string) ([]*model.RedisScooter, error) {
	scooters, err := rs.repo.GetScooters(longitude, latitude, radius, city)
	if err != nil {
		return nil, fmt.Errorf("getting scooters: %w", err)
	}

	results := make([]*model.RedisScooter, len(scooters))

	for i := range scooters {
		var scooterUUID uuid.UUID

		scooterUUID, err = uuid.Parse(scooters[i].Name)
		if err != nil {
			return nil, fmt.Errorf("parsing scooter's uuid: %w", err)
		}

		var coords *redis.GeoPos

		coords, err = rs.repo.GetScooterLocation(scooterUUID, city)
		if err != nil {
			return nil, fmt.Errorf("getting scooter's coords: %w", err)
		}

		var availability bool

		availability, err = rs.repo.GetScooterAvailability(scooterUUID)
		if err != nil {
			return nil, fmt.Errorf("getting scooter's availability: %w", err)
		}

		result := model.NewRedisScooter(&scooters[i], coords, availability)
		results[i] = result
	}

	return results, nil
}

func (rs *redisService) UpdateScooter(scooter *model.RedisScooter, city string) error {
	err := rs.repo.UpdateScooterLocation(scooter.Scooter, city)
	if err != nil {
		return fmt.Errorf("updating scooter's location: %w", err)
	}

	scooterUUID, err := uuid.Parse(scooter.Scooter.Name)
	if err != nil {
		return fmt.Errorf("parsing scooter's uuid: %w", err)
	}

	err = rs.repo.UpdateScooterAvailability(scooterUUID, scooter.Availability)
	if err != nil {
		return fmt.Errorf("updating scooter's availability: %w", err)
	}

	return nil
}

func (rs *redisService) UpdateScooterLocation(location *redis.GeoLocation, city string) error {
	err := rs.repo.UpdateScooterLocation(location, city)
	if err != nil {
		return fmt.Errorf("updating scooter's location: %w", err)
	}

	return nil
}

func (rs *redisService) UpdateScooterAvailability(scooterUUID uuid.UUID, availability bool) error {
	err := rs.repo.UpdateScooterAvailability(scooterUUID, availability)
	if err != nil {
		return fmt.Errorf("updating scooter's availability: %w", err)
	}

	return nil
}
