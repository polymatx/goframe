package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestDefaultCORS(t *testing.T) {
	t.Run("preflight OPTIONS request", func(t *testing.T) {
		handlerCalled := false
		wrapped := DefaultCORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		}))

		req := httptest.NewRequest(http.MethodOptions, "/resource", nil)
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected preflight status %d, got %d", http.StatusNoContent, w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("expected Access-Control-Allow-Origin '*', got %q", got)
		}
		if got := w.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPost) {
			t.Errorf("expected Access-Control-Allow-Methods to contain POST, got %q", got)
		}
		if handlerCalled {
			t.Error("expected preflight request to not reach the wrapped handler")
		}
	})

	t.Run("simple GET request", func(t *testing.T) {
		wrapped := DefaultCORS()(okHandler("data"))

		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("expected Access-Control-Allow-Origin '*', got %q", got)
		}
		if got := w.Body.String(); got != "data" {
			t.Errorf("expected body 'data', got %q", got)
		}
	})

	t.Run("request without Origin passes through untouched", func(t *testing.T) {
		wrapped := DefaultCORS()(okHandler("plain"))

		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("expected no Access-Control-Allow-Origin header, got %q", got)
		}
		if got := w.Body.String(); got != "plain" {
			t.Errorf("expected body 'plain', got %q", got)
		}
	})
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name         string
		config       CORSConfig
		method       string
		origin       string
		reqMethodHdr string // Access-Control-Request-Method for preflight
		wantStatus   int
		wantOrigin   string
		wantHeaders  map[string]string
	}{
		{
			name:       "allowed origin is echoed on simple request",
			config:     CORSConfig{AllowedOrigins: []string{"http://allowed.com"}},
			method:     http.MethodGet,
			origin:     "http://allowed.com",
			wantStatus: http.StatusOK,
			wantOrigin: "http://allowed.com",
		},
		{
			name:       "disallowed origin gets no CORS headers",
			config:     CORSConfig{AllowedOrigins: []string{"http://allowed.com"}},
			method:     http.MethodGet,
			origin:     "http://evil.com",
			wantStatus: http.StatusOK,
			wantOrigin: "",
		},
		{
			name:         "preflight with allowed origin",
			config:       CORSConfig{AllowedOrigins: []string{"http://allowed.com"}, MaxAge: 600},
			method:       http.MethodOptions,
			origin:       "http://allowed.com",
			reqMethodHdr: http.MethodPut,
			wantStatus:   http.StatusNoContent,
			wantOrigin:   "http://allowed.com",
			wantHeaders: map[string]string{
				"Access-Control-Max-Age": strconv.Itoa(600),
			},
		},
		{
			name: "credentials and exposed headers on simple request",
			config: CORSConfig{
				AllowedOrigins:   []string{"http://allowed.com"},
				AllowCredentials: true,
				ExposedHeaders:   []string{"X-Total-Count"},
			},
			method:     http.MethodGet,
			origin:     "http://allowed.com",
			wantStatus: http.StatusOK,
			wantOrigin: "http://allowed.com",
			wantHeaders: map[string]string{
				"Access-Control-Allow-Credentials": "true",
				"Access-Control-Expose-Headers":    "X-Total-Count",
			},
		},
		{
			name:         "preflight with disallowed method gets no allow headers",
			config:       CORSConfig{AllowedOrigins: []string{"http://allowed.com"}, AllowedMethods: []string{http.MethodGet}},
			method:       http.MethodOptions,
			origin:       "http://allowed.com",
			reqMethodHdr: http.MethodDelete,
			wantStatus:   http.StatusNoContent,
			wantOrigin:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := CORS(tt.config)(okHandler("ok"))

			req := httptest.NewRequest(tt.method, "/resource", nil)
			req.Header.Set("Origin", tt.origin)
			if tt.reqMethodHdr != "" {
				req.Header.Set("Access-Control-Request-Method", tt.reqMethodHdr)
			}
			w := httptest.NewRecorder()
			wrapped.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
				t.Errorf("expected Access-Control-Allow-Origin %q, got %q", tt.wantOrigin, got)
			}
			for k, want := range tt.wantHeaders {
				if got := w.Header().Get(k); got != want {
					t.Errorf("expected header %s=%q, got %q", k, want, got)
				}
			}
		})
	}
}
