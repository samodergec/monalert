package config

import (
	"flag"
	handlersConfig "monalert/internal/handlers/config"
	"os"
)

type Config struct {
	Handlers handlersConfig.Config
}

func GetConfig() Config {
	cfg := Config{}
	flag.StringVar(&cfg.Handlers.ServerAddr, "a", "localhost:8080", "address of http server")
	flag.Parse()
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		cfg.Handlers.ServerAddr = envRunAddr
	}
	return cfg
}
