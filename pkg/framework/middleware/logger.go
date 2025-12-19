package middleware

import (
	"bytes"
	"net/http"
	"time"

	"github.com/polymatx/goframe/pkg/framework"
	"github.com/sirupsen/logrus"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// Logger is a middleware that logs HTTP requests
func Logger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call next handler
		next(rw, r)

		// Log request details
		duration := time.Since(start)

		logrus.WithFields(logrus.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"query":      r.URL.RawQuery,
			"status":     rw.statusCode,
			"duration":   duration.Milliseconds(),
			"ip":         framework.RealIP(r),
			"user_agent": r.UserAgent(),
		}).Info("HTTP request")
	}
}
