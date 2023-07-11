package api

import "net/http"

const (
	version      = "/v1"
	scootersPath = "/scooters"
	rentPath     = "/rent"
	freePath     = "/free"
)

// registerRoutes sets service routes.
func (s *Server) registerRoutes() {
	versionRoute := s.router.PathPrefix(version).Subrouter()

	versionRoute.Path(scootersPath).Methods(http.MethodGet).HandlerFunc(s.GetScooters)

	versionRoute.Path(rentPath).Methods(http.MethodPost).HandlerFunc(s.RentScooter)
	versionRoute.Path(freePath).Methods(http.MethodPost).HandlerFunc(s.FreeScooter)
}
