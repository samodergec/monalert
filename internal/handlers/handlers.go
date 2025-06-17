package handlers

import (
	"errors"
	"fmt"
	"log"
	"monalert/internal/handlers/config"
	"monalert/internal/service"
	"net/http"
	"regexp"
	"strconv"
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

func newRouter(h *handlers) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", defaultHandler)
	mux.HandleFunc("/update/", h.myMiddleware(h.metricHandler))
	return mux
}

type Monalert interface {
	GaugeUpdate(req *service.GaugeUpdateRequest)
	CounterUpdate(req *service.CounterUpdateRequest)
}

type handlers struct {
	monalert Monalert
}

func newHandlers(monalert Monalert) *handlers {
	return &handlers{
		monalert: monalert,
	}
}

func (h *handlers) myMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			next(w, r) // Передаем запрос дальше
		} else {
			http.Error(w, "Only POST method is allowed", http.StatusBadRequest)
			return // Добавляем return, чтобы не выполнялся next()
		}
	}
}

func (h *handlers) metricHandler(w http.ResponseWriter, r *http.Request) {
	metricType, name, value, err := getMetricNameAndValue(w, r)
	w.Header().Set("content-type", "text/plain")
	log.Println(metricType, name, value, err)
	if err != nil {
		if name == "none" {
			http.Error(w, "no name for metric", http.StatusNotFound)
			return
		}
		http.Error(w, "Incorrect metric type", http.StatusBadRequest)
		return
	}

	if metricType == "gauge" {
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Print("error in converting metric value to float64")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.monalert.GaugeUpdate(&service.GaugeUpdateRequest{
			Metric: name,
			Value:  v,
		})
	}
	if metricType == "counter" {
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			log.Print("error in converting metric value to int64", err, v)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.monalert.CounterUpdate(&service.CounterUpdateRequest{
			Metric: name,
			Value:  v,
		})
	}
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
}

var validPath = regexp.MustCompile(`^/update/(gauge|counter)/([^/]+)/(\d+(\.\d*)?$)`)

func getMetricNameAndValue(w http.ResponseWriter, r *http.Request) (string, string, string, error) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	log.Println(m)
	if m == nil {
		matched, _ := regexp.MatchString(`(gauge|counter)/$`, r.URL.Path)
		if matched {
			w.WriteHeader(http.StatusNotFound)
			return "", "none", "", errors.New("no name for metric")
		}
		return "", "", "", errors.New("invalid url address " + r.URL.Path)
	}
	return m[1], m[2], m[3], nil
}
