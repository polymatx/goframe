package main

import (
	"net/http"

	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/middleware"
)

func main() {
	// Create app
	a := app.New(&app.Config{
		Name: "basic-example",
		Port: ":8080",
	})

	// Add middleware
	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())
	a.Use(middleware.Compress()) // Gzip compression

	// Routes
	a.Router().HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := app.NewContext(w, r)
		ctx.JSON(200, map[string]string{
			"message": "Hello from GoFrame!",
		})
	}).Methods("GET")

	a.Router().HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		ctx := app.NewContext(w, r)
		ctx.String(200, "pong")
	}).Methods("GET")

	// Start
	a.StartWithGracefulShutdown()
}
