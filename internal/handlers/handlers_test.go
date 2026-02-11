package handlers

import (
	"errors"
	"io"
	"monalert/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMonalert struct {
}

func (m *mockMonalert) MetricUpdate(req *models.Metrics) (*models.Metrics, error) {
	return req, nil
}

func (m *mockMonalert) GetMetric(req *models.Metrics) (*models.Metrics, error) {
	switch req.MType {
	case "gauge":
		val := 42.5
		return &models.Metrics{Value: &val}, nil
	case "counter":
		var val int64 = 100
		return &models.Metrics{Delta: &val}, nil
	default:
		return nil, errors.New("service: failed to get metric value")
	}
}

func (m *mockMonalert) GetAllMetrics() []models.Metrics {
	value := 1.2
	var delta int64 = 1
	return []models.Metrics{
		{
			ID:    "metric1",
			MType: "gauge",
			Value: &value,
		},
		{
			ID:    "metric2",
			MType: "counter",
			Delta: &delta,
		},
	}
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string) (*http.Response, string) {
	t.Helper()

	req, err := http.NewRequest(method, ts.URL+path, http.NoBody)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestRouter(t *testing.T) {
	mock := &mockMonalert{}
	h := newHandlers(mock)
	ts := httptest.NewServer(newRouter(h))
	defer ts.Close()
	tests := []struct {
		name         string
		method       string
		url          string
		expectedCode int
	}{
		{
			name:         "1 valid gauge",
			method:       http.MethodPost,
			url:          "/update/gauge/temperature/42.5",
			expectedCode: http.StatusOK,
		},
		{
			name:         "2 valid counter",
			method:       http.MethodPost,
			url:          "/update/counter/test/42",
			expectedCode: http.StatusOK,
		},
		{
			name:         "3 Valid get",
			method:       http.MethodGet,
			url:          "/value/gauge/temperature",
			expectedCode: http.StatusOK,
		},
		{
			name:         "4 Invalid path",
			method:       http.MethodPost,
			url:          "/update/foo/bar",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "4.1 Invalid method",
			method:       http.MethodPost,
			url:          "/update/foo/bar/10",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "5 Invalid gauge metric value (NaN)",
			method:       http.MethodPost,
			url:          "/update/gauge/temperature/nan",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "6 Invalid gauge metric value (string)",
			method:       http.MethodPost,
			url:          "/update/gauge/temperature/nat",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "POST root",
			method:       http.MethodPost,
			url:          "/",
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			name:         "GET root",
			method:       http.MethodGet,
			url:          "/",
			expectedCode: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := testRequest(t, ts, tt.method, tt.url)
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			/* r := httptest.NewRequest(tt.method, tt.url, http.NoBody)
			rw := httptest.NewRecorder()
			mock := &mockMonalert{}
			h := newHandlers(mock)
			h.mainHandle(rw, r)
			assert.Equal(t, tt.expectedCode, rw.Code) */
		})
	}
}

/*
	func TestHandler_mainHandle(t *testing.T) {
		tests := []struct {
			name         string
			method       string
			url          string
			expectedCode int
			// expectedBody string
		}{
			{
				name:         "1 POST Root",
				method:       http.MethodPost,
				url:          "/",
				expectedCode: http.StatusBadRequest,
			},
			{
				name:         "2 GET root",
				method:       http.MethodGet,
				url:          "/",
				expectedCode: http.StatusOK,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(tt.method, tt.url, http.NoBody)
				rw := httptest.NewRecorder()
				mock := &mockMonalert{}
				h := newHandlers(mock)
				h.mainHandle(rw, r)
				assert.Equal(t, tt.expectedCode, rw.Code)
			})
		}
	}

	func Test_metricUpdateHandler(t *testing.T) {
		tests := []struct {
			name          string
			method        string
			url           string
			expectedCode  int
			expectGauge   bool
			expectCounter bool
			// expectedBody string
		}{
			{
				name:         "1 valid gauge",
				method:       http.MethodPost,
				url:          "/update/gauge/temperature/42.5",
				expectedCode: http.StatusOK,
				expectGauge:  true,
			},
			{
				name:          "2 valid counter",
				method:        http.MethodPost,
				url:           "/update/counter/test/42",
				expectedCode:  http.StatusOK,
				expectCounter: true,
			},
			{
				name:         "3 Invalid method",
				method:       http.MethodGet,
				url:          "/update/gauge/temperature/42.5",
				expectedCode: http.StatusBadRequest,
			},
			{
				name:         "4 Invalid path",
				method:       http.MethodPost,
				url:          "/update/foo/bar",
				expectedCode: http.StatusBadRequest,
			},
			{
				name:         "5 Invalid metric value (non-numeric)",
				method:       http.MethodPost,
				url:          "/update/gauge/temperature/nan",
				expectedCode: http.StatusBadRequest,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mock := &mockMonalert{}
				h := newHandlers(mock)
				r := httptest.NewRequest(tt.method, tt.url, http.NoBody)
				rw := httptest.NewRecorder()
				h.metricUpdateHandler(rw, r)
				resp := rw.Result()
				assert.Equal(t, tt.expectedCode, resp.StatusCode)
				assert.Equal(t, tt.expectGauge, mock.gaugeCalled)
				assert.Equal(t, tt.expectCounter, mock.counterCalled)
			})
		}
	}
*/
