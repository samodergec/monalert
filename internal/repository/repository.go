package repository

import "sync"

type Store struct {
	mux     *sync.Mutex
	gauge   map[string]float64
	counter map[string]int64
}

func NewStore() *Store {
	return &Store{
		mux:     &sync.Mutex{},
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

type GaugeUpdateRequest struct {
	Metric string
	Value  float64
}

type CounterUpdateRequest struct {
	Metric string
	Value  int64
}

func (s *Store) GaugeUpdate(req *GaugeUpdateRequest) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.gauge[req.Metric] = req.Value
}

func (s *Store) CounterUpdate(req *CounterUpdateRequest) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.counter[req.Metric] += req.Value
}
