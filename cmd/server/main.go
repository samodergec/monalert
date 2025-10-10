package main

import (
	"fmt"
	"log"
	"monalert/internal/config"
	"monalert/internal/handlers"
	"monalert/internal/logger"
	"monalert/internal/repository"
	"monalert/internal/service"

	"go.uber.org/zap"
)

func main() {
	parseFlags()
	if err := logger.Initialize(flagLogLevel); err != nil {
		log.Fatal(err)
	}
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	logger.Log.Info("Running server", zap.String("log level", flagLogLevel))

	cfg := config.GetConfig()
	store := repository.NewStore()
	monalertService := service.NewMonalert(store)
	if err := handlers.Serve(cfg.Handlers, monalertService); err != nil {
		return fmt.Errorf("failed to start server with config %s: %w", cfg.Handlers.ServerAddr, err)
	}
	return nil
}
