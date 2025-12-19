package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter holds rate limiter instances per IP
type RateLimiter struct {
	limiters    map[string]*rateLimiterEntry
	mu          sync.RWMutex
	rate        rate.Limit
	burst       int
	cleanupOnce sync.Once
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(r rate.Limit, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rate:     r,
		burst:    burst,
	}
	return rl
}

// startCleanup starts a background cleanup goroutine (only once)
func (rl *RateLimiter) startCleanup() {
	rl.cleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				rl.cleanup()
			}
		}()
	})
}

// cleanup removes stale rate limiter entries
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	threshold := time.Now().Add(-3 * time.Minute)
	for ip, entry := range rl.limiters {
		if entry.lastSeen.Before(threshold) {
			delete(rl.limiters, ip)
		}
	}
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
		entry = &rateLimiterEntry{
			limiter:  rate.NewLimiter(rl.rate, rl.burst),
			lastSeen: time.Now(),
		}
		rl.limiters[ip] = entry
	} else {
		entry.lastSeen = time.Now()
	}

	return entry.limiter
}

// RateLimit middleware limits requests per IP
func RateLimit(requestsPerSecond float64, burst int) func(http.Handler) http.Handler {
	rl := NewRateLimiter(rate.Limit(requestsPerSecond), burst)
	rl.startCleanup()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			limiter := rl.getLimiter(ip)

			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"Rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
