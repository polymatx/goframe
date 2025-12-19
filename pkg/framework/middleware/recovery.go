package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/polymatx/goframe/pkg/framework"
	"github.com/sirupsen/logrus"
)

// Recovery middleware recovers from panics and returns a 500 error
func Recovery(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
					"stack": string(debug.Stack()),
					"path":  r.URL.Path,
				}).Error("Panic recovered")

				_ = framework.JSON(w,
					http.StatusInternalServerError,
					struct {
						Error string `json:"error"`
					}{
						Error: http.StatusText(http.StatusInternalServerError),
					},
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
