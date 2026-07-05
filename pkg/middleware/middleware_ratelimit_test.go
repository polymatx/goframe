package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

// doRequest sends a GET request with the given client IP through the handler.
func doRequest(handler http.Handler, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.Header.Set("X-Real-IP", ip)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func TestRateLimit(t *testing.T) {
	// Refill rate is negligible so only the burst matters during the test.
	const burst = 3

	t.Run("allows burst then blocks with 429", func(t *testing.T) {
		wrapped := RateLimit(0.0001, burst)(okHandler("ok"))

		for i := 1; i <= burst; i++ {
			w := doRequest(wrapped, "10.0.0.1")
			if w.Code != http.StatusOK {
				t.Fatalf("request %d: expected status 200, got %d", i, w.Code)
			}
			if got := w.Body.String(); got != "ok" {
				t.Fatalf("request %d: expected body 'ok', got %q", i, got)
			}
		}

		w := doRequest(wrapped, "10.0.0.1")
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("expected status 429 after burst exceeded, got %d", w.Code)
		}
		if got := w.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", got)
		}
		if got := w.Body.String(); got != `{"error":"Rate limit exceeded"}` {
			t.Errorf("unexpected 429 body: %q", got)
		}
	})

	t.Run("limits are tracked per IP", func(t *testing.T) {
		wrapped := RateLimit(0.0001, burst)(okHandler("ok"))

		// Exhaust the budget for the first IP.
		for i := 0; i < burst; i++ {
			if w := doRequest(wrapped, "10.0.0.1"); w.Code != http.StatusOK {
				t.Fatalf("expected status 200 while within burst, got %d", w.Code)
			}
		}
		if w := doRequest(wrapped, "10.0.0.1"); w.Code != http.StatusTooManyRequests {
			t.Fatalf("expected first IP to be rate limited, got %d", w.Code)
		}

		// A different IP has its own budget.
		if w := doRequest(wrapped, "10.0.0.2"); w.Code != http.StatusOK {
			t.Errorf("expected second IP to be allowed, got %d", w.Code)
		}
	})

	t.Run("separate middleware instances have separate state", func(t *testing.T) {
		first := RateLimit(0.0001, 1)(okHandler("ok"))
		second := RateLimit(0.0001, 1)(okHandler("ok"))

		if w := doRequest(first, "10.0.0.3"); w.Code != http.StatusOK {
			t.Fatalf("expected first instance to allow, got %d", w.Code)
		}
		if w := doRequest(first, "10.0.0.3"); w.Code != http.StatusTooManyRequests {
			t.Fatalf("expected first instance to block, got %d", w.Code)
		}
		if w := doRequest(second, "10.0.0.3"); w.Code != http.StatusOK {
			t.Errorf("expected second instance to allow, got %d", w.Code)
		}
	})
}

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(5), 10)
	if rl == nil {
		t.Fatal("expected rate limiter to be created")
	}

	// The same IP must get the same limiter back.
	a := rl.getLimiter("1.2.3.4")
	b := rl.getLimiter("1.2.3.4")
	if a != b {
		t.Error("expected the same limiter instance for the same IP")
	}

	// A different IP gets its own limiter.
	c := rl.getLimiter("5.6.7.8")
	if a == c {
		t.Error("expected a different limiter instance for a different IP")
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(1), 1)
	rl.getLimiter("1.2.3.4")

	// Entry was just seen, cleanup must keep it.
	rl.cleanup()

	rl.mu.RLock()
	_, exists := rl.limiters["1.2.3.4"]
	rl.mu.RUnlock()
	if !exists {
		t.Error("expected recently seen entry to survive cleanup")
	}
}
