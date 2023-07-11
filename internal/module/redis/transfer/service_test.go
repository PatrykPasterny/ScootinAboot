//go:build unit

package transfer

import (
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"scootinAboot/internal/module/redis/model"
	"scootinAboot/internal/module/redis/transfer/mock"
)

const (
	testCity      = "Montreal"
	testLongitude = 70.0
	testLatitude  = 60.0
	testRadius    = 10000.0
)

func TestGetScooters(t *testing.T) {
	logger := log.New(os.Stdout, "TEST ", log.LstdFlags)

	firstScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	secScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	thirdScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scootersInRadius := []redis.GeoLocation{
		{
			Name: firstScooterUUID.String(),
		},
		{
			Name: secScooterUUID.String(),
		},
		{
			Name: thirdScooterUUID.String(),
		},
	}

	scootersLocations := []*redis.GeoPos{
		{
			Longitude: 60.0,
			Latitude:  40.0,
		},
		{
			Longitude: 60.001,
			Latitude:  40.02,
		},
		{
			Longitude: 60.12415,
			Latitude:  -40.0,
		},
	}

	scootersActivities := []bool{true, false, true}

	scooters := []*model.RedisScooter{
		model.NewRedisScooter(
			&redis.GeoLocation{
				Name: scootersInRadius[0].Name,
			},
			&redis.GeoPos{
				Longitude: scootersLocations[0].Longitude,
				Latitude:  scootersLocations[0].Latitude,
			},
			scootersActivities[0],
		),
		model.NewRedisScooter(
			&redis.GeoLocation{
				Name: scootersInRadius[1].Name,
			},
			&redis.GeoPos{
				Longitude: scootersLocations[1].Longitude,
				Latitude:  scootersLocations[1].Latitude,
			},
			scootersActivities[1],
		),
		model.NewRedisScooter(
			&redis.GeoLocation{
				Name: scootersInRadius[2].Name,
			},
			&redis.GeoPos{
				Longitude: scootersLocations[2].Longitude,
				Latitude:  scootersLocations[2].Latitude,
			},
			scootersActivities[2],
		),
	}

	tests := map[string]struct {
		logger                     *log.Logger
		mockRedisRepositoryHandler func(mock *mock.MockRedisRepository)
		want                       []*model.RedisScooter
		wantErr                    bool
	}{
		"getting scooters successfully": {
			logger: logger,
			mockRedisRepositoryHandler: func(mock *mock.MockRedisRepository) {
				mock.EXPECT().GetScooters(testLongitude, testLatitude, testRadius, testCity).
					Return(scootersInRadius, nil).Times(1)
				for i := range scootersInRadius {
					scooterUUID, innerErr := uuid.Parse(scootersInRadius[i].Name)
					require.NoError(t, innerErr)

					mock.EXPECT().GetScooterLocation(scooterUUID, testCity).
						Return(scootersLocations[i], nil).Times(1)

					mock.EXPECT().GetScooterAvailability(scooterUUID).Return(scootersActivities[i], nil).Times(1)
				}
			},
			want:    scooters,
			wantErr: false,
		},
		"getting scooters failed, because repository threw an error when getting scooters": {
			logger: logger,
			mockRedisRepositoryHandler: func(mock *mock.MockRedisRepository) {
				mock.EXPECT().GetScooters(testLongitude, testLatitude, testRadius, testCity).
					Return(nil, redis.ErrClosed).Times(1)
			},
			want:    nil,
			wantErr: true,
		},
		"getting scooters failed, because repository threw an error when getting scooter's location": {
			logger: logger,
			mockRedisRepositoryHandler: func(mock *mock.MockRedisRepository) {
				mock.EXPECT().GetScooters(testLongitude, testLatitude, testRadius, testCity).
					Return(scootersInRadius, nil).Times(1)

				scooterUUID, innerErr := uuid.Parse(scootersInRadius[0].Name)
				require.NoError(t, innerErr)

				mock.EXPECT().GetScooterLocation(scooterUUID, testCity).
					Return(nil, redis.ErrClosed).Times(1)
			},
			want:    nil,
			wantErr: true,
		},
		"getting scooters failed, because repository threw an error when getting scooter's availability": {
			logger: logger,
			mockRedisRepositoryHandler: func(mock *mock.MockRedisRepository) {
				mock.EXPECT().GetScooters(testLongitude, testLatitude, testRadius, testCity).
					Return(scootersInRadius, nil).Times(1)

				scooterUUID, innerErr := uuid.Parse(scootersInRadius[0].Name)
				require.NoError(t, innerErr)

				mock.EXPECT().GetScooterLocation(scooterUUID, testCity).
					Return(scootersLocations[0], nil).Times(1)

				mock.EXPECT().GetScooterAvailability(scooterUUID).Return(false, redis.ErrClosed).Times(1)
			},
			want:    nil,
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockRedisRepository := mock.NewMockRedisRepository(controller)

			tt.mockRedisRepositoryHandler(mockRedisRepository)

			rs := NewRedisService(tt.logger, mockRedisRepository)
			got, err := rs.GetScooters(testLongitude, testLatitude, testRadius, testCity)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetScooters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetScooters() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateScooter(t *testing.T) {
	logger := &log.Logger{}

	firstScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scooter := model.RedisScooter{
		Scooter: &redis.GeoLocation{
			Name:      firstScooterUUID.String(),
			Longitude: testLongitude,
			Latitude:  testLatitude,
		},
		Availability: true,
	}

	tests := map[string]struct {
		logger                     *log.Logger
		mockRedisRepositoryHandler func(mock *mock.MockRedisRepository)
		wantErr                    bool
	}{
		"updating scooter successfully": {
			logger: logger,
			mockRedisRepositoryHandler: func(mock *mock.MockRedisRepository) {
				mock.EXPECT().UpdateScooterLocation(scooter.Scooter, testCity).
					Return(nil).Times(1)

				scooterUUID, innerErr := uuid.Parse(scooter.Scooter.Name)
				require.NoError(t, innerErr)

				mock.EXPECT().UpdateScooterAvailability(scooterUUID, scooter.Availability).
					Return(nil).Times(1)
			},
			wantErr: false,
		},
		"updating scooter failed, because repository threw an error when updating scooter's location": {
			logger: logger,
			mockRedisRepositoryHandler: func(mock *mock.MockRedisRepository) {
				mock.EXPECT().UpdateScooterLocation(scooter.Scooter, testCity).
					Return(redis.ErrClosed).Times(1)
			},
			wantErr: true,
		},
		"updating scooter failed, because repository threw an error when updating scooter's availability": {
			logger: logger,
			mockRedisRepositoryHandler: func(mock *mock.MockRedisRepository) {
				mock.EXPECT().UpdateScooterLocation(scooter.Scooter, testCity).
					Return(nil).Times(1)

				scooterUUID, innerErr := uuid.Parse(scooter.Scooter.Name)
				require.NoError(t, innerErr)

				mock.EXPECT().UpdateScooterAvailability(scooterUUID, scooter.Availability).
					Return(redis.ErrClosed).Times(1)
			},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockRedisRepository := mock.NewMockRedisRepository(controller)

			tt.mockRedisRepositoryHandler(mockRedisRepository)

			rs := NewRedisService(logger, mockRedisRepository)

			if err := rs.UpdateScooter(&scooter, testCity); (err != nil) != tt.wantErr {
				t.Errorf("UpdateScooter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateScooterLocation(t *testing.T) {
	logger := &log.Logger{}

	firstScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scooterLocation := &redis.GeoLocation{
		Name:      firstScooterUUID.String(),
		Longitude: testLongitude,
		Latitude:  testLatitude,
	}

	tests := map[string]struct {
		logger                     *log.Logger
		mockRedisRepositoryHandler func(mock *mock.MockRedisRepository)
		wantErr                    bool
	}{
		"updating scooter successfully": {
			logger: logger,
			mockRedisRepositoryHandler: func(mock *mock.MockRedisRepository) {
				mock.EXPECT().UpdateScooterLocation(scooterLocation, testCity).
					Return(nil).Times(1)
			},
			wantErr: false,
		},
		"updating scooter failed, because repository threw an error when updating scooter's location": {
			logger: logger,
			mockRedisRepositoryHandler: func(mock *mock.MockRedisRepository) {
				mock.EXPECT().UpdateScooterLocation(scooterLocation, testCity).
					Return(redis.ErrClosed).Times(1)
			},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockRedisRepository := mock.NewMockRedisRepository(controller)

			tt.mockRedisRepositoryHandler(mockRedisRepository)

			rs := NewRedisService(logger, mockRedisRepository)

			if err := rs.UpdateScooterLocation(scooterLocation, testCity); (err != nil) != tt.wantErr {
				t.Errorf("UpdateScooterLocation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateScooterAvailability(t *testing.T) {
	logger := &log.Logger{}

	firstScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scooterAvailability := true

	tests := map[string]struct {
		logger                     *log.Logger
		mockRedisRepositoryHandler func(mock *mock.MockRedisRepository)
		wantErr                    bool
	}{
		"updating scooter successfully": {
			logger: logger,
			mockRedisRepositoryHandler: func(mock *mock.MockRedisRepository) {
				mock.EXPECT().UpdateScooterAvailability(firstScooterUUID, scooterAvailability).
					Return(nil).Times(1)
			},
			wantErr: false,
		},
		"updating scooter failed, because repository threw an error when updating scooter's availability": {
			logger: logger,
			mockRedisRepositoryHandler: func(mock *mock.MockRedisRepository) {
				mock.EXPECT().UpdateScooterAvailability(firstScooterUUID, scooterAvailability).
					Return(nil).Times(1)
			},
			wantErr: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockRedisRepository := mock.NewMockRedisRepository(controller)

			tt.mockRedisRepositoryHandler(mockRedisRepository)

			rs := NewRedisService(logger, mockRedisRepository)
			if err := rs.UpdateScooterAvailability(firstScooterUUID, scooterAvailability); (err != nil) != tt.wantErr {
				t.Errorf("UpdateScooterAvailability() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
