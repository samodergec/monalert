package repository

import (
	"fmt"
	"log"
	"strconv"
	"sync"
)

type Store struct {
	mux          *sync.Mutex
	gaugeStore   map[string]float64
	counterStore map[string]int64
}

func NewStore() *Store {
	return &Store{
		mux:          &sync.Mutex{},
		gaugeStore:   make(map[string]float64),
		counterStore: make(map[string]int64),
	}
}

type GaugeUpdateRequest struct {
	MetricName  string
	MetricValue float64
}

type CounterUpdateRequest struct {
	MetricName  string
	MetricValue int64
}

func (s *Store) GaugeUpdate(req *GaugeUpdateRequest) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.gaugeStore[req.MetricName] = req.MetricValue
}

func (s *Store) CounterUpdate(req *CounterUpdateRequest) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.counterStore[req.MetricName] += req.MetricValue
}

type GetMetricValueRequest struct {
	MetricType string
	MetricName string
}

type GetMetricValueResponse struct {
	MetricValue string
}

func (s *Store) GetMetricValue(req *GetMetricValueRequest) (*GetMetricValueResponse, error) {
	
	s.mux.Lock()
	defer s.mux.Unlock()
	if req.MetricType == "gauge" {
		log.Printf("repo = %+v", s.gaugeStore)
		if val, ok := s.gaugeStore[req.MetricName]; ok {
			return &GetMetricValueResponse{
				MetricValue: fmt.Sprintf("%f", val),
			}, nil
		} else {
			return &GetMetricValueResponse{
				MetricValue: "",
			}, fmt.Errorf("no metric in storage with provided name: %s", req.MetricName)
		}
	}
	if req.MetricType == "counter" {
		if val, ok := s.counterStore[req.MetricName]; ok {
			return &GetMetricValueResponse{
				MetricValue: strconv.FormatInt(val, 10),
			}, nil
		} else {
			return &GetMetricValueResponse{
				MetricValue: "",
			}, fmt.Errorf("no metric in storage with provided name: %s", req.MetricName)
		}
	}
	return &GetMetricValueResponse{
		MetricValue: "",
	}, fmt.Errorf("no metric in storage with provided type: %s", req.MetricType)
}

func (s *Store) GetAllMetrics() []string {
	s.mux.Lock()
	defer s.mux.Unlock()
	metrics := make([]string, 0, len(s.gaugeStore)+len(s.counterStore))
	for metric := range s.gaugeStore {
		metrics = append(metrics, fmt.Sprintf("gauge:%s:%f", metric, s.gaugeStore[metric]))
	}
	for metric := range s.counterStore {
		metrics = append(metrics, fmt.Sprintf("counter:%s:%d", metric, s.counterStore[metric]))
	}
	return metrics
}
