package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/elasticsearch"
	"github.com/polymatx/goframe/pkg/middleware"
)

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"created_at"`
}

func main() {
	ctx := context.Background()

	// Register Elasticsearch
	elasticsearch.RegisterElasticSearch(
		"main",
		"http://localhost:9200",
		"",
		"",
	)

	if err := elasticsearch.Initialize(ctx); err != nil {
		panic(err)
	}

	a := app.New(&app.Config{
		Name: "elasticsearch-example",
		Port: ":8080",
	})

	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())

	api := a.Group("/api/v1")
	api.POST("/products", indexProduct)
	api.GET("/products/search", searchProducts)
	api.GET("/products/{id}", getProduct)
	api.DELETE("/products/{id}", deleteProduct)

	fmt.Println("Elasticsearch example running on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  POST   /api/v1/products")
	fmt.Println("  GET    /api/v1/products/search?q=query")
	fmt.Println("  GET    /api/v1/products/:id")
	fmt.Println("  DELETE /api/v1/products/:id")

	a.StartWithGracefulShutdown()
}

func indexProduct(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var product Product
	if err := ctx.Bind(&product); err != nil {
		ctx.JSONError(400, err)
		return
	}

	product.ID = fmt.Sprintf("prod_%d", time.Now().Unix())
	product.CreatedAt = time.Now()

	client, _ := elasticsearch.GetElasticSearchConnection("main")
	if err := client.Index(r.Context(), "products", product.ID, product); err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.JSON(201, product)
}

func searchProducts(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	query := ctx.Query("q")

	if query == "" {
		ctx.JSONError(400, fmt.Errorf("query parameter required"))
		return
	}

	client, _ := elasticsearch.GetElasticSearchConnection("main")

	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"name", "description", "category"},
			},
		},
	}

	results, err := client.Search(r.Context(), "products", searchQuery)
	if err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.JSON(200, map[string]interface{}{
		"query":   query,
		"results": results,
	})
}

func getProduct(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")

	client, _ := elasticsearch.GetElasticSearchConnection("main")

	var product Product
	if err := client.Get(r.Context(), "products", id, &product); err != nil {
		ctx.JSONError(404, fmt.Errorf("product not found"))
		return
	}

	ctx.JSON(200, product)
}

func deleteProduct(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")

	client, _ := elasticsearch.GetElasticSearchConnection("main")
	if err := client.Delete(r.Context(), "products", id); err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.NoContent()
}
