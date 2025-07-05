package main

import "flag"

var flagServerAddr string
var flagReportInterval int
var flagPollInterval int


func parseFlags() {
	flag.StringVar(&flagServerAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&flagReportInterval, "r", 2, "interval for metric send")
	flag.IntVar(&flagPollInterval, "p", 1, "interval for collecting metrics")
	flag.Parse()
}
