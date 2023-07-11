package transfer

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	commonRedis "scootinAboot/internal/module/redis/transfer"
	"scootinAboot/internal/module/tracker/model"
)

const (
	north = 1
	west  = 2
	east  = 3
	south = 4

	MovingTimeInSeconds = 3

	oneSecondDecimal float64 = 0.000278
)

var (
	myMux = sync.Mutex{}

	ErrRentAlreadyRentedScooter = errors.New("this scooter is already rented, choose another one")
	ErrNoScooterToFree          = errors.New("can't free scooter that have not been rented")
)

//go:generate mockgen -source=service.go -destination=mock/tracker_mock.go -package=mock
type TrackerService interface {
	TrackScooter(scooterUUID uuid.UUID, scooter *model.TrackerScooter) error
	FreeScooter(scooterUUID uuid.UUID) error
}

type trackingService struct {
	logger         *log.Logger
	service        commonRedis.RedisService
	rentedScooters map[uuid.UUID]chan uuid.UUID
	errorsChan     map[uuid.UUID]chan error
}

func NewTrackingService(logger *log.Logger, service commonRedis.RedisService) *trackingService {
	return &trackingService{
		logger:         logger,
		service:        service,
		rentedScooters: make(map[uuid.UUID]chan uuid.UUID),
		errorsChan:     make(map[uuid.UUID]chan error),
	}
}

func (ts *trackingService) TrackScooter(scooterUUID uuid.UUID, scooter *model.TrackerScooter) error {
	rentedScooterChan := make(chan uuid.UUID)
	rentalErrorsChan := make(chan error)

	myMux.Lock()

	if val, ok := ts.rentedScooters[scooterUUID]; ok {
		if val != nil {
			myMux.Unlock()

			return ErrRentAlreadyRentedScooter
		}
	}

	ts.rentedScooters[scooterUUID] = rentedScooterChan
	ts.errorsChan[scooterUUID] = rentalErrorsChan

	myMux.Unlock()

	go func() {
		defer close(rentedScooterChan)

		rentalErrors := make(map[string]int)
		for {
			select {
			case <-time.After(MovingTimeInSeconds * time.Second):
				simulateScooterMove(scooter.GeoLocation, MovingTimeInSeconds, north)

				ts.logger.Printf(
					"Scooter with UUID: %s continues his journey. Now it is at %f, %f.",
					scooterUUID,
					scooter.Longitude,
					scooter.Latitude,
				)

				err := ts.service.UpdateScooterLocation(scooter.GeoLocation, scooter.City)
				if err != nil {
					if _, ok := rentalErrors[err.Error()]; ok {
						rentalErrors[err.Error()] += 1
					}

					rentalErrors[err.Error()] = 1
				}
			case <-rentedScooterChan: // Signal to stop tracking
				if len(rentalErrors) == 0 {
					rentalErrorsChan <- nil

					return
				}

				routineErrors := fmt.Errorf(
					"go routine assigned to scooterUUID - %s met several errors",
					scooterUUID,
				)

				for key, value := range rentalErrors {
					routineErrors = fmt.Errorf("%w: %s, %d times", routineErrors, key, value)
				}

				rentalErrorsChan <- routineErrors

				return
			}
		}
	}()

	return nil
}

func (ts *trackingService) FreeScooter(scooterUUID uuid.UUID) error {
	myMux.Lock()

	scooterToFree, ok := ts.rentedScooters[scooterUUID]
	if !ok {
		return ErrNoScooterToFree
	}

	scooterToFree <- scooterUUID

	ts.rentedScooters[scooterUUID] = nil

	myMux.Unlock()

	potentialErrors := <-ts.errorsChan[scooterUUID]
	if potentialErrors != nil {
		return fmt.Errorf("freeing scooter: %w", potentialErrors)
	}

	close(ts.errorsChan[scooterUUID])

	return nil
}

// simulateScooterMove is simulating the move of the scooter, I assume that each scooter goes on average 36 km/h
// which is around one second degree per second(approximately for both latitude and longitude). I pick
// one of four sides(north, west, east, south) and move the scooter three second degrees in that direction.
func simulateScooterMove(scooter *redis.GeoLocation, timeInSeconds int, direction int) {
	if direction == north {
		scooter.Latitude += float64(timeInSeconds) * oneSecondDecimal
	} else if direction == south {
		scooter.Latitude -= float64(timeInSeconds) * oneSecondDecimal
	} else if direction == east {
		scooter.Longitude += float64(timeInSeconds) * oneSecondDecimal
	} else if direction == west {
		scooter.Longitude -= float64(timeInSeconds) * oneSecondDecimal
	}
}
