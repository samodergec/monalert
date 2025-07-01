package service

import (
	"fmt"
	"log"
	"monalert/internal/repository"
)

type Repository interface {
	GaugeUpdate(req *repository.GaugeUpdateRequest)
	CounterUpdate(req *repository.CounterUpdateRequest)
	GetMetricValue(req *repository.GetMetricValueRequest) (*repository.GetMetricValueResponse, error)
	GetAllMetrics() []string
}

type Monalert struct {
	store Repository
}

func NewMonalert(store Repository) *Monalert {
	return &Monalert{
		store: store,
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

func (m *Monalert) GaugeUpdate(req *GaugeUpdateRequest) {
	m.store.GaugeUpdate(&repository.GaugeUpdateRequest{
		MetricName:  req.Metric,
		MetricValue: req.Value,
	})
}

func (m *Monalert) CounterUpdate(req *CounterUpdateRequest) {
	m.store.CounterUpdate(&repository.CounterUpdateRequest{
		MetricName:  req.Metric,
		MetricValue: req.Value,
	})
}

type GetMetricValueRequest struct {
	MetricType string
	MetricName string
}

type GetMetricValueResponse struct {
	MetricValue string
}

func (m *Monalert) GetMetricValue(req *GetMetricValueRequest) (*GetMetricValueResponse, error) {
	log.Printf("service: request: %+v", req)

	res, err := m.store.GetMetricValue(&repository.GetMetricValueRequest{
		MetricType: req.MetricType,
		MetricName: req.MetricName,
	})
	if err != nil {
		return nil, fmt.Errorf("service: failed to get metric value: %w", err)
	}
	log.Printf("service: got value from repo: %+v", res)

	return &GetMetricValueResponse{
		MetricValue: res.MetricValue,
	}, nil
}

func (m *Monalert) GetAllMetrics() []string {
	metrics := m.store.GetAllMetrics()
	if len(metrics) == 0 {
		return []string{}
	}
	return metrics
}
