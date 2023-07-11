//go:build unit

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	rentalmodel "scootinAboot/internal/module/rental/model"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"scootinAboot/internal/model"
	redismodel "scootinAboot/internal/module/redis/model"
	mockredis "scootinAboot/internal/module/redis/transfer/mock"
	mockrental "scootinAboot/internal/module/rental/transfer/mock"
)

const (
	testCity      = "Montreal"
	testLongitude = 70.0
	testLatitude  = 60.0
	testRadius    = 10000.0
)

func TestGetScooters(t *testing.T) {
	s, mockRedisService, _ := beforeTest(t)

	scooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	redisScooters := []*redismodel.RedisScooter{
		redismodel.NewRedisScooter(
			&redis.GeoLocation{
				Name: scooterUUID.String(),
			},
			&redis.GeoPos{
				Longitude: testLongitude,
				Latitude:  testLatitude,
			},
			true,
		),
	}

	expectedScooters := []model.ScooterGet{
		{
			UUID:         scooterUUID,
			Longitude:    testLongitude,
			Latitude:     testLatitude,
			Availability: true,
		},
	}

	params := &model.ScooterQueryParams{
		Longitude: testLongitude,
		Latitude:  testLatitude,
		Radius:    testRadius,
		City:      testCity,
	}

	validURLQuery := &url.Values{}
	validURLQuery.Add("longitude", strconv.FormatFloat(params.Longitude, 'f', -1, 64))
	validURLQuery.Add("latitude", strconv.FormatFloat(params.Latitude, 'f', -1, 64))
	validURLQuery.Add("radius", strconv.FormatFloat(params.Radius, 'f', -1, 64))
	validURLQuery.Add("city", params.City)

	invalidURLQuery := &url.Values{}
	invalidURLQuery.Add("wrong", "wrong")

	expectedScootersJSON, err := json.Marshal(expectedScooters)
	require.NoError(t, err)

	tests := map[string]struct {
		mockRedisServiceHandler func(mock *mockredis.MockRedisService)
		urlQuery                *url.Values
		withHeader              bool
		expectedCode            int
		expectedBody            string
	}{
		"successfully getting scooters": {
			mockRedisServiceHandler: func(mock *mockredis.MockRedisService) {
				mock.EXPECT().GetScooters(testLongitude, testLatitude, testRadius, testCity).
					Return(redisScooters, nil).Times(1)
			},
			urlQuery:     validURLQuery,
			withHeader:   true,
			expectedCode: http.StatusOK,
			expectedBody: string(expectedScootersJSON),
		},
		"failed getting scooter because request has no clientUUID in header": {
			mockRedisServiceHandler: nil,
			urlQuery:                validURLQuery,
			withHeader:              false,
			expectedCode:            http.StatusBadRequest,
			expectedBody:            "{\"Error\":\"expected header parameter was not found\",\"Message\":\"getting clientUUID from header\"}",
		},
		"failed getting scooter because request has wrong query params": {
			mockRedisServiceHandler: nil,
			urlQuery:                invalidURLQuery,
			withHeader:              true,
			expectedCode:            http.StatusBadRequest,
			expectedBody:            "{\"Error\":\"schema: invalid path \\\"wrong\\\"\",\"Message\":\"decoding query params\"}",
		},
		"failed getting scooter because redis service threw error while getting scooters": {
			mockRedisServiceHandler: func(mock *mockredis.MockRedisService) {
				mock.EXPECT().GetScooters(testLongitude, testLatitude, testRadius, testCity).
					Return(nil, errors.New("")).Times(1)
			},
			urlQuery:     validURLQuery,
			withHeader:   true,
			expectedCode: http.StatusBadRequest,
			expectedBody: "{\"Error\":\"\",\"Message\":\"getting scooters\"}",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			request := buildRequest(t, scootersPath, http.MethodGet, &bytes.Buffer{}, tt.withHeader)

			request.URL.RawQuery = tt.urlQuery.Encode()

			responseRecorder := httptest.NewRecorder()

			if tt.mockRedisServiceHandler != nil {
				tt.mockRedisServiceHandler(mockRedisService)
			}

			s.GetScooters(responseRecorder, request)

			if status := responseRecorder.Code; status != tt.expectedCode {
				t.Errorf("handler returned wrong status code: got = %v want = %v",
					status, tt.expectedCode)
			}

			if body := responseRecorder.Body.String(); body != tt.expectedBody {
				t.Errorf("handler returned unexpected body: got = %v want = %v",
					body, tt.expectedBody)
			}
		})
	}
}

