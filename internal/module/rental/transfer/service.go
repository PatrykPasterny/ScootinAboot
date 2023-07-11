package transfer

import (
	"fmt"
	"log"

	"github.com/google/uuid"

	redis "scootinAboot/internal/module/redis/transfer"
	"scootinAboot/internal/module/rental/model"
	trackermodel "scootinAboot/internal/module/tracker/model"
	tracker "scootinAboot/internal/module/tracker/transfer"
)

//go:generate mockgen -source=service.go -destination=mock/rental_mock.go -package=mock
type RentalService interface {
	Rent(scooter *model.RentalScooter) error
	Free(scooterUUID uuid.UUID) error
}

type rentalService struct {
	logger          *log.Logger
	redisService    redis.RedisService
	trackingService tracker.TrackerService
}

func NewRentalService(logger *log.Logger, rService redis.RedisService, tracker tracker.TrackerService) *rentalService {
	return &rentalService{
		logger:          logger,
		redisService:    rService,
		trackingService: tracker,
	}
}

func (rs *rentalService) Rent(scooter *model.RentalScooter) error {
	scooterUUID, err := uuid.Parse(scooter.GeoLocation.Name)
	if err != nil {
		return fmt.Errorf("parsing scooter's uuid: %w", err)
	}

	err = rs.redisService.UpdateScooterAvailability(scooterUUID, false)
	if err != nil {
		return fmt.Errorf("updating scooter availability: %w", err)
	}

	rs.logger.Printf(
		"Scooter with UUID: %s started his journey from %f, %f.",
		scooterUUID,
		scooter.Longitude,
		scooter.Latitude,
	)

	trackingScooter := trackermodel.NewTrackerScooter(scooter.GeoLocation, scooter.City)

	if err = rs.trackingService.TrackScooter(scooterUUID, trackingScooter); err != nil {
		return fmt.Errorf("tracking scooter: %w", err)
	}

	return nil
}

func (rs *rentalService) Free(scooterUUID uuid.UUID) error {
	if err := rs.trackingService.FreeScooter(scooterUUID); err != nil {
		return fmt.Errorf("freeing scooter: %w", err)
	}

	rs.logger.Printf("Scooter with UUID: %s ended his journey.", scooterUUID)

	if err := rs.redisService.UpdateScooterAvailability(scooterUUID, true); err != nil {
		return fmt.Errorf("updating scooter availability: %w", err)
	}

	return nil
}
