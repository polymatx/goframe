package router

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/polymatx/goframe/pkg/framework/middleware"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	once              = sync.Once{}
	globalMiddlewares []GlobalMiddleware
	all               []Router // Router registry
)

// GlobalMiddleware interface for middleware that applies to all routes
type GlobalMiddleware interface {
	Handler(http.HandlerFunc) http.HandlerFunc
}

// Router interface for route registration
type Router interface {
	Routes(route *mux.Router)
}

// RegisterGlobalMiddleware registers a middleware to be applied to all routes
func RegisterGlobalMiddleware(m GlobalMiddleware) {
	globalMiddlewares = append(globalMiddlewares, m)
}

// Register registers a router
func Register(r Router) {
	all = append(all, r)
}

// Initialize starts the HTTP server with all registered routes and middleware
func Initialize(ctx context.Context) {
	once.Do(func() {
		r := mux.NewRouter()

		// Register all routes
		for _, route := range all {
			route.Routes(r)
		}

		// Apply global middlewares
		var handler http.Handler = r
		if len(globalMiddlewares) > 0 {
			handler = applyGlobalMiddlewares(r)
		}

		// Apply framework middlewares (recovery and logging)
		handler = middleware.Recovery(
			middleware.Logger(handler.ServeHTTP).ServeHTTP,
		)

		// Setup CORS
		corsHandler := setupCORS()
		handler = corsHandler.Handler(handler)

		// Get server configuration
		port := viper.GetString("port")
		if port == "" {
			port = ":8080"
		}

		server := &http.Server{
			Addr:              port,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		}

		// Start server in goroutine
		go func() {
			logrus.Infof("HTTP server listening on %s", port)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logrus.Fatalf("Failed to start server: %v", err)
			}
		}()
	})
}

// applyGlobalMiddlewares wraps the handler with all registered global middlewares
func applyGlobalMiddlewares(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var handler http.HandlerFunc = next.ServeHTTP

		// Apply middlewares in reverse order
		for i := len(globalMiddlewares) - 1; i >= 0; i-- {
			handler = globalMiddlewares[i].Handler(handler)
		}

		handler(w, r)
	})
}

// setupCORS configures CORS settings
func setupCORS() *cors.Cors {
	// Get CORS configuration from config or use defaults
	allowedOrigins := viper.GetStringSlice("cors_allowed_origins")
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	allowedMethods := viper.GetStringSlice("cors_allowed_methods")
	if len(allowedMethods) == 0 {
		allowedMethods = []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		}
	}

	allowedHeaders := viper.GetStringSlice("cors_allowed_headers")
	if len(allowedHeaders) == 0 {
		allowedHeaders = []string{"*"}
	}

	return cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   allowedMethods,
		AllowedHeaders:   allowedHeaders,
		AllowCredentials: viper.GetBool("cors_allow_credentials"),
	})
}
