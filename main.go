package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"scootinAboot/ScootinAboot/internal/config"
	"scootinAboot/ScootinAboot/internal/transfer/rest/api"

	"github.com/gorilla/mux"
)

const configPath = "internal/config/default.env"

func main() {
	cfg, err := config.NewConfig(context.Background(), configPath)
	if err != nil {
		log.Fatal(fmt.Errorf("config retrieval failed: %w", err))
	}

	logger := log.New(os.Stdout, "CUSTOM ", log.LstdFlags)

	logger.Println("Starting Scootin Aboot")

	router := mux.NewRouter()

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTP),
		Handler: router,
	}

	server := api.NewServer(logger, httpServer, router)

	server.Run()
}
