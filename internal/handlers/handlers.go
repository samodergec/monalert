package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"monalert/internal/compress"
	"monalert/internal/logger"
	"monalert/internal/models"
	"monalert/internal/service"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func Serve(flagServerAddr string, monalert *service.Monalert) error {
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
	r.Use(gzipMiddleware())
	r.Get("/", h.handleMain)
	r.Route("/update", func(r chi.Router) {
		r.Post("/", h.handleMetricUpdate)
		r.Post("/{metricType}/{metricName}/{metricValue}", h.handleMetricUpdate)
		r.Post("/{metricType}/{metricName}", h.handleIncompleteURL)
	})
	r.Get("/value/{metricType}/{metricName}", h.handleGetMetric)
	r.Post("/value/", h.handleGetMetricJSON)
	return r
}

type (
	// Берём структуру для хранения сведений об ответе.
	responseData struct {
		status int
		size   int
	}

	// Добавляем реализацию http.ResponseWriter.
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
			host := r.Host
			next.ServeHTTP(&lw, r)

			status := responseData.status
			if status == 0 {
				status = http.StatusOK
			}

			duration := time.Since(start)
			logger.Log.Info("got incoming HTTP request",
				zap.String("method", method),
				zap.String("host", host),
				zap.String("path", uri),
				zap.String("duration", duration.String()),
				zap.String("status", strconv.Itoa(status)),
				zap.String("size", strconv.Itoa(responseData.size)),
			)
		})
	}
}

type Service interface {
	MetricUpdate(req *models.Metrics) (*models.Metrics, error)
	GetMetric(req *models.Metrics) (*models.Metrics, error)
	GetAllMetrics() []models.Metrics
}

type handlers struct {
	monalert Service
}

func newHandlers(monalert Service) *handlers {
	return &handlers{
		monalert: monalert,
	}
}

func gzipMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
			// который будем передавать следующей функции
			ow := w

			// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
			acceptEncoding := r.Header.Get("Accept-Encoding")
			supportsGzip := strings.Contains(acceptEncoding, "gzip")
			if supportsGzip {
				// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
				cw := compress.NewCompressWriter(w)
				// меняем оригинальный http.ResponseWriter на новый
				ow = cw
				// не забываем отправить клиенту все сжатые данные после завершения middleware
				ow.Header().Set("Content-Encoding", "gzip")
				ow.Header().Del("Content-Length")
				defer cw.Close()
			}

			// проверяем, что клиент отправил серверу сжатые данные в формате gzip
			contentEncoding := r.Header.Get("Content-Encoding")
			sendsGzip := strings.Contains(contentEncoding, "gzip")
			if sendsGzip {
				// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
				cr, err := compress.NewCompressReader(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				// меняем тело запроса на новое
				r.Body = cr
				defer cr.Close()
			}

			// передаём управление хендлеру
			next.ServeHTTP(ow, r)
		})
	}
}

func (h *handlers) handleMetricUpdate(w http.ResponseWriter, r *http.Request) {
	logger.Log.Debug("handleMetricUpdate: incoming request", zap.String("method", r.Method), zap.String("path", r.URL.Path))

	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "metricValue")

	if metricType != "" && metricName != "" && metricValue != "" {
		switch metricType {
		case "gauge":
			val, err := strconv.ParseFloat(metricValue, 64)
			if err != nil || math.IsNaN(val) || math.IsInf(val, 0) {
				log.Print("error in converting metric value to float64")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, err = h.monalert.MetricUpdate(&models.Metrics{
				MType: metricType,
				ID:    metricName,
				Value: &val,
			})
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		case "counter":
			val, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				log.Print("error in converting metric value to int64", err, val)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, err = h.monalert.MetricUpdate(&models.Metrics{
				MType: metricType,
				ID:    metricName,
				Delta: &val,
			})
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		default:
			http.Error(w, "unsupported metric type", http.StatusBadRequest)
			return
		}
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "Unsupported content type", http.StatusUnsupportedMediaType)
		return
	}

	var req models.Metrics

	logger.Log.Debug("decoding request")
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err), zap.Any("decoded json:", &req))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if req.ID == "" || req.MType == "" || (req.MType == "counter" && req.Delta == nil) || (req.MType == "gauge" && req.Value == nil) {
		logger.Log.Debug("empty fields in JSON body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp, err := h.monalert.MetricUpdate(&models.Metrics{
		MType: req.MType,
		ID:    req.ID,
		Value: req.Value,
		Delta: req.Delta,
	})

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// сериализуем ответ сервера
	enc := json.NewEncoder(w)
	logger.Log.Debug("headers before encode", zap.Any("headers", w.Header()))
	if err := enc.Encode(resp); err != nil {
		logger.Log.Error("error encoding response", zap.Error(err))
		return
	}
	logger.Log.Debug("sending HTTP 200 response")
}

func (h *handlers) handleGetMetricJSON(w http.ResponseWriter, r *http.Request) {
	logger.Log.Debug("handleGetMetric: incoming request", zap.String("method", r.Method), zap.String("path", r.URL.Path))
	var req models.Metrics
	logger.Log.Debug("decoding request")
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Error("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp, err := h.monalert.GetMetric(&models.Metrics{
		MType: req.MType,
		ID:    req.ID,
	})
	if err != nil {
		logger.Log.Error("handler: error from service", zap.Error(err))
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// сериализуем ответ сервера
	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		logger.Log.Error("error encoding response", zap.Error(err))
		return
	}
	logger.Log.Debug("sending HTTP 200 response")
}

func (h *handlers) handleGetMetric(rw http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	val, err := h.monalert.GetMetric(&models.Metrics{
		MType: metricType,
		ID:    metricName,
	})
	if err != nil {
		logger.Log.Debug("handler: error from service", zap.Error(err))
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	switch metricType {
	case "gauge":
		// TODO add check for *val.Value nil
		//nolint:gosec // write error is not actionable in HTTP handler
		rw.Write([]byte(strconv.FormatFloat(*val.Value, 'f', -1, 64)))
	case "counter":
		// TODO add check for *val.Value nil
		//nolint:gosec // write error is not actionable in HTTP handler
		rw.Write([]byte(strconv.FormatInt(*val.Delta, 10)))
	}
}

func (h *handlers) handleMain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html")
	data, err := json.MarshalIndent(h.monalert.GetAllMetrics(), "", " ")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err = w.Write(data)
	if err != nil {
		logger.Log.Error("handleMain: write error", zap.Error(err))
	}
}

func (h *handlers) handleIncompleteURL(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "unsupported URL", http.StatusBadRequest)
}
