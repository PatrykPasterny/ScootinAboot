//go:build unit

package transfer

import (
	"errors"
	"log"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	redisservicemock "scootinAboot/internal/module/redis/transfer/mock"
	"scootinAboot/internal/module/rental/model"
	trackermodel "scootinAboot/internal/module/tracker/model"
	trackermock "scootinAboot/internal/module/tracker/transfer/mock"
)

const (
	testLongitude = 70
	testLatitude  = 60
	testCity      = "Montreal"
)

func TestRent(t *testing.T) {
	logger := log.New(os.Stdout, "TEST ", log.LstdFlags)

	firstScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scooter := model.NewRentalScooter(
		&redis.GeoLocation{
			Name:      firstScooterUUID.String(),
			Longitude: testLongitude,
			Latitude:  testLatitude,
		},
		testCity,
		true)

	wrongUUIDScooter := model.NewRentalScooter(
		&redis.GeoLocation{
			Name: "dd-dd-dd",
		},
		testCity,
		false,
	)

	tests := map[string]struct {
		logger                     *log.Logger
		scooter                    *model.RentalScooter
		mockRedisServiceHandler    func(mock *redisservicemock.MockRedisService)
		mockTrackingServiceHandler func(mock *trackermock.MockTrackerService)
		wantErr                    bool
	}{
		"successfully rent scooter": {
			logger:  logger,
			scooter: scooter,
			mockRedisServiceHandler: func(mock *redisservicemock.MockRedisService) {
				scooterUUID, innerErr := uuid.Parse(scooter.Name)
				require.NoError(t, innerErr)

				mock.EXPECT().UpdateScooterAvailability(scooterUUID, false).Return(nil).Times(1)
			},
			mockTrackingServiceHandler: func(mock *trackermock.MockTrackerService) {
				trackerScooter := &trackermodel.TrackerScooter{
					GeoLocation: scooter.GeoLocation,
					City:        scooter.City,
				}

				scooterUUID, innerErr := uuid.Parse(trackerScooter.Name)
				require.NoError(t, innerErr)

				mock.EXPECT().TrackScooter(scooterUUID, trackerScooter).Return(nil).Times(1)
			},
			wantErr: false,
		},
		"rent scooter failing because chosen scooter has incorrect UUID": {
			logger:                     logger,
			scooter:                    wrongUUIDScooter,
			mockRedisServiceHandler:    nil,
			mockTrackingServiceHandler: nil,
			wantErr:                    true,
		},
		"rent scooter failing because redis service threw an error": {
			logger:  logger,
			scooter: scooter,
			mockRedisServiceHandler: func(mock *redisservicemock.MockRedisService) {
				scooterUUID, innerErr := uuid.Parse(scooter.Name)
				require.NoError(t, innerErr)

				mock.EXPECT().UpdateScooterAvailability(scooterUUID, false).Return(redis.ErrClosed)
			},
			mockTrackingServiceHandler: nil,
			wantErr:                    true,
		},
		"rent scooter failing because tracking service threw an error": {
			logger:  logger,
			scooter: scooter,
			mockRedisServiceHandler: func(mock *redisservicemock.MockRedisService) {
				scooterUUID, innerErr := uuid.Parse(scooter.Name)
				require.NoError(t, innerErr)

				mock.EXPECT().UpdateScooterAvailability(scooterUUID, false).Return(nil).Times(1)
			},
			mockTrackingServiceHandler: func(mock *trackermock.MockTrackerService) {
				trackerScooter := &trackermodel.TrackerScooter{
					GeoLocation: scooter.GeoLocation,
					City:        scooter.City,
				}

				scooterUUID, innerErr := uuid.Parse(trackerScooter.Name)
				require.NoError(t, innerErr)

				mock.EXPECT().TrackScooter(scooterUUID, trackerScooter).Return(errors.New("")).Times(1)
			},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockRedisService := redisservicemock.NewMockRedisService(controller)
			mockTrackingService := trackermock.NewMockTrackerService(controller)

			if tt.mockRedisServiceHandler != nil {
				tt.mockRedisServiceHandler(mockRedisService)
			}

			if tt.mockTrackingServiceHandler != nil {
				tt.mockTrackingServiceHandler(mockTrackingService)
			}

			rs := NewRentalService(logger, mockRedisService, mockTrackingService)
			if err = rs.Rent(tt.scooter); (err != nil) != tt.wantErr {
				t.Errorf("Rent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFree(t *testing.T) {
	logger := log.New(os.Stdout, "TEST ", log.LstdFlags)

	firstScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	tests := map[string]struct {
		logger                     *log.Logger
		mockRedisServiceHandler    func(mock *redisservicemock.MockRedisService)
		mockTrackingServiceHandler func(mock *trackermock.MockTrackerService)
		wantErr                    bool
	}{
		"successfully freed scooter": {
			logger: logger,
			mockRedisServiceHandler: func(mock *redisservicemock.MockRedisService) {
				mock.EXPECT().UpdateScooterAvailability(firstScooterUUID, true).Return(nil).Times(1)
			},
			mockTrackingServiceHandler: func(mock *trackermock.MockTrackerService) {
				mock.EXPECT().FreeScooter(firstScooterUUID).Return(nil).Times(1)
			},
			wantErr: false,
		},
		"freeing scooter failed because redis service threw an error when updating availability": {
			logger: logger,
			mockRedisServiceHandler: func(mock *redisservicemock.MockRedisService) {
				mock.EXPECT().UpdateScooterAvailability(firstScooterUUID, true).Return(redis.ErrClosed).Times(1)
			},
			mockTrackingServiceHandler: func(mock *trackermock.MockTrackerService) {
				mock.EXPECT().FreeScooter(firstScooterUUID).Return(nil).Times(1)
			},
			wantErr: true,
		},
		"freeing scooter failed because tracking service threw an error when freeing scooter": {
			logger:                  logger,
			mockRedisServiceHandler: nil,
			mockTrackingServiceHandler: func(mock *trackermock.MockTrackerService) {
				mock.EXPECT().FreeScooter(firstScooterUUID).Return(errors.New("")).Times(1)
			},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockRedisService := redisservicemock.NewMockRedisService(controller)
			mockTrackingService := trackermock.NewMockTrackerService(controller)

			if tt.mockRedisServiceHandler != nil {
				tt.mockRedisServiceHandler(mockRedisService)
			}

			if tt.mockTrackingServiceHandler != nil {
				tt.mockTrackingServiceHandler(mockTrackingService)
			}

			rs := NewRentalService(logger, mockRedisService, mockTrackingService)
			if err = rs.Free(firstScooterUUID); (err != nil) != tt.wantErr {
				t.Errorf("Free() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
