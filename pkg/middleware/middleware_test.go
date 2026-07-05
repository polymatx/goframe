package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func TestMain(m *testing.M) {
	// Silence logrus output (Recovery and Logger middleware log to the
	// standard logger); test hooks still receive entries.
	logrus.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// okHandler writes a fixed body with status 200.
func okHandler(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})
}

func TestRecovery(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.Handler
		wantStatus int
		wantBody   string
		wantCT     string
	}{
		{
			name: "panic with string returns 500",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("boom")
			}),
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"Internal Server Error"}`,
			wantCT:     "application/json",
		},
		{
			name: "panic with error returns 500",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(io.ErrUnexpectedEOF)
			}),
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"Internal Server Error"}`,
			wantCT:     "application/json",
		},
		{
			name:       "no panic passes through",
			handler:    okHandler("hello"),
			wantStatus: http.StatusOK,
			wantBody:   "hello",
		},
		{
			name: "no panic preserves custom status",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTeapot)
				_, _ = w.Write([]byte("teapot"))
			}),
			wantStatus: http.StatusTeapot,
			wantBody:   "teapot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := Recovery()(tt.handler)

			req := httptest.NewRequest(http.MethodGet, "/panic-test", nil)
			w := httptest.NewRecorder()

			// The middleware must not let the panic escape.
			wrapped.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if got := w.Body.String(); got != tt.wantBody {
				t.Errorf("expected body %q, got %q", tt.wantBody, got)
			}
			if tt.wantCT != "" {
				if got := w.Header().Get("Content-Type"); got != tt.wantCT {
					t.Errorf("expected Content-Type %q, got %q", tt.wantCT, got)
				}
			}
		})
	}
}

func TestLogger(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.Handler
		method     string
		target     string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "passes through 200 and body",
			handler:    okHandler("hello world"),
			method:     http.MethodGet,
			target:     "/logged?foo=bar",
			wantStatus: http.StatusOK,
			wantBody:   "hello world",
		},
		{
			name: "passes through custom status",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("created"))
			}),
			method:     http.MethodPost,
			target:     "/created",
			wantStatus: http.StatusCreated,
			wantBody:   "created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logrustest.NewGlobal()
			defer hook.Reset()

			wrapped := Logger()(tt.handler)

			req := httptest.NewRequest(tt.method, tt.target, nil)
			w := httptest.NewRecorder()
			wrapped.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if got := w.Body.String(); got != tt.wantBody {
				t.Errorf("expected body %q, got %q", tt.wantBody, got)
			}

			entry := hook.LastEntry()
			if entry == nil {
				t.Fatal("expected a log entry to be emitted")
			}
			if entry.Message != "HTTP request" {
				t.Errorf("expected message 'HTTP request', got %q", entry.Message)
			}
			if got := entry.Data["method"]; got != tt.method {
				t.Errorf("expected logged method %q, got %v", tt.method, got)
			}
			if got := entry.Data["status"]; got != tt.wantStatus {
				t.Errorf("expected logged status %d, got %v", tt.wantStatus, got)
			}
			if got := entry.Data["bytes"]; got != int64(len(tt.wantBody)) {
				t.Errorf("expected logged bytes %d, got %v", len(tt.wantBody), got)
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		remote  string
		want    string
	}{
		{
			name:    "CF-Connecting-IP has highest priority",
			headers: map[string]string{"CF-Connecting-IP": "1.1.1.1", "X-Forwarded-For": "2.2.2.2", "X-Real-IP": "3.3.3.3"},
			remote:  "4.4.4.4:1234",
			want:    "1.1.1.1",
		},
		{
			name:    "X-Forwarded-For beats X-Real-IP",
			headers: map[string]string{"X-Forwarded-For": "2.2.2.2", "X-Real-IP": "3.3.3.3"},
			remote:  "4.4.4.4:1234",
			want:    "2.2.2.2",
		},
		{
			name:    "X-Real-IP used when others missing",
			headers: map[string]string{"X-Real-IP": "3.3.3.3"},
			remote:  "4.4.4.4:1234",
			want:    "3.3.3.3",
		},
		{
			name:    "falls back to RemoteAddr",
			headers: nil,
			remote:  "4.4.4.4:1234",
			want:    "4.4.4.4:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remote
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			if got := getClientIP(req); got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
