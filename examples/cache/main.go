package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/cache"
	"github.com/polymatx/goframe/pkg/middleware"
)

type CacheItem struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
	TTL   int         `json:"ttl"`
}

func main() {
	ctx := context.Background()

	// Register Redis cache
	cache.Register(cache.Config{
		Name:  "main",
		Addrs: []string{"localhost:6379"},
		Mode:  cache.ModeStandalone,
	})

	if err := cache.Initialize(ctx); err != nil {
		panic(err)
	}

	a := app.New(&app.Config{
		Name: "cache-example",
		Port: ":8080",
	})

	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())

	api := a.Group("/api/v1")
	api.POST("/cache", setCache)
	api.GET("/cache/{key}", getCache)
	api.DELETE("/cache/{key}", deleteCache)
	api.GET("/cache/json/{key}", getJSONCache)
	api.POST("/cache/json", setJSONCache)

	fmt.Println("Cache example running on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  POST   /api/v1/cache")
	fmt.Println("  GET    /api/v1/cache/:key")
	fmt.Println("  DELETE /api/v1/cache/:key")
	fmt.Println("  POST   /api/v1/cache/json")
	fmt.Println("  GET    /api/v1/cache/json/:key")

	a.StartWithGracefulShutdown()
}

func setCache(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var item CacheItem
	if err := ctx.Bind(&item); err != nil {
		ctx.JSONError(400, err)
		return
	}

	mgr, _ := cache.Get("main")
	ttl := time.Duration(item.TTL) * time.Second
	if err := mgr.Set(r.Context(), item.Key, fmt.Sprint(item.Value), ttl); err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.JSON(200, map[string]string{"message": "cached"})
}

func getCache(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	key := ctx.Param("key")

	mgr, _ := cache.Get("main")
	value, err := mgr.Get(r.Context(), key)
	if err != nil {
		ctx.JSONError(404, fmt.Errorf("key not found"))
		return
	}

	ctx.JSON(200, map[string]string{"key": key, "value": value})
}

func deleteCache(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	key := ctx.Param("key")

	mgr, _ := cache.Get("main")
	if err := mgr.Del(r.Context(), key); err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.NoContent()
}

func setJSONCache(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var item struct {
		Key   string                 `json:"key"`
		Value map[string]interface{} `json:"value"`
		TTL   int                    `json:"ttl"`
	}

	if err := ctx.Bind(&item); err != nil {
		ctx.JSONError(400, err)
		return
	}

	mgr, _ := cache.Get("main")
	ttl := time.Duration(item.TTL) * time.Second
	if err := mgr.SetJSON(r.Context(), item.Key, item.Value, ttl); err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.JSON(200, map[string]string{"message": "cached"})
}

func getJSONCache(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	key := ctx.Param("key")

	mgr, _ := cache.Get("main")
	var value map[string]interface{}
	if err := mgr.GetJSON(r.Context(), key, &value); err != nil {
		ctx.JSONError(404, fmt.Errorf("key not found"))
		return
	}

	ctx.JSON(200, map[string]interface{}{"key": key, "value": value})
}
