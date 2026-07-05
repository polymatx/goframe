package healthz

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	// The check handler logs failing checks via logrus; keep test output clean.
	logrus.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// checkerFunc adapts a plain function to the Healthy interface.
type checkerFunc func(ctx context.Context) error

func (f checkerFunc) Health(ctx context.Context) error {
	return f(ctx)
}

// resetChecks clears the package-level checker registry between tests.
func resetChecks() {
	lock.Lock()
	defer lock.Unlock()
	all = nil
}

func registeredCount() int {
	lock.RLock()
	defer lock.RUnlock()
	return len(all)
}

// newHealthzRouter builds a mux router with the healthz route mounted,
// mirroring how RegisterRoute wires it in production.
func newHealthzRouter() *mux.Router {
	m := mux.NewRouter()
	route{}.Routes(m)
	return m
}

func TestRegister(t *testing.T) {
	resetChecks()
	defer resetChecks()

	Register(checkerFunc(func(ctx context.Context) error { return nil }))
	if got := registeredCount(); got != 1 {
		t.Errorf("expected 1 registered checker, got %d", got)
	}

	// Variadic registration appends all checkers.
	Register(
		checkerFunc(func(ctx context.Context) error { return nil }),
		checkerFunc(func(ctx context.Context) error { return nil }),
	)
	if got := registeredCount(); got != 3 {
		t.Errorf("expected 3 registered checkers, got %d", got)
	}
}

func TestRegister_Concurrent(t *testing.T) {
	resetChecks()
	defer resetChecks()

	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Register(checkerFunc(func(ctx context.Context) error { return nil }))
		}()
	}
	wg.Wait()

	if got := registeredCount(); got != n {
		t.Errorf("expected %d registered checkers, got %d", n, got)
	}
}

func TestHealthzEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		checkers     []Healthy
		wantStatus   int
		bodyContains []string
	}{
		{
			name:       "no registered checks is healthy",
			checkers:   nil,
			wantStatus: http.StatusOK,
		},
		{
			name: "all checks healthy",
			checkers: []Healthy{
				checkerFunc(func(ctx context.Context) error { return nil }),
				checkerFunc(func(ctx context.Context) error { return nil }),
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "one failing check returns 500",
			checkers: []Healthy{
				checkerFunc(func(ctx context.Context) error { return nil }),
				checkerFunc(func(ctx context.Context) error { return errors.New("db down") }),
			},
			wantStatus:   http.StatusInternalServerError,
			bodyContains: []string{"db down"},
		},
		{
			name: "all errors are reported",
			checkers: []Healthy{
				checkerFunc(func(ctx context.Context) error { return errors.New("db down") }),
				checkerFunc(func(ctx context.Context) error { return errors.New("cache down") }),
			},
			wantStatus:   http.StatusInternalServerError,
			bodyContains: []string{"db down", "cache down"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetChecks()
			defer resetChecks()
			Register(tt.checkers...)

			router := newHealthzRouter()
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if got := w.Header().Get("time"); got == "" {
				t.Error("expected 'time' response header to be set")
			}

			body := w.Body.String()
			for _, want := range tt.bodyContains {
				if !strings.Contains(body, want) {
					t.Errorf("expected body to contain %q, got %q", want, body)
				}
			}

			if tt.wantStatus == http.StatusOK {
				var resp struct {
					Time string `json:"time"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("expected valid JSON body, got %q: %v", body, err)
				}
				if resp.Time == "" {
					t.Error("expected non-empty time field in JSON response")
				}
			}
		})
	}
}

func TestHealthzEndpoint_ChecksReceiveRequestContext(t *testing.T) {
	resetChecks()
	defer resetChecks()

	type ctxKey struct{}
	var gotValue any
	Register(checkerFunc(func(ctx context.Context) error {
		gotValue = ctx.Value(ctxKey{})
		return nil
	}))

	router := newHealthzRouter()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxKey{}, "marker"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if gotValue != "marker" {
		t.Errorf("expected checker to receive the request context, got %v", gotValue)
	}
}

func TestHealthzEndpoint_Methods(t *testing.T) {
	tests := []struct {
		method     string
		wantStatus int
	}{
		{http.MethodGet, http.StatusOK},
		{http.MethodHead, http.StatusOK},
		{http.MethodPost, http.StatusMethodNotAllowed},
		{http.MethodDelete, http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			resetChecks()
			defer resetChecks()

			router := newHealthzRouter()
			req := httptest.NewRequest(tt.method, "/healthz", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d for %s, got %d", tt.wantStatus, tt.method, w.Code)
			}
		})
	}
}

func TestRegisterRoute(t *testing.T) {
	// RegisterRoute only appends to the framework router registry; it must
	// not panic and must not start anything.
	RegisterRoute()
}
