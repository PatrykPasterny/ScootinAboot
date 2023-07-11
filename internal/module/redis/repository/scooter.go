package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const unitOfLength = "m" // in meters

var errNotFoundGivenScooter = errors.New("scooter with given UUID was not found")

type redisRepository struct {
	logger *log.Logger
	client *redis.Client
}

func NewRedisRepository(logger *log.Logger, client *redis.Client) *redisRepository {
	return &redisRepository{
		logger: logger,
		client: client,
	}
}

func (rr *redisRepository) GetScooters(long, lat, radius float64, city string) ([]redis.GeoLocation, error) {
	// Perform the GeoRadius search
	result, err := rr.client.GeoRadius(context.Background(), city, long, lat, &redis.GeoRadiusQuery{
		Radius: radius,
		Unit:   unitOfLength,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("getting scooters from redis using geo index: %w", err)
	}

	return result, err
}

func (rr *redisRepository) GetScooterLocation(scooterUUID uuid.UUID, city string) (*redis.GeoPos, error) {
	coords, err := rr.client.GeoPos(context.Background(), city, scooterUUID.String()).Result()
	if err != nil {
		return nil, fmt.Errorf("retrieving coordinates: %w", err)
	}

	if len(coords) == 0 {
		return nil, errNotFoundGivenScooter
	}

	return coords[0], nil
}

func (rr *redisRepository) GetScooterAvailability(scooterUUID uuid.UUID) (bool, error) {
	// Retrieve the scooter directly using its UUID
	scooterJSON, err := rr.client.Get(context.Background(), scooterUUID.String()).Result()
	if err != nil {
		return false, fmt.Errorf("getting scooters availability from redis: %w", err)
	}

	var availabilityAsInt int

	err = json.Unmarshal([]byte(scooterJSON), &availabilityAsInt)
	if err != nil {
		return false, fmt.Errorf("unmarshaling scooters availability: %w", err)
	}

	availability := availabilityAsInt == 1

	return availability, nil
}

func (rr *redisRepository) UpdateScooterLocation(scooter *redis.GeoLocation, city string) error {
	// Update the Geo index with scooter information
	if _, err := rr.client.GeoAdd(context.Background(), city, scooter).Result(); err != nil {
		return fmt.Errorf("adding scooter's location to redis: %w", err)
	}

	return nil
}

func (rr *redisRepository) UpdateScooterAvailability(scooterUUID uuid.UUID, availability bool) error {
	// Store the additional data as a string in Redis
	if err := rr.client.Set(context.Background(), scooterUUID.String(), availability, 0).Err(); err != nil {
		return fmt.Errorf("setting scooter's availability in redis: %w", err)
	}

	return nil
}
