package repository

import (
	"encoding/json"
	"fmt"
	"monalert/internal/logger"
	"monalert/internal/models"
	"os"
	"sync"

	"go.uber.org/zap"
)

type Store struct {
	mux          *sync.Mutex
	gaugeStore   map[string]float64
	counterStore map[string]int64
	filePath     string
}

func NewStore(filepath string) *Store {
	return &Store{
		mux:          &sync.Mutex{},
		gaugeStore:   make(map[string]float64),
		counterStore: make(map[string]int64),
		filePath:     filepath,
	}
}

func (s *Store) MetricUpdate(req *models.Metrics) (*models.Metrics, error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	switch req.MType {
	case "gauge":
		logger.Log.Debug("repository: storage updated metric request", zap.String("type", req.MType), zap.String("name", req.ID))
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

func (s *Store) GetAllMetrics() []models.Metrics {
	s.mux.Lock()
	defer s.mux.Unlock()
	allMetrics := make([]models.Metrics, 0, len(s.gaugeStore)+len(s.counterStore))
	for name, value := range s.gaugeStore {
		allMetrics = append(allMetrics, models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &value,
		})
	}
	for name, delta := range s.counterStore {
		allMetrics = append(allMetrics, models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &delta,
		})
	}
	logger.Log.Debug("repository: storage provided all metric")
	return allMetrics
}

func (s *Store) Persist() error {
	file, err := os.OpenFile(s.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := json.MarshalIndent(s.GetAllMetrics(), "", " ")
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		return err
	}
	logger.Log.Debug("data saved to file")
	return nil
}

func (s *Store) Restore() error {
	file, err := os.OpenFile(s.filePath, os.O_RDONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	data := make([]byte, 64)
	_, err = file.Read(data)
	if err != nil {
		return err
	}
	var allMetrics []models.Metrics
	if err := json.Unmarshal(data, &allMetrics); err != nil {
		return err
	}
	fmt.Println("data restored from file")
	return nil
}
