package handlers

import (
	"fmt"
	"log"
	"math"
	handlersConfig "monalert/internal/handlers/config"
	"monalert/internal/service"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func Serve(cfg handlersConfig.Config, monalert Monalert) error {
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
	r.Get("/", h.handleMain)
	r.Route("/update", func(r chi.Router) {
		r.Post("/{metricType}/{metricName}/{metricValue}", h.handleMetricUpdate)
		r.Post("/{metricType}/{metricName}", h.handleIncompleteURL)
	})
	r.Get("/value/{metricType}/{metricName}", h.handleGetMetric)
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
	MetricUpdate(req *service.Metric) error
	GetMetric(req *service.Metric) (*service.Metric, error)
	GetAllMetrics() []string
}

func (h *handlers) handleMetricUpdate(rw http.ResponseWriter, r *http.Request) {
	//rw.Header().Set("content-type", "text/plain")
	log.Printf("handler: incoming request: %s %s", r.Method, r.URL.Path)
	log.Printf("handler: handleUpdate called")
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "metricValue")
	switch metricType {
	case "gauge":
		val, err := strconv.ParseFloat(metricValue, 64)
		if err != nil || math.IsNaN(val) || math.IsInf(val, 0) {
			log.Print("error in converting metric value to float64")
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		er := h.monalert.MetricUpdate(&service.Metric{
			Type:  metricType,
			Name:  metricName,
			Float: val,
		})
		if er != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.WriteHeader(http.StatusOK)
	case "counter":
		val, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			log.Print("error in converting metric value to int64", err, val)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		er := h.monalert.MetricUpdate(&service.Metric{
			Type: metricType,
			Name: metricName,
			Int:  val,
		})
		if er != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.WriteHeader(http.StatusOK)
	default:
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("unsupported metric type"))
	}
}

func (h *handlers) handleGetMetric(rw http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	//log.Printf("handler: received request for metric: %s", metricName)
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	val, err := h.monalert.GetMetric(&service.Metric{
		Type: metricType,
		Name: metricName,
	})
	if err != nil {
		log.Printf("handler: error from service: %v", err)
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	switch metricType {
	case "gauge":
		rw.Write([]byte(strconv.FormatFloat(val.Float, 'f', -1, 64)))
	case "counter":
		rw.Write([]byte(strconv.FormatInt(val.Int, 10)))
	}

}

func (h *handlers) handleMain(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("content-type", "text/html")
	rw.Write([]byte(strings.Join(h.monalert.GetAllMetrics(), ", ")))
}

func (h *handlers) handleIncompleteURL(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusBadRequest)
	rw.Write([]byte("unsupported URL"))
}
