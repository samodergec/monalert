package handlers

import (
	"monalert/internal/service"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockMonalert struct {
	gaugeCalled   bool
	counterCalled bool
	lastGauge     *service.GaugeUpdateRequest
	lastCounter   *service.CounterUpdateRequest
}

func (m *mockMonalert) GaugeUpdate(req *service.GaugeUpdateRequest) {
	m.gaugeCalled = true
	m.lastGauge = req
}

func (m *mockMonalert) CounterUpdate(req *service.CounterUpdateRequest) {
	m.counterCalled = true
	m.lastCounter = req
}

func Test_defaultHandler(t *testing.T) {
	/* 	type args struct {
		w http.ResponseWriter
		r *http.Request
	} */
	tests := []struct {
		name         string
		method       string
		url          string
		expectedCode int
		// expectedBody string
	}{
		// TODO: Add test cases.
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
			expectedCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.url, http.NoBody)
			w := httptest.NewRecorder()
			defaultHandler(w, r)
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func Test_metricHandler(t *testing.T) {
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
			w := httptest.NewRecorder()
			handler := h.myMiddleware(h.metricHandler)
			handler(w, r)
			resp := w.Result()
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.Equal(t, tt.expectGauge, mock.gaugeCalled)
			assert.Equal(t, tt.expectCounter, mock.counterCalled)
		})
	}
}
