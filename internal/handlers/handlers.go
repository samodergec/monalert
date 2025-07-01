package handlers

import (
	"fmt"
	"io"
	"log"
	"math"
	"monalert/internal/handlers/config"
	"monalert/internal/service"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func Serve(cfg config.Config, monalert Monalert) error {
	h := newHandlers(monalert)
	router := newRouter(h)
	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}
	if err := srv.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start server on %s: %w", cfg.ServerAddr, err)
	}
	return nil
}

func newRouter(h *handlers) chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.mainHandle)
	r.Route("/update", func(r chi.Router) {
		r.Post("/{metricType}/{metricName}/{metricValue}", h.metricUpdateHandler)
	})
	r.Get("/value/{metricType}/{metricName}", h.metricValueProvider)
	return r
}

type handlers struct {
	monalert Monalert
}

func newHandlers(monalert Monalert) *handlers {
	return &handlers{
		monalert: monalert,
	}
}

type Monalert interface {
	GaugeUpdate(req *service.GaugeUpdateRequest)
	CounterUpdate(req *service.CounterUpdateRequest)
	GetMetricValue(req *service.GetMetricValueRequest) (*service.GetMetricValueResponse, error)
	GetAllMetrics() []string
}

func (h *handlers) metricUpdateHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("content-type", "text/plain")
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "metricValue")
	if metricType == "gauge" {
		v, err := strconv.ParseFloat(metricValue, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			log.Print("error in converting metric value to float64")
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		h.monalert.GaugeUpdate(&service.GaugeUpdateRequest{
			Metric: metricName,
			Value:  v,
		})
	}
	if metricType == "counter" {
		v, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			log.Print("error in converting metric value to int64", err, v)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		h.monalert.CounterUpdate(&service.CounterUpdateRequest{
			Metric: metricName,
			Value:  v,
		})
	}
}

func (h *handlers) metricValueProvider(rw http.ResponseWriter, r *http.Request) {

	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	log.Printf("handler: received request for metric: %s", metricName)
	rw.Header().Set("content-type", "text/plain")
	val, err := h.monalert.GetMetricValue(&service.GetMetricValueRequest{
		MetricType: metricType,
		MetricName: metricName,
	})
	if err != nil {
		log.Printf("handler: error from service: %v", err)
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	rw.Write([]byte(val.MetricValue))
	return
}

func (h *handlers) mainHandle(rw http.ResponseWriter, r *http.Request) {
	io.WriteString(rw, strings.Join(h.monalert.GetAllMetrics(), ", "))
}
