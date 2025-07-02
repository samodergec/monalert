package service

import (
	"fmt"
	"log"
	"monalert/internal/repository"
)

type Repository interface {
	MetricUpdate(req *repository.Metric) error
	GetMetric(req *repository.Metric) (*repository.Metric, error)
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

type Metric struct {
	Name  string
	Type  string
	Float float64
	Int   int64
}

func (m *Monalert) MetricUpdate(req *Metric) error {
	log.Printf("service: request for metric update: %+v", req)
	err := m.store.MetricUpdate(&repository.Metric{
		Name:  req.Name,
		Type:  req.Type,
		Float: req.Float,
		Int:   req.Int,
	})
	if err != nil {
		log.Printf("service: got error from repository:%s", err)
		return fmt.Errorf("service: failed to update metric value: %w", err)
	}
	return nil
}

func (m *Monalert) GetMetric(req *Metric) (*Metric, error) {
	log.Printf("service: request for get metric: %+v", req)
	resp, err := m.store.GetMetric(&repository.Metric{
		Type: req.Type,
		Name: req.Name,
	})
	if err != nil {
		log.Printf("service: got error from repository:%s", err)
		return nil, fmt.Errorf("service: failed to get metric value: %w", err)
	}
	log.Printf("service: got value from repo: %+v", resp)
	return &Metric{
		Float: resp.Float,
		Int:   resp.Int,
	}, nil
}

func (m *Monalert) GetAllMetrics() []string {
	metrics := m.store.GetAllMetrics()
	if len(metrics) == 0 {
		return []string{}
	}
	return metrics
}
