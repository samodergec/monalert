package main

import (
	"fmt"
	"log"
	"monalert/internal/handlers"
	"monalert/internal/logger"
	"monalert/internal/repository"
	"monalert/internal/service"
	"time"

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
	store := repository.NewStore(flagFileStoragePath)
	monalertService := service.NewMonalert(store)
	if flagStoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(flagStoreInterval) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := store.Persist(); err != nil {
					logger.Log.Error("persist error:", zap.Error(err))
				}
			}
		}()
	}
	if err := handlers.Serve(flagServerAddr, monalertService); err != nil {
		return fmt.Errorf("failed to start server with config %s: %w", flagServerAddr, err)
	}
	return nil
}
