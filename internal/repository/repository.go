package repository

import (
	"fmt"
	"monalert/internal/logger"
	"monalert/internal/models"
	"strconv"
	"sync"

	"go.uber.org/zap"
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

func (s *Store) MetricUpdate(req *models.Metrics) (*models.Metrics, error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	switch req.MType {
	case "gauge":
		logger.Log.Debug("repository: storage updated metric request1", zap.String("type", req.MType), zap.String("name", req.ID))
		s.gaugeStore[req.ID] = *req.Value
		val := s.gaugeStore[req.ID]
		logger.Log.Debug("repository: storage updated metric", zap.String("type", req.MType), zap.String("name", req.ID), zap.Float64("value:", val))
		return &models.Metrics{
			ID:    req.ID,
			MType: "gauge",
			Value: &val,
		}, nil
	case "counter":
		s.counterStore[req.ID] += *req.Delta
		val := s.counterStore[req.ID]
		logger.Log.Debug("repository: storage updated metric", zap.String("type", req.MType), zap.String("name", req.ID), zap.Int64("value:", val))
		return &models.Metrics{
			ID:    req.ID,
			MType: "counter",
			Delta: &val,
		}, nil
	default:
		return nil, fmt.Errorf("repository: storage doesn't support this type of metrics: %s", req.MType)
	}
}

func (s *Store) GetMetric(req *models.Metrics) (*models.Metrics, error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	switch req.MType {
	case "gauge":
		if val, ok := s.gaugeStore[req.ID]; ok {
			logger.Log.Debug("repository: storage provided metric", zap.String("type", req.MType), zap.String("name", req.ID), zap.Float64("value:", val))
			return &models.Metrics{
				ID:    req.ID,
				MType: "gauge",
				Value: &val,
			}, nil
		} else {
			return nil, fmt.Errorf("repository: no metric in storage with provided name: %s", req.ID)
		}
	case "counter":
		if val, ok := s.counterStore[req.ID]; ok {
			logger.Log.Debug("repository: storage provided metric", zap.String("type", req.MType), zap.String("name", req.ID), zap.Int64("value:", val))
			return &models.Metrics{
				ID:    req.ID,
				MType: "counter",
				Delta: &val,
			}, nil
		} else {
			return nil, fmt.Errorf("repository: no metric in storage with provided type: %s and name: %s", req.MType, req.ID)
		}
	default:
		return nil, fmt.Errorf("repository: storage doesn't support this type of metrics: %s", req.MType)
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
	logger.Log.Debug("repository: storage provided all metric", zap.Any("metrics:", metrics))
	return metrics
}
