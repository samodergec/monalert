package main

import (
	"flag"
	"os"
)

var (
	flagLogLevel   string
	flagServerAddr string
)

func parseFlags() {
	flag.StringVar(&flagLogLevel, "l", "info", "log level")
	flag.StringVar(&flagServerAddr, "a", ":8080", "address of http server")
	flag.Parse()
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		flagLogLevel = envLogLevel
	}
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagServerAddr = envRunAddr
	}
}
