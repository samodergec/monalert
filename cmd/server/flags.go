package main

import (
    "flag"
    "os"
)

var (
    flagLogLevel string
)

func parseFlags() {
    flag.StringVar(&flagLogLevel, "l", "info", "log level")
    flag.Parse()
    if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
        flagLogLevel = envLogLevel
    }
} 