package main

import (
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
	return handlers.Serve(cfg.Handlers, monalertService)
}
