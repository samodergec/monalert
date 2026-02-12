package service

import (
	"fmt"
	"monalert/internal/logger"
	"monalert/internal/models"

	"go.uber.org/zap"
)

type Repository interface {
	MetricUpdate(req *models.Metrics) (*models.Metrics, error)
	GetMetric(req *models.Metrics) (*models.Metrics, error)
	GetAllMetrics() []models.Metrics
	Persist() error
}

type Monalert struct {
	store          Repository
	persistentMode bool
}

func NewMonalert(store Repository, persistentMode bool) *Monalert {
	return &Monalert{
		store:          store,
		persistentMode: persistentMode,
	}
}

func (m *Monalert) MetricUpdate(req *models.Metrics) (*models.Metrics, error) {
	logger.Log.Debug("service: request for metric update")
	resp, err := m.store.MetricUpdate(&models.Metrics{
		ID:    req.ID,
		MType: req.MType,
		Value: req.Value,
		Delta: req.Delta,
	})
	if err != nil {
		logger.Log.Debug("service: failed for metric update", zap.Error(err))
		return nil, fmt.Errorf("service: failed to update metric value: %w", err)
	}
	if m.persistentMode {
		err := m.store.Persist()
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (m *Monalert) GetMetric(req *models.Metrics) (*models.Metrics, error) {
	logger.Log.Debug("service: request for get metric")
	resp, err := m.store.GetMetric(&models.Metrics{
		ID:    req.ID,
		MType: req.MType,
		Value: req.Value,
		Delta: req.Delta,
	})
	if err != nil {
		logger.Log.Debug("service: failed to get metric value", zap.Error(err))
		return nil, fmt.Errorf("service: failed to get metric value: %w", err)
	}
	logger.Log.Debug("service: got value from repo", zap.Any("resp:", resp))
	return &models.Metrics{
		ID:    resp.ID,
		MType: resp.MType,
		Value: resp.Value,
		Delta: resp.Delta,
	}, nil
}

func (m *Monalert) GetAllMetrics() []models.Metrics {
	metrics := m.store.GetAllMetrics()
	return metrics
}
