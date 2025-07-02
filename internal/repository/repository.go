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

type Metric struct {
	Name  string
	Type  string
	Float float64
	Int   int64
}

func (s *Store) MetricUpdate(req *Metric) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	switch req.Type {
	case "gauge":
		s.gaugeStore[req.Name] = req.Float
		log.Printf("repository: storage saved metric type: %s, name: %s, value:%f", req.Type, req.Name, req.Float)
		return nil
	case "counter":
		s.counterStore[req.Name] += req.Int
		log.Printf("repository: storage saved metric type: %s, name: %s, value:%d", req.Type, req.Name, req.Int)
		return nil
	default:
		return fmt.Errorf("repository: storage doesn't support this type of metrics: %s", req.Type)
	}
}

func (s *Store) GetMetric(req *Metric) (*Metric, error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	switch req.Type {
	case "gauge":
		if val, ok := s.gaugeStore[req.Name]; ok {
			log.Printf("repository: storage provided metric type: %s, name: %s, value:%f", req.Type, req.Name, val)
			return &Metric{
				Float: val,
			}, nil
		} else {
			return nil, fmt.Errorf("repository: no metric in storage with provided name: %s", req.Name)
		}
	case "counter":
		if val, ok := s.counterStore[req.Name]; ok {
			log.Printf("repository: storage provided metric type: %s, name: %s, value:%d", req.Type, req.Name, val)
			return &Metric{
				Int: val,
			}, nil
		} else {
			return nil, fmt.Errorf("repository: no metric in storage with provided type: %s and name: %s", req.Type, req.Name)
		}
	default:
		return nil, fmt.Errorf("repository: storage doesn't support this type of metrics: %s", req.Type)
	}
}

func (s *Store) GetAllMetrics() []string {
	s.mux.Lock()
	defer s.mux.Unlock()
	metrics := make([]string, 0, len(s.gaugeStore)+len(s.counterStore))
	for metric := range s.gaugeStore {
		metrics = append(metrics, fmt.Sprintf("gauge:%s:%s", metric, strconv.FormatFloat(s.gaugeStore[metric], 'f', -1, 64)))
	}
	for metric := range s.counterStore {
		metrics = append(metrics, fmt.Sprintf("counter:%s:%d", metric, s.counterStore[metric]))
	}
	log.Printf("repository: storage provided all metric type: %v", metrics)
	return metrics
}
