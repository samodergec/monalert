package service

import (
	"monalert/internal/repository"
)

type Repository interface{
	GaugeUpdate(req *repository.GaugeUpdateRequest)
	CounterUpdate(req *repository.CounterUpdateRequest)
}

type Monalert struct{
	store Repository
}

func NewMonalert(store Repository) *Monalert{
	return &Monalert{
		store: store,
	}
}

type GaugeUpdateRequest struct{
	Metric string
	Value float64
}

type CounterUpdateRequest struct{
	Metric string
	Value int64
}

func (m *Monalert) GaugeUpdate(req *GaugeUpdateRequest){
	m.store.GaugeUpdate(&repository.GaugeUpdateRequest{
		Metric: req.Metric,
		Value: req.Value,
	})
}

func (m *Monalert) CounterUpdate(req *CounterUpdateRequest){
	m.store.CounterUpdate(&repository.CounterUpdateRequest{
		Metric: req.Metric,
		Value: req.Value,
	})
}