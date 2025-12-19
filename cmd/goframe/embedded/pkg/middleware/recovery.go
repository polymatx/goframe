package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

// Recovery middleware recovers from panics
func Recovery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logrus.WithFields(logrus.Fields{
						"error": err,
						"stack": string(debug.Stack()),
						"path":  r.URL.Path,
					}).Error("Panic recovered")

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":"Internal Server Error"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
