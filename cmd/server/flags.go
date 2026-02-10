package main

import (
	"flag"
	"log"
	"os"
	"strconv"
)

var (
	flagLogLevel        string
	flagServerAddr      string
	flagStoreInterval   int
	flagFileStoragePath string
	flagRestore         bool
)

func parseFlags() {
	flag.StringVar(&flagLogLevel, "l", "info", "log level")
	flag.StringVar(&flagServerAddr, "a", ":8080", "address of http server")
	flag.IntVar(&flagStoreInterval, "i", 300, "store interval")
	flag.StringVar(&flagFileStoragePath, "f", "/tmp/metrics-db.json", "file for storage")
	flag.BoolVar(&flagRestore, "r", true, "restore data from storage file")
	flag.Parse()
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		flagLogLevel = envLogLevel
	}
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagServerAddr = envRunAddr
	}
	if v := os.Getenv("STORE_INTERVAL"); v != "" {
		envStoreInterval, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("invalid STORE_INTERVAL=%q: %v", v, err)
		}
		flagStoreInterval = envStoreInterval
	}
	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		flagFileStoragePath = envFileStoragePath
	}
	if v := os.Getenv("RESTORE"); v != "" {
		envRestore, err := strconv.ParseBool(v)
		if err != nil {
			log.Fatalf("invalid RESTORE=%q: %v", v, err)
		}
		flagRestore = envRestore
	}
}
