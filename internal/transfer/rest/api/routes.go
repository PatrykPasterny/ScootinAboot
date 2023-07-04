package api

import "net/http"

const (
	version  = "/v1"
	scooters = "/scooters"
	scooter  = "/scooters/{scooterUUID}"
	rent     = "/rent"
	free     = "/rent"
)

// registerRoutes sets service routes.
func (s *Server) registerRoutes() {
	versionRoute := s.router.PathPrefix(version).Subrouter()

	versionRoute.Path(scooters).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	versionRoute.Path(scooter).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	versionRoute.Path(rent).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	versionRoute.Path(free).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
}
