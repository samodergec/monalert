package config

import (
	"flag"
	handlersConfig "monalert/internal/handlers/config"
)

type Config struct {
	Handlers handlersConfig.Config
}

func GetConfig() Config{
	cfg := Config{}
	flag.StringVar(&cfg.Handlers.ServerAddr, "addr", "localhost:8080", "address of http server")
	flag.Parse()
	return cfg
}