//go:build unit

package repository

import (
	"encoding/json"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const (
	testCity      = "Montreal"
	testLongitude = 70.0
	testLatitude  = 60.0
	testRadius    = 10000.0
	testUnit      = "m"
)

func TestGetScooters(t *testing.T) {
	logger := log.New(os.Stdout, "TEST ", log.LstdFlags)

	firstScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	secScooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scooters := []redis.GeoLocation{
		{
			Name: firstScooterUUID.String(),
		},
		{
			Name: secScooterUUID.String(),
		},
	}

	tests := map[string]struct {
		logger        *log.Logger
		mockGeoRadius func(mock redismock.ClientMock)
		want          []redis.GeoLocation
		wantErr       bool
	}{
		"getting scooters successfully": {
			logger: logger,
			mockGeoRadius: func(mock redismock.ClientMock) {
				geoQuery := &redis.GeoRadiusQuery{
					Radius: testRadius,
					Unit:   testUnit,
				}

				mock.ExpectGeoRadius(testCity, testLongitude, testLatitude, geoQuery).SetVal(scooters)
			},
			want:    scooters,
			wantErr: false,
		},
		"getting scooters failed because of geo redis error": {
			logger: logger,
			mockGeoRadius: func(mock redismock.ClientMock) {
				geoQuery := &redis.GeoRadiusQuery{
					Radius: testRadius,
					Unit:   testUnit,
				}

				mock.ExpectGeoRadius(testCity, testLongitude, testLatitude, geoQuery).SetErr(redis.ErrClosed)
			},
			want:    nil,
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()

			tt.mockGeoRadius(mock)

			rr := NewRedisRepository(tt.logger, db)

			got, err := rr.GetScooters(testLongitude, testLatitude, testRadius, testCity)
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

func TestGetScooterLocation(t *testing.T) {
	logger := &log.Logger{}

	scooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scooterLocation := []*redis.GeoPos{
		{
			Longitude: testLongitude,
			Latitude:  testLatitude,
		},
	}

	tests := map[string]struct {
		logger     *log.Logger
		mockGeoPos func(mock redismock.ClientMock)
		want       *redis.GeoPos
		wantErr    bool
	}{
		"getting scooter's location successfully": {
			logger: logger,
			mockGeoPos: func(mock redismock.ClientMock) {
				mock.ExpectGeoPos(testCity, scooterUUID.String()).SetVal(scooterLocation)
			},
			want:    scooterLocation[0],
			wantErr: false,
		},
		"getting scooter's location failed, because geoPos returned empty list": {
			logger: logger,
			mockGeoPos: func(mock redismock.ClientMock) {
				mock.ExpectGeoPos(testCity, scooterUUID.String()).SetVal(nil)
			},
			want:    nil,
			wantErr: true,
		},
		"getting scooter's location failed, because of geoPos error": {
			logger: logger,
			mockGeoPos: func(mock redismock.ClientMock) {
				mock.ExpectGeoPos(testCity, scooterUUID.String()).SetErr(redis.ErrClosed)
			},
			want:    nil,
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()

			tt.mockGeoPos(mock)

			rr := NewRedisRepository(tt.logger, db)

			got, err := rr.GetScooterLocation(scooterUUID, testCity)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetScooterLocation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetScooterLocation() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetScooterAvailability(t *testing.T) {
	logger := &log.Logger{}

	scooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scooterAvailabilityAsInt := 1
	scooterAvailabilityAsJSON, err := json.Marshal(scooterAvailabilityAsInt)
	require.NoError(t, err)

	scooterAvailabilityIncorrectJSON, err := json.Marshal([]int{})
	require.NoError(t, err)

	tests := map[string]struct {
		logger  *log.Logger
		mockGet func(mock redismock.ClientMock)
		want    bool
		wantErr bool
	}{
		"getting scooter's availability successfully": {
			logger: logger,
			mockGet: func(mock redismock.ClientMock) {
				mock.ExpectGet(scooterUUID.String()).SetVal(string(scooterAvailabilityAsJSON))
			},
			want:    true,
			wantErr: false,
		},
		"getting scooter's availability failed, because of redisGet error": {
			logger: logger,
			mockGet: func(mock redismock.ClientMock) {
				mock.ExpectGet(scooterUUID.String()).SetErr(redis.ErrClosed)
			},
			want:    false,
			wantErr: true,
		},
		"getting scooter's availability failed, because of incorrect redis data": {
			logger: logger,
			mockGet: func(mock redismock.ClientMock) {
				mock.ExpectGet(scooterUUID.String()).SetVal(string(scooterAvailabilityIncorrectJSON))
			},
			want:    false,
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()

			tt.mockGet(mock)

			rr := NewRedisRepository(tt.logger, db)

			got, err := rr.GetScooterAvailability(scooterUUID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetScooterAvailability() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetScooterAvailability() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateScooterLocation(t *testing.T) {
	logger := &log.Logger{}

	scooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scooter := &redis.GeoLocation{
		Name:      scooterUUID.String(),
		Longitude: testLongitude,
		Latitude:  testLatitude,
	}

	tests := map[string]struct {
		logger     *log.Logger
		mockGeoAdd func(mock redismock.ClientMock)
		wantErr    bool
	}{
		"updating scooter's location successfully": {
			logger: logger,
			mockGeoAdd: func(mock redismock.ClientMock) {
				mock.ExpectGeoAdd(testCity, scooter).SetVal(1)
			},
			wantErr: false,
		},
		"updating scooter's location failed, because of GeoAdd error": {
			logger: logger,
			mockGeoAdd: func(mock redismock.ClientMock) {
				mock.ExpectGeoAdd(testCity, scooter).SetErr(redis.ErrClosed)
			},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()

			tt.mockGeoAdd(mock)

			rr := NewRedisRepository(tt.logger, db)

			if err = rr.UpdateScooterLocation(scooter, testCity); (err != nil) != tt.wantErr {
				t.Errorf("UpdateScooterLocation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateScooterAvailability(t *testing.T) {
	logger := &log.Logger{}

	scooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scooterAvailability := true

	tests := map[string]struct {
		logger  *log.Logger
		mockSet func(mock redismock.ClientMock)
		wantErr bool
	}{
		"updating scooter's availability successfully": {
			logger: logger,
			mockSet: func(mock redismock.ClientMock) {
				mock.ExpectSet(scooterUUID.String(), scooterAvailability, 0).SetVal("status")
			},
			wantErr: false,
		},
		"updating scooter's availability failed, because of redis Set error": {
			logger: logger,
			mockSet: func(mock redismock.ClientMock) {
				mock.ExpectSet(scooterUUID.String(), scooterAvailability, 0).SetErr(redis.ErrClosed)
			},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()

			tt.mockSet(mock)

			rr := NewRedisRepository(tt.logger, db)

			if err = rr.UpdateScooterAvailability(scooterUUID, scooterAvailability); (err != nil) != tt.wantErr {
				t.Errorf("UpdateScooterAvailability() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
