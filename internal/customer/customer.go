package customer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"scootinAboot/internal/model"
	"strconv"
	"sync"
	"time"
)

const (
	base         = "http://localhost:8081"
	version      = "/v1"
	scootersPath = "/scooters"
	rentPath     = "/rent"
	freePath     = "/free"

	numberOfScooterRentals = 5
	timeOfScooterRentals   = 10
)

var errUnexpectedResponseStatus = errors.New("received unexpected response status")

type Client struct {
	ClientUUID uuid.UUID
	Longitude  float64
	Latitude   float64
	Radius     float64
	City       string
}

type ClientService interface {
	UseScooterAboot()
}

type clientService struct {
	logger *log.Logger
	client *http.Client
}

func NewClientService(logger *log.Logger) *clientService {
	return &clientService{
		logger: logger,
		client: http.DefaultClient,
	}
}

func (c *clientService) UseScooterAboot(client *Client, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	for i := 1; i <= numberOfScooterRentals; i++ {
		scooters, err := c.getScooters(client)
		if err != nil {
			c.logger.Fatal(err.Error())
		}

		availableScooters := filterAvailableScooters(scooters)
		if len(availableScooters) == 0 {
			c.logger.Println("no available scooters")
			i--
			continue
		}

		j := rand.Intn(len(availableScooters))

		innerErr := c.rentScooter(client, &availableScooters[j], client.City)
		if innerErr != nil {
			i--
			continue
		}

		time.Sleep(timeOfScooterRentals * time.Second)

		innerErr = c.freeScooter(client, availableScooters[j].UUID)
		if innerErr != nil {
			c.logger.Fatal(innerErr.Error())
		}
	}
}

func (c *clientService) getScooters(client *Client) ([]model.ScooterGet, error) {
	requestScooters, err := buildRequest(client, scootersPath, http.MethodGet, &bytes.Buffer{})
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	validURLQuery := &url.Values{}
	validURLQuery.Add("longitude", strconv.FormatFloat(client.Longitude, 'f', -1, 64))
	validURLQuery.Add("latitude", strconv.FormatFloat(client.Latitude, 'f', -1, 64))
	validURLQuery.Add("radius", strconv.FormatFloat(client.Radius, 'f', -1, 64))
	validURLQuery.Add("city", client.City)

	requestScooters.URL.RawQuery = validURLQuery.Encode()

	response, err := c.client.Do(requestScooters)
	if err != nil {
		return nil, fmt.Errorf("requesting scooters: %w", err)
	}

	defer func() {
		if err = response.Body.Close(); err != nil {
			c.logger.Fatal(err.Error())
		}
	}()

	switch response.StatusCode {
	case http.StatusOK:
		var scooters []model.ScooterGet
		if err = json.NewDecoder(response.Body).Decode(&scooters); err != nil {
			return nil, fmt.Errorf("decoding response body: %w", err)
		}

		return scooters, nil
	default:
		return nil, fmt.Errorf(
			"receiving response with status %s: %w",
			response.Status,
			errUnexpectedResponseStatus,
		)
	}
}

func (c *clientService) rentScooter(client *Client, scooter *model.ScooterGet, city string) error {
	scooterPost := model.ScooterPost{
		UUID:         scooter.UUID,
		Longitude:    scooter.Longitude,
		Latitude:     scooter.Latitude,
		Availability: scooter.Availability,
		City:         city,
	}

	scooterJSON, err := json.Marshal(scooterPost)
	if err != nil {
		return fmt.Errorf("marshaling scooter to JSON: %w", err)
	}
	requestRental, err := buildRequest(client, rentPath, http.MethodPost, bytes.NewBuffer(scooterJSON))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	response, err := c.client.Do(requestRental)
	if err != nil {
		return fmt.Errorf("requesting rental of a scooter: %w", err)
	}

	defer func() {
		if err = response.Body.Close(); err != nil {
			c.logger.Fatal(err.Error())
		}
	}()

	switch response.StatusCode {
	case http.StatusNoContent:
		c.logger.Println("rented scooter successfully", scooter.UUID)

		return nil
	default:
		return fmt.Errorf(
			"receiving response with status %s: %w",
			response.Status,
			errUnexpectedResponseStatus,
		)
	}
}

func (c *clientService) freeScooter(client *Client, scooterUUID uuid.UUID) error {
	scooterUUIDJSON, err := json.Marshal(scooterUUID)
	if err != nil {
		return fmt.Errorf("marshaling scooterUUID to JSON: %w", err)
	}
	requestFreeingScooter, err := buildRequest(client, freePath, http.MethodPost, bytes.NewBuffer(scooterUUIDJSON))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	response, err := c.client.Do(requestFreeingScooter)
	if err != nil {
		return fmt.Errorf("requesting freeing of the scooter: %w", err)
	}

	defer func() {
		if err = response.Body.Close(); err != nil {
			c.logger.Fatal(err.Error())
		}
	}()

	switch response.StatusCode {
	case http.StatusNoContent:
		c.logger.Println("freed scooter successfully", scooterUUID)

		return nil
	default:
		return fmt.Errorf(
			"receiving response with status %s: %w",
			response.Status,
			errUnexpectedResponseStatus,
		)
	}
}

func buildRequest(c *Client, path string, method string, body *bytes.Buffer) (*http.Request, error) {

	request, err := http.NewRequestWithContext(
		context.Background(),
		method,
		base+version+path,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("creating new request: %w", err)
	}

	request.Header.Set("clientUUID", c.ClientUUID.String())

	return request, nil
}

func filterAvailableScooters(scooters []model.ScooterGet) []model.ScooterGet {
	var filteredScooters []model.ScooterGet

	for i := range scooters {
		if scooters[i].Availability {
			filteredScooters = append(filteredScooters, scooters[i])
		}
	}

	return filteredScooters
}
