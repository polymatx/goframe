package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// CORS middleware with custom configuration
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"*"}
	}
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{"*"}
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   config.AllowedOrigins,
		AllowedMethods:   config.AllowedMethods,
		AllowedHeaders:   config.AllowedHeaders,
		ExposedHeaders:   config.ExposedHeaders,
		AllowCredentials: config.AllowCredentials,
		MaxAge:           config.MaxAge,
	})

	return func(next http.Handler) http.Handler {
		return c.Handler(next)
	}
}

// DefaultCORS returns CORS middleware with default configuration
func DefaultCORS() func(http.Handler) http.Handler {
	return CORS(CORSConfig{})
}
