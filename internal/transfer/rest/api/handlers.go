package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/schema"

	"scootinAboot/internal/model"
	modelrental "scootinAboot/internal/module/rental/model"
)

const (
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
)

var (
	errExpectedHeaderParamNotFound = errors.New("expected header parameter was not found")
)

func (s *Server) GetScooters(w http.ResponseWriter, r *http.Request) {
	if _, err := clientUUIDFromHeader(r); err != nil {
		fmt.Println(err.Error())

		Error(w, http.StatusBadRequest, err, "getting clientUUID from header")

		return
	}

	var queryParams model.ScooterQueryParams

	decoder := schema.NewDecoder()

	if err := decoder.Decode(&queryParams, r.URL.Query()); err != nil {
		fmt.Println(err.Error())

		Error(w, http.StatusBadRequest, err, "decoding query params")

		return
	}

	redisScooters, err := s.redisService.GetScooters(
		queryParams.Longitude,
		queryParams.Latitude,
		queryParams.Radius,
		queryParams.City,
	)
	if err != nil {
		fmt.Println(err.Error())

		Error(w, http.StatusBadRequest, err, "getting scooters")

		return
	}

	scooters := make([]model.ScooterGet, len(redisScooters))

	for i := range redisScooters {
		scooterUUID, innerErr := uuid.Parse(redisScooters[i].Scooter.Name)
		if innerErr != nil {
			fmt.Println(innerErr.Error())

			Error(w, http.StatusBadRequest, innerErr, "parsing scooter's uuid")

			return
		}

		scooters[i] = model.ScooterGet{
			UUID:         scooterUUID,
			Latitude:     redisScooters[i].Scooter.Latitude,
			Longitude:    redisScooters[i].Scooter.Longitude,
			Availability: redisScooters[i].Availability,
		}
	}

	JSON(w, http.StatusOK, scooters)
}

func (s *Server) RentScooter(w http.ResponseWriter, r *http.Request) {
	if _, err := clientUUIDFromHeader(r); err != nil {
		fmt.Println(err.Error())

		Error(w, http.StatusBadRequest, err, "getting clientUUID from header")

		return
	}

	var scooter model.ScooterPost

	if err := json.NewDecoder(r.Body).Decode(&scooter); err != nil {
		fmt.Println(err.Error())

		Error(w, http.StatusBadRequest, err, "decoding request body to scooter")

		return
	}

	rentalScooter := modelrental.RentalScooter{
		GeoLocation: &redis.GeoLocation{
			Name:      scooter.UUID.String(),
			Longitude: scooter.Longitude,
			Latitude:  scooter.Latitude,
		},
		City:         scooter.City,
		Availability: scooter.Availability,
	}

	if err := s.rentalService.Rent(&rentalScooter); err != nil {
		fmt.Println(err.Error())

		Error(w, http.StatusBadRequest, err, "renting scooter")

		return
	}

	JSON(w, http.StatusNoContent, nil)
}

func (s *Server) FreeScooter(w http.ResponseWriter, r *http.Request) {
	if _, err := clientUUIDFromHeader(r); err != nil {
		fmt.Println(err.Error())

		Error(w, http.StatusBadRequest, err, "getting clientUUID from header")

		return
	}

	var scooterUUID uuid.UUID

	if err := json.NewDecoder(r.Body).Decode(&scooterUUID); err != nil {
		fmt.Println(err.Error())

		Error(w, http.StatusBadRequest, err, "decoding request body to scooter")

		return
	}

	if err := s.rentalService.Free(scooterUUID); err != nil {
		fmt.Println(err.Error())

		Error(w, http.StatusBadRequest, err, "freeing scooter")

		return
	}

	JSON(w, http.StatusNoContent, nil)
}

func clientUUIDFromHeader(r *http.Request) (uuid.UUID, error) {
	clientUUIDAsString := r.Header.Get("clientUUID")
	if len(clientUUIDAsString) == 0 {
		return uuid.Nil, errExpectedHeaderParamNotFound
	}

	clientUUID, err := uuid.Parse(clientUUIDAsString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parsing clientUUID: %w", err)
	}

	return clientUUID, nil
}

// JSON writes a JSON response.
func JSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set(headerContentType, contentTypeJSON)
	body, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"Error":"%v"}`, err.Error())))

		return
	}

	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}

// Error writes an error response.
func Error(w http.ResponseWriter, statusCode int, err error, message string) {
	apiError := struct {
		Error   string `json:"Error"`
		Message string `json:"Message"`
	}{
		Error:   err.Error(),
		Message: message,
	}

	JSON(w, statusCode, &apiError)
}
