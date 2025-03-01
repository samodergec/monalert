package main

import (
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

type gaugeMetrics map[string]float64
type counterMetrics map[string]int64

type MetricPoll struct {
	CounterMetrics counterMetrics
	GaugeMetrics   gaugeMetrics
	Sent           bool
}

func NewMetricPoll() MetricPoll {
	return MetricPoll{
		CounterMetrics: make(counterMetrics), // Инициализируем пустую мапу
		GaugeMetrics:   make(gaugeMetrics),   // Инициализируем пустую мапу
		Sent:           false,                // По умолчанию "не отправлено"
	}
}

type CollectedMetricPoll []MetricPoll

func NewCollectedMetricPoll() CollectedMetricPoll {
	return CollectedMetricPoll{} // Инициализируем пустой срез MetricPoll
}

func metricCollector(cm *CollectedMetricPoll) {
	const pollInterval = 2 * time.Second
	c := time.Tick(pollInterval)
	PollCount := 0
	for range c {
		var rtm runtime.MemStats
		runtime.ReadMemStats(&rtm)
		(*cm) = append((*cm), NewMetricPoll())
		(*cm)[PollCount].GaugeMetrics["Alloc"] = float64(rtm.Alloc)
		(*cm)[PollCount].GaugeMetrics["BuckHashSys"] = float64(rtm.BuckHashSys)
		(*cm)[PollCount].GaugeMetrics["Frees"] = float64(rtm.Frees)
		(*cm)[PollCount].GaugeMetrics["GCCPUFraction"] = rtm.GCCPUFraction
		(*cm)[PollCount].GaugeMetrics["GCSys"] = float64(rtm.GCSys)
		(*cm)[PollCount].GaugeMetrics["HeapAlloc"] = float64(rtm.HeapAlloc)
		(*cm)[PollCount].GaugeMetrics["HeapIdle"] = float64(rtm.HeapIdle)
		(*cm)[PollCount].GaugeMetrics["HeapInuse"] = float64(rtm.HeapInuse)
		(*cm)[PollCount].GaugeMetrics["HeapObjects"] = float64(rtm.HeapObjects)
		(*cm)[PollCount].GaugeMetrics["HeapReleased"] = float64(rtm.HeapReleased)
		(*cm)[PollCount].GaugeMetrics["HeapSys"] = float64(rtm.HeapSys)
		(*cm)[PollCount].GaugeMetrics["Lookups"] = float64(rtm.Lookups)
		(*cm)[PollCount].GaugeMetrics["MCacheInuse"] = float64(rtm.MCacheInuse)
		(*cm)[PollCount].GaugeMetrics["MCacheSys"] = float64(rtm.MCacheSys)
		(*cm)[PollCount].GaugeMetrics["MSpanInuse"] = float64(rtm.MSpanInuse)
		(*cm)[PollCount].GaugeMetrics["MSpanSys"] = float64(rtm.MSpanSys)
		(*cm)[PollCount].GaugeMetrics["Mallocs"] = float64(rtm.Mallocs)
		(*cm)[PollCount].GaugeMetrics["NextGC"] = float64(rtm.NextGC)
		(*cm)[PollCount].GaugeMetrics["NumForcedGC"] = float64(rtm.NumForcedGC)
		(*cm)[PollCount].GaugeMetrics["NumGC"] = float64(rtm.NumGC)
		(*cm)[PollCount].GaugeMetrics["OtherSys"] = float64(rtm.OtherSys)
		(*cm)[PollCount].GaugeMetrics["PauseTotalNs"] = float64(rtm.PauseTotalNs)
		(*cm)[PollCount].GaugeMetrics["StackInuse"] = float64(rtm.StackInuse)
		(*cm)[PollCount].GaugeMetrics["StackSys"] = float64(rtm.StackSys)
		(*cm)[PollCount].GaugeMetrics["Sys"] = float64(rtm.Sys)
		(*cm)[PollCount].GaugeMetrics["TotalAlloc"] = float64(rtm.TotalAlloc)
		(*cm)[PollCount].GaugeMetrics["RandomValue"] = rand.Float64()
		(*cm)[PollCount].CounterMetrics["PollCount"] = int64(PollCount)
		PollCount++
	}
}

func metricSender(cm *CollectedMetricPoll) {
	for {
		for i, v := range *cm {
			if !(*cm)[i].Sent {
				for m, v := range v.GaugeMetrics {
					response, err := http.Post("http://localhost:8080/update/gauge/"+m+"/"+strconv.FormatFloat(v, 'f', -1, 64), "text/html; charset=utf-8", nil)
					if err != nil {
						log.Fatal(err)
					}
					if response.StatusCode != http.StatusOK {
						log.Fatal(response.StatusCode)
					}

				}
				for m, v := range v.CounterMetrics {
					response, err := http.Post("http://localhost:8080/update/counter/"+m+"/"+strconv.FormatInt(v, 10), "text/html; charset=utf-8", nil)
					if err != nil {
						log.Fatal(err)
					}
					if response.StatusCode != http.StatusOK {
						log.Fatal(response.StatusCode)
					}

				}
				(*cm)[i].Sent = true
			}
		}
		time.Sleep(10 * time.Second)
	}
}

func main() {
	cm := NewCollectedMetricPoll()
	cm = append(cm, NewMetricPoll())
	go metricCollector(&cm)
	metricSender(&cm)
}
