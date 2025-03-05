package main

import (
	"fmt"
	"log"
	"monalert/internal/config"
	"monalert/internal/handlers"
	"monalert/internal/repository"
	"monalert/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.GetConfig()
	store := repository.NewStore()
	monalertService := service.NewMonalert(store)
	if err := handlers.Serve(cfg.Handlers, monalertService); err != nil {
		return fmt.Errorf("failed to start server with config %s: %w", cfg.Handlers.ServerAddr, err)
	}
	return nil
}
