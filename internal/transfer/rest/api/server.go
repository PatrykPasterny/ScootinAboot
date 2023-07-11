package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gorilla/mux"

	redis "scootinAboot/internal/module/redis/transfer"
	"scootinAboot/internal/module/rental/transfer"
)

type Server struct {
	logger        *log.Logger
	httpServer    *http.Server
	router        *mux.Router
	redisService  redis.RedisService
	rentalService transfer.RentalService
}

func NewServer(
	logger *log.Logger,
	server *http.Server,
	router *mux.Router,
	redis redis.RedisService,
	rental transfer.RentalService,
) *Server {

	s := &Server{
		logger:        logger,
		httpServer:    server,
		router:        router,
		redisService:  redis,
		rentalService: rental,
	}

	s.registerRoutes()

	return s
}

// Run starts the work of the service.
func (s *Server) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	waitGroup := sync.WaitGroup{}

	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

		sig := <-signalChan
		s.logger.Println(sig.String())

		cancel()
	}()

	waitGroup.Add(1)

	go func(running *sync.WaitGroup) {
		defer running.Done()
		s.logger.Printf("Starting HTTP server: address - %s\n", s.httpServer.Addr)

		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			cancel()

			s.logger.Fatalf("can't close http server: %v\n", err)
		}

		s.logger.Println("Stopping HTTP server")
	}(&waitGroup)

	<-ctx.Done()

	if err := s.httpServer.Shutdown(context.Background()); err != nil {
		s.logger.Fatalf("can't shutdown gracefully: %v\n", err)
	}

	waitGroup.Wait()
}
