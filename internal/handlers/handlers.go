package handlers

import (
	"fmt"
	"log"
	"math"
	"monalert/internal/logger"
	"monalert/internal/service"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func Serve(flagServerAddr string, monalert Monalert) error {
	h := newHandlers(monalert)
	router := newRouter(h)
	srv := &http.Server{
		Addr:    flagServerAddr,
		Handler: router,
	}
	if err := srv.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start server on %s: %w", flagServerAddr, err)
	}
	return nil
}

func newRouter(h *handlers) chi.Router {
	r := chi.NewRouter()
	r.Use(MyLogger())
	r.Get("/", h.handleMain)
	r.Route("/update", func(r chi.Router) {
		r.Post("/{metricType}/{metricName}/{metricValue}", h.handleMetricUpdate)
		r.Post("/{metricType}/{metricName}", h.handleIncompleteURL)
	})
	r.Get("/value/{metricType}/{metricName}", h.handleGetMetric)
	return r
}

type (
	// берём структуру для хранения сведений об ответе
	responseData struct {
		status int
		size   int
	}

	// добавляем реализацию http.ResponseWriter
	loggingResponseWriter struct {
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseData        *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}

func MyLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			responseData := &responseData{
				status: 0,
				size:   0,
			}
			lw := loggingResponseWriter{
				ResponseWriter: w, // встраиваем оригинальный http.ResponseWriter
				responseData:   responseData,
			}

			method := r.Method
			uri := r.URL.Path
			next.ServeHTTP(&lw, r)
			duration := time.Since(start)
			logger.Log.Info("got incoming HTTP request",
				zap.String("method", method),
				zap.String("path", uri),
				zap.String("duration", duration.String()),
				zap.String("status", strconv.Itoa(responseData.status)),
				zap.String("size", strconv.Itoa(responseData.size)),
			)
		})
	}
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
	logger.Log.Debug("handler: incoming request in: handleUpdate", zap.String("method", r.Method), zap.String("path", r.URL.Path))
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
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	val, err := h.monalert.GetMetric(&service.Metric{
		Type: metricType,
		Name: metricName,
	})
	if err != nil {
		logger.Log.Debug("handler: error from service", zap.Error(err))
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
