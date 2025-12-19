package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/polymatx/goframe/pkg/container"
	"github.com/sirupsen/logrus"
)

// App represents the application
type App struct {
	router     *mux.Router
	server     *http.Server
	middleware []MiddlewareFunc
	config     *Config
	container  *container.Container
}

// Config holds application configuration
type Config struct {
	Name            string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// MiddlewareFunc is a middleware function type
type MiddlewareFunc func(http.Handler) http.Handler

// New creates a new application instance
func New(cfg *Config) *App {
	if cfg == nil {
		cfg = &Config{
			Name:            "goframe-app",
			Port:            ":8080",
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		}
	}

	app := &App{
		router:     mux.NewRouter(),
		middleware: make([]MiddlewareFunc, 0),
		config:     cfg,
		container:  container.New(),
	}

	// Bind app to container
	_ = app.container.Bind("app", app)

	return app
}

// Router returns the underlying mux router
func (a *App) Router() *mux.Router {
	return a.router
}

// Container returns the IoC container
func (a *App) Container() *container.Container {
	return a.container
}

// Use adds middleware to the application
func (a *App) Use(middleware ...MiddlewareFunc) {
	a.middleware = append(a.middleware, middleware...)
}

// Group creates a route group with optional middleware
func (a *App) Group(prefix string, middleware ...MiddlewareFunc) *RouteGroup {
	return &RouteGroup{
		router:     a.router.PathPrefix(prefix).Subrouter(),
		middleware: middleware,
		container:  a.container,
	}
}

// Start starts the HTTP server
func (a *App) Start(ctx context.Context) error {
	handler := a.buildHandler()

	a.server = &http.Server{
		Addr:         a.config.Port,
		Handler:      handler,
		ReadTimeout:  a.config.ReadTimeout,
		WriteTimeout: a.config.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logrus.Infof("Starting %s on %s", a.config.Name, a.config.Port)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return a.Shutdown(context.Background())
	}
}

// StartWithGracefulShutdown starts the server and handles graceful shutdown
func (a *App) StartWithGracefulShutdown() error {
	handler := a.buildHandler()

	a.server = &http.Server{
		Addr:         a.config.Port,
		Handler:      handler,
		ReadTimeout:  a.config.ReadTimeout,
		WriteTimeout: a.config.WriteTimeout,
	}

	go func() {
		logrus.Infof("Starting %s on %s", a.config.Name, a.config.Port)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	logrus.Info("Server exited")
	return nil
}

// Shutdown gracefully shuts down the server
func (a *App) Shutdown(ctx context.Context) error {
	if a.server == nil {
		return nil
	}

	logrus.Info("Shutting down server...")
	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	logrus.Info("Server stopped")
	return nil
}

// buildHandler builds the final handler with all middleware
func (a *App) buildHandler() http.Handler {
	handler := http.Handler(a.router)

	for i := len(a.middleware) - 1; i >= 0; i-- {
		handler = a.middleware[i](handler)
	}

	return handler
}

// RouteGroup represents a group of routes with shared middleware
type RouteGroup struct {
	router     *mux.Router
	middleware []MiddlewareFunc
	container  *container.Container
}

// Use adds middleware to the group
func (g *RouteGroup) Use(middleware ...MiddlewareFunc) {
	g.middleware = append(g.middleware, middleware...)
}

// Group creates a sub-group
func (g *RouteGroup) Group(prefix string, middleware ...MiddlewareFunc) *RouteGroup {
	allMiddleware := append(g.middleware, middleware...)
	return &RouteGroup{
		router:     g.router.PathPrefix(prefix).Subrouter(),
		middleware: allMiddleware,
		container:  g.container,
	}
}

// GET registers a GET route
func (g *RouteGroup) GET(path string, handler http.HandlerFunc) {
	g.handle("GET", path, handler)
}

// POST registers a POST route
func (g *RouteGroup) POST(path string, handler http.HandlerFunc) {
	g.handle("POST", path, handler)
}

// PUT registers a PUT route
func (g *RouteGroup) PUT(path string, handler http.HandlerFunc) {
	g.handle("PUT", path, handler)
}

// DELETE registers a DELETE route
func (g *RouteGroup) DELETE(path string, handler http.HandlerFunc) {
	g.handle("DELETE", path, handler)
}

// PATCH registers a PATCH route
func (g *RouteGroup) PATCH(path string, handler http.HandlerFunc) {
	g.handle("PATCH", path, handler)
}

// Handle registers a route with specific method
func (g *RouteGroup) Handle(method, path string, handler http.HandlerFunc) {
	g.handle(method, path, handler)
}

func (g *RouteGroup) handle(method, path string, handler http.HandlerFunc) {
	var h http.Handler = handler
	for i := len(g.middleware) - 1; i >= 0; i-- {
		h = g.middleware[i](h)
	}

	g.router.Handle(path, h).Methods(method)
}
