package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
)

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
}

type memStorage interface {
	gaugeUpdate()
	counterUpdate()
}

func (ms *MemStorage) gaugeUpdate(metric string, value float64) {
	ms.gauge[metric] = value
}

func (ms *MemStorage) counterUpdate(metric string, value int64) {
	ms.counter[metric] += value
}

func myMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			log.Println("hello from middleware")
			next(w, r) // Передаем запрос дальше
		} else {
			http.Error(w, "Only POST method is allowed", http.StatusBadRequest)
			return // Добавляем return, чтобы не выполнялся next()
		}
	}
}

func metricHandler(w http.ResponseWriter, r *http.Request) {
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
		ms.gaugeUpdate(name, v)
	}
	if metricType == "counter" {
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			log.Print("error in converting metric value to int64")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ms.counterUpdate(name, v)
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
		return "", "", "", errors.New("invalid url address")
	}

	return m[1], m[2], m[3], nil
}

var ms = NewMemStorage()

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

func main() {
	http.HandleFunc("/", defaultHandler)
	http.HandleFunc("/update/", myMiddleware(metricHandler))
	err := http.ListenAndServe(`:8080`, nil)
	if err != nil {
		panic(err)
	}
}
