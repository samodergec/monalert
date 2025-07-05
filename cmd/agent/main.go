package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type MetricPoll struct {
	PollNumber     int64
	CounterMetrics map[string]int64
	GaugeMetrics   map[string]float64
}

var pollID int64

type CollectedMetricPolls struct {
	mux   *sync.Mutex
	Items []*MetricPoll
}

func NewMetricPoll() *MetricPoll {
	return &MetricPoll{
		CounterMetrics: make(map[string]int64),
		GaugeMetrics:   make(map[string]float64),
	}
}

func NewCollectedMetricPoll() CollectedMetricPolls {
	return CollectedMetricPolls{
		mux: &sync.Mutex{},
	}
}

func (cm *CollectedMetricPolls) Add(mp *MetricPoll) {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	cm.Items = append(cm.Items, mp)
}

func (cm *CollectedMetricPolls) Swap() []*MetricPoll {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	toSend := cm.Items
	cm.Items = nil
	return toSend
}

func CollectMetrics() *MetricPoll {
	poll := NewMetricPoll()
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)
	poll.GaugeMetrics["Alloc"] = float64(rtm.Alloc)
	poll.GaugeMetrics["BuckHashSys"] = float64(rtm.BuckHashSys)
	poll.GaugeMetrics["Frees"] = float64(rtm.Frees)
	poll.GaugeMetrics["GCCPUFraction"] = rtm.GCCPUFraction
	poll.GaugeMetrics["GCSys"] = float64(rtm.GCSys)
	poll.GaugeMetrics["HeapAlloc"] = float64(rtm.HeapAlloc)
	poll.GaugeMetrics["HeapIdle"] = float64(rtm.HeapIdle)
	poll.GaugeMetrics["HeapInuse"] = float64(rtm.HeapInuse)
	poll.GaugeMetrics["HeapObjects"] = float64(rtm.HeapObjects)
	poll.GaugeMetrics["HeapReleased"] = float64(rtm.HeapReleased)
	poll.GaugeMetrics["HeapSys"] = float64(rtm.HeapSys)
	poll.GaugeMetrics["Lookups"] = float64(rtm.Lookups)
	poll.GaugeMetrics["MCacheInuse"] = float64(rtm.MCacheInuse)
	poll.GaugeMetrics["MCacheSys"] = float64(rtm.MCacheSys)
	poll.GaugeMetrics["MSpanInuse"] = float64(rtm.MSpanInuse)
	poll.GaugeMetrics["MSpanSys"] = float64(rtm.MSpanSys)
	poll.GaugeMetrics["Mallocs"] = float64(rtm.Mallocs)
	poll.GaugeMetrics["NextGC"] = float64(rtm.NextGC)
	poll.GaugeMetrics["NumForcedGC"] = float64(rtm.NumForcedGC)
	poll.GaugeMetrics["NumGC"] = float64(rtm.NumGC)
	poll.GaugeMetrics["OtherSys"] = float64(rtm.OtherSys)
	poll.GaugeMetrics["PauseTotalNs"] = float64(rtm.PauseTotalNs)
	poll.GaugeMetrics["StackInuse"] = float64(rtm.StackInuse)
	poll.GaugeMetrics["StackSys"] = float64(rtm.StackSys)
	poll.GaugeMetrics["Sys"] = float64(rtm.Sys)
	poll.GaugeMetrics["TotalAlloc"] = float64(rtm.TotalAlloc)
	poll.GaugeMetrics["RandomValue"] = rand.Float64()
	poll.CounterMetrics["PollCount"] = int64(pollID)
	poll.PollNumber=pollID
	atomic.AddInt64(&pollID, 1)
	return poll
}

func (cm *CollectedMetricPolls) Collector() {
	pollInterval := time.Duration(flagPollInterval) * time.Second
	c := time.Tick(pollInterval)
	for range c {
		mp := CollectMetrics()
		cm.Add(mp)
	}
}

func (cm *CollectedMetricPolls) Sender() {
	reportInterval := time.Duration(flagReportInterval) * time.Second
	c := time.Tick(reportInterval)
	for range c {
		batch := cm.Swap()
		if len(batch) == 0 {
			continue
		}
		err := Send(batch)
		if err != nil {
			log.Printf("Failed to send metric %v", err)
		}
	}

}

func Send(cm []*MetricPoll) error {
	for _, v := range cm {
		for m, v := range v.GaugeMetrics {
			response, err := http.Post("http://"+flagServerAddr+"/update/gauge/"+m+"/"+strconv.FormatFloat(v, 'f', -1, 64), "text/html; charset=utf-8", nil)
			if err != nil {
				log.Printf("error in sending request %v", err)
				return err
			}

			if response.StatusCode != http.StatusOK {
				log.Printf("Сервер вернул статус: %d", response.StatusCode)
				return fmt.Errorf("Сервер вернул статус: %d", response.StatusCode)
			}

		}
		for m, v := range v.CounterMetrics {
			response, err := http.Post("http://"+flagServerAddr+"/update/counter/"+m+"/"+strconv.FormatInt(v, 10), "text/html; charset=utf-8", nil)
			if err != nil {
				log.Printf("error in sending request %v", err)
				return err
			}

			if response.StatusCode != http.StatusOK {
				log.Printf("Сервер вернул статус: %d", response.StatusCode)
				return fmt.Errorf("Сервер вернул статус: %d", response.StatusCode)
			}
		}
		log.Printf("New POLL sent %d", v.PollNumber)
	}
	return nil
}


func main() {
	parseFlags()
	collection := NewCollectedMetricPoll()
	go collection.Collector()
	go collection.Sender()
	select {}
}
