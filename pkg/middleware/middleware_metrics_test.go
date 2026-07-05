package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetrics(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.Handler
		wantStatus int
		wantBody   string
	}{
		{
			name:       "passes through 200 response",
			handler:    okHandler("metrics ok"),
			wantStatus: http.StatusOK,
			wantBody:   "metrics ok",
		},
		{
			name: "passes through error status",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			}),
			wantStatus: http.StatusNotFound,
			wantBody:   "not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := Metrics()(tt.handler)

			req := httptest.NewRequest(http.MethodGet, "/observed", nil)
			w := httptest.NewRecorder()
			wrapped.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if got := w.Body.String(); got != tt.wantBody {
				t.Errorf("expected body %q, got %q", tt.wantBody, got)
			}
		})
	}
}

func TestMetricsHandler(t *testing.T) {
	// Drive a request through the Metrics middleware first so the counters
	// have at least one observation with a unique, recognizable path.
	const probePath = "/metrics-probe-path"
	wrapped := Metrics()(okHandler("ok"))
	req := httptest.NewRequest(http.MethodGet, probePath, nil)
	wrapped.ServeHTTP(httptest.NewRecorder(), req)

	handler := MetricsHandler()
	if handler == nil {
		t.Fatal("expected metrics handler to be non-nil")
	}

	mreq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, mreq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("expected prometheus text format content type, got %q", ct)
	}

	body := w.Body.String()
	for _, want := range []string{
		"http_requests_total",
		"http_request_duration_seconds",
		`path="` + probePath + `"`,
		`method="GET"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected metrics output to contain %q", want)
		}
	}
}
