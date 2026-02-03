package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"monalert/internal/logger"
	"monalert/internal/models"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type MetricPoll struct {
	CounterMetrics map[string]int64
	GaugeMetrics   map[string]float64
	PollNumber     int64
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
	poll.GaugeMetrics["LastGC"] = float64(rtm.LastGC)
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
	poll.CounterMetrics["PollCount"] = pollID
	poll.PollNumber = pollID
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
			log.Printf("failed to send metric poll %v", err)
		}
	}
}

func SendRequest(address string) error {
	var lastErr error
	for range 3 {
		response, err := http.Post(address, "text/html; charset=utf-8", nil)
		if err != nil {
			if errors.Is(err, io.EOF) {
				lastErr = err
				continue
			}
			if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "connection refused") {
				lastErr = err
				time.Sleep(100 * time.Millisecond)
				continue
			}
			logger.Log.Error("error in sending request", zap.String("request:", address), zap.Error(err))
			return fmt.Errorf("error in sending request: %w", err)
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close() //nolint:gosec // response.Body.Close() error is intentionally ignored
			logger.Log.Error("server returned status", zap.Int("code:", response.StatusCode))
			return fmt.Errorf("server returned status: %d", response.StatusCode)
		}
		response.Body.Close() //nolint:gosec // response.Body.Close() error is intentionally ignored
		return nil
	}
	return fmt.Errorf("request failed after retry: %w", lastErr)
}

func SendJSONRequest(buf *bytes.Buffer) error {
	response, err := http.Post("http://"+flagServerAddr+"/update", `application/json`, buf)

	if err != nil {
		return fmt.Errorf("sending request failed: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", response.StatusCode)
	}
	defer response.Body.Close()
	return nil
}

func Send(cm []*MetricPoll) error {
	for _, poll := range cm {
		if flagUseJSON != "" {
			logger.Log.Debug("using JSON for sending", zap.String("JSON flag", flagUseJSON))
			for m, v := range poll.GaugeMetrics {
				var buf bytes.Buffer
				enc := json.NewEncoder(&buf)
				if err := enc.Encode(&models.Metrics{
					ID:    m,
					MType: "gauge",
					Value: &v,
				}); err != nil {
					logger.Log.Debug("error encoding response", zap.Error(err))
					return fmt.Errorf("error encoding response %w", err)
				}
				if err := SendJSONRequest(&buf); err != nil {
					return err
				}
			}
			for m, v := range poll.CounterMetrics {
				var buf bytes.Buffer
				enc := json.NewEncoder(&buf)
				if err := enc.Encode(&models.Metrics{
					ID:    m,
					MType: "counter",
					Delta: &v,
				}); err != nil {
					logger.Log.Debug("error encoding response", zap.Error(err))
					return err
				}
				if err := SendJSONRequest(&buf); err != nil {
					return err
				}
			}
		} else {
			logger.Log.Debug("using URL for sending", zap.String("JSON flag", flagUseJSON))
			for metricName, value := range poll.GaugeMetrics {
				address := "http://" + flagServerAddr + "/update/gauge/" + metricName + "/" + strconv.FormatFloat(value, 'f', -1, 64)
				if err := SendRequest(address); err != nil {
					return err
				}
			}
			for metricName, value := range poll.CounterMetrics {
				address := "http://" + flagServerAddr + "/update/counter/" + metricName + "/" + strconv.FormatInt(value, 10)
				if err := SendRequest(address); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func main() {
	parseFlags()
	if flagPollInterval == 0 || flagReportInterval == 0 {
		logger.Log.Panic("incorrect poll or report interval:", zap.Any("poll interval:", flagPollInterval), zap.Any("report interval", flagReportInterval))
	}
	collection := NewCollectedMetricPoll()
	go collection.Collector()
	go collection.Sender()
	select {}
}
