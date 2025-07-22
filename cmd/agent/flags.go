package main

import (
	"flag"
	"os"
	"strconv"
)

type Config struct {
}

var flagServerAddr string
var flagReportInterval int
var flagPollInterval int

func parseFlags() {
	flag.StringVar(&flagServerAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&flagReportInterval, "r", 2, "interval for metric send")
	flag.IntVar(&flagPollInterval, "p", 1, "interval for collecting metrics")
	flag.Parse()

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagServerAddr = envRunAddr
	}
	if envReportInterval, _ := strconv.Atoi(os.Getenv("REPORT_INTERVAL")); envReportInterval != 0 {
		flagReportInterval = envReportInterval
	}
	if envPollInterval, _ := strconv.Atoi(os.Getenv("POLL_INTERVAL")); envPollInterval != 0 {
		flagPollInterval = envPollInterval
	}
}
