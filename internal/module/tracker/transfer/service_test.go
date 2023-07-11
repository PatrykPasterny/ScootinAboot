//go:build unit

package transfer

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"scootinAboot/internal/module/redis/transfer/mock"
	"scootinAboot/internal/module/tracker/model"
)

const (
	amountOfScooterTrackingEvents = 2
	firstTestCity                 = "Montreal"
	secondTestCity                = "Ottawa"
)

func TestTrackScooter(t *testing.T) {
	logger := log.New(os.Stdout, "TEST ", log.LstdFlags)

	firstScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	thirdScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scooters := []*model.TrackerScooter{
		{
			GeoLocation: &redis.GeoLocation{
				Name:      firstScooterUUID.String(),
				Longitude: 70.01,
				Latitude:  60.01,
			},
			City: firstTestCity,
		},
		{
			GeoLocation: &redis.GeoLocation{
				Name:      thirdScooterUUID.String(),
				Longitude: 69.99,
				Latitude:  59.99,
			},
			City: secondTestCity,
		},
	}

	tests := map[string]struct {
		logger                  *log.Logger
		mockRedisServiceHandler func(mock *mock.MockRedisService)
		wantErr                 bool
	}{
		"successfully tracking multiple scooters": {
			logger: logger,
			mockRedisServiceHandler: func(mock *mock.MockRedisService) {
				for i := range scooters {
					mock.EXPECT().UpdateScooterLocation(gomock.Any(), scooters[i].City).
						Return(nil).Times(amountOfScooterTrackingEvents)
				}
			},
			wantErr: false,
		},
		"failed tracking multiple scooters, because of redis service threw error when updating scooter location ": {
			logger: logger,
			mockRedisServiceHandler: func(mock *mock.MockRedisService) {
				mock.EXPECT().UpdateScooterLocation(gomock.Any(), scooters[0].City).
					Return(nil).Times(amountOfScooterTrackingEvents - 1)
				mock.EXPECT().UpdateScooterLocation(gomock.Any(), scooters[0].City).
					Return(redis.ErrClosed).Times(1)
				for i := 1; i < len(scooters); i++ {
					mock.EXPECT().UpdateScooterLocation(gomock.Any(), scooters[i].City).
						Return(nil).Times(amountOfScooterTrackingEvents)
				}
			},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockRedisService := mock.NewMockRedisService(controller)

			tt.mockRedisServiceHandler(mockRedisService)

			ts := NewTrackingService(tt.logger, mockRedisService)

			for i := range scooters {
				scooterUUID, innerErr := uuid.Parse(scooters[i].Name)
				require.NoError(t, innerErr)

				innerErr = ts.TrackScooter(scooterUUID, scooters[i])
				require.NoError(t, innerErr)
			}

			if len(ts.rentedScooters) != len(scooters) {
				t.Errorf(
					"TrackScooter() should rent all given scooters = %v, want %v",
					len(ts.rentedScooters),
					len(scooters),
				)
			}

			for i := 0; i < amountOfScooterTrackingEvents; i++ {
				time.Sleep(MovingTimeInSeconds * time.Second)
			}

			for i := range scooters {
				scooterUUID, err := uuid.Parse(scooters[i].Name)
				require.NoError(t, err)

				ts.rentedScooters[scooterUUID] <- scooterUUID
			}

			var errorFound bool

			for i := range ts.errorsChan {
				if err = <-ts.errorsChan[i]; err != nil {
					errorFound = true
				}

				close(ts.errorsChan[i])
			}

			if errorFound != tt.wantErr {
				t.Errorf("TrackScooter() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
		})
	}
}

func TestFreeScooter(t *testing.T) {
	logger := log.New(os.Stdout, "TEST ", log.LstdFlags)

	firstScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scooter := &model.TrackerScooter{
		GeoLocation: &redis.GeoLocation{
			Name:      firstScooterUUID.String(),
			Longitude: 70.01,
			Latitude:  60.01,
		},
		City: firstTestCity,
	}

	tests := map[string]struct {
		logger                  *log.Logger
		mockRedisServiceHandler func(mock *mock.MockRedisService)
		rentScooterHandler      func(tracker *trackingService)
		wantErr                 bool
	}{
		"successfully freeing scooter": {
			logger:                  logger,
			mockRedisServiceHandler: nil,
			rentScooterHandler: func(ts *trackingService) {
				ts.TrackScooter(firstScooterUUID, scooter)
			},
			wantErr: false,
		},
		"freeing scooter failed, because freed scooter have not been rented": {
			logger:                  logger,
			mockRedisServiceHandler: nil,
			rentScooterHandler:      func(ts *trackingService) {},
			wantErr:                 true,
		},
		"freeing scooter failed, because scooter's rental process threw error": {
			logger: logger,
			mockRedisServiceHandler: func(mock *mock.MockRedisService) {
				mock.EXPECT().UpdateScooterLocation(scooter.GeoLocation, scooter.City).Return(redis.ErrClosed)
			},
			rentScooterHandler: func(ts *trackingService) {
				ts.TrackScooter(firstScooterUUID, scooter)

				time.Sleep(MovingTimeInSeconds * time.Second)
			},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockRedisService := mock.NewMockRedisService(controller)
			if tt.mockRedisServiceHandler != nil {
				tt.mockRedisServiceHandler(mockRedisService)
			}

			ts := NewTrackingService(tt.logger, mockRedisService)

			tt.rentScooterHandler(ts)

			if err = ts.FreeScooter(firstScooterUUID); (err != nil) != tt.wantErr {
				t.Errorf("FreeScooter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