func TestRentScooter(t *testing.T) {
	s, _, mockRentalService := beforeTest(t)

	scooterUUID, err := uuid.NewRandom()
	require.NoError(t, err)

	scooter := model.ScooterPost{
		UUID:         scooterUUID,
		Longitude:    testLongitude,
		Latitude:     testLatitude,
		City:         testCity,
		Availability: true,
	}

	scooterJSON, err := json.Marshal(scooter)
	require.NoError(t, err)

	invalidScooterJSON, err := json.Marshal("invalidScooter")
	require.NoError(t, err)

	rentalScooter := &rentalmodel.RentalScooter{
		GeoLocation: &redis.GeoLocation{
			Name:      scooter.UUID.String(),
			Latitude:  scooter.Latitude,
			Longitude: scooter.Longitude,
		},
		City:         scooter.City,
		Availability: scooter.Availability,
	}

	tests := map[string]struct {
		mockRentalServiceHandler func(mock *mockrental.MockRentalService)
		body                     *bytes.Buffer
		withHeader               bool
		expectedCode             int
	}{
		"successfully renting scooter": {
			mockRentalServiceHandler: func(mock *mockrental.MockRentalService) {
				mock.EXPECT().Rent(rentalScooter).Return(nil).Times(1)
			},
			body:         bytes.NewBuffer(scooterJSON),
			withHeader:   true,
			expectedCode: http.StatusNoContent,
		},
		"failed renting scooter because request has no clientUUID in header": {
			mockRentalServiceHandler: nil,
			body:                     bytes.NewBuffer(scooterJSON),
			withHeader:               false,
			expectedCode:             http.StatusBadRequest,
		},
		"failed renting scooter because request has invalid body": {
			mockRentalServiceHandler: nil,
			body:                     bytes.NewBuffer(invalidScooterJSON),
			withHeader:               true,
			expectedCode:             http.StatusBadRequest,
		},
		"failed renting scooter because rental service threw error while renting scooter": {
			mockRentalServiceHandler: func(mock *mockrental.MockRentalService) {
				mock.EXPECT().Rent(rentalScooter).Return(errors.New("")).Times(1)
			},
			body:         bytes.NewBuffer(scooterJSON),
			withHeader:   true,
			expectedCode: http.StatusBadRequest,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			request := buildRequest(t, rentPath, http.MethodPost, tt.body, tt.withHeader)

			responseRecorder := httptest.NewRecorder()

			if tt.mockRentalServiceHandler != nil {
				tt.mockRentalServiceHandler(mockRentalService)
			}
			s.RentScooter(responseRecorder, request)

			if status := responseRecorder.Code; status != tt.expectedCode {
				t.Errorf("handler returned wrong status code: got = %v want = %v",
					status, tt.expectedCode)
			}
		})
	}
}

func beforeTest(t *testing.T) (*Server, *mockredis.MockRedisService, *mockrental.MockRentalService) {
	logger := log.New(os.Stdout, "TEST ", log.LstdFlags)

	controller := gomock.NewController(t)
	httpRouter := mux.NewRouter()

	mockRedisService := mockredis.NewMockRedisService(controller)
	mockRentalService := mockrental.NewMockRentalService(controller)

	s := NewServer(
		logger,
		&http.Server{
			Addr:    fmt.Sprintf(":%d", 8081),
			Handler: httpRouter,
		},
		httpRouter,
		mockRedisService,
		mockRentalService,
	)

	return s, mockRedisService, mockRentalService
}

func buildRequest(t *testing.T, path string, method string, body *bytes.Buffer, withHeader bool) *http.Request {
	t.Helper()

	request, err := http.NewRequestWithContext(
		context.Background(),
		method,
		version+path,
		body,
	)
	require.NoErrorf(t, err, "Building new request")

	if withHeader {
		clientUUID, innerErr := uuid.NewRandom()
		require.NoError(t, innerErr)

		request.Header.Set("clientUUID", clientUUID.String())
	}

	return request
}
