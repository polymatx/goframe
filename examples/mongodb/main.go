package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/middleware"
	"github.com/polymatx/goframe/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Product struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	Price       float64            `json:"price" bson:"price"`
	Category    string             `json:"category" bson:"category"`
	Stock       int                `json:"stock" bson:"stock"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

func main() {
	ctx := context.Background()

	// Register MongoDB
	mongodb.Register(mongodb.Config{
		Name:     "main",
		URI:      "mongodb://localhost:27017",
		Database: "goframe",
	})

	if err := mongodb.Initialize(ctx); err != nil {
		panic(err)
	}

	a := app.New(&app.Config{
		Name: "mongodb-example",
		Port: ":8080",
	})

	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())

	api := a.Group("/api/v1")
	api.POST("/products", createProduct)
	api.GET("/products", getProducts)
	api.GET("/products/{id}", getProduct)
	api.PUT("/products/{id}", updateProduct)
	api.DELETE("/products/{id}", deleteProduct)
	api.GET("/products/search", searchProducts)

	fmt.Println("MongoDB example running on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  POST   /api/v1/products")
	fmt.Println("  GET    /api/v1/products")
	fmt.Println("  GET    /api/v1/products/:id")
	fmt.Println("  PUT    /api/v1/products/:id")
	fmt.Println("  DELETE /api/v1/products/:id")
	fmt.Println("  GET    /api/v1/products/search?category=electronics")

	a.StartWithGracefulShutdown()
}

func createProduct(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var product Product
	if err := ctx.Bind(&product); err != nil {
		ctx.JSONError(400, err)
		return
	}

	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	client, _ := mongodb.Get("main")
	result, err := client.InsertOne(r.Context(), "products", product)
	if err != nil {
		ctx.JSONError(500, err)
		return
	}

	product.ID = result.InsertedID.(primitive.ObjectID)
	ctx.JSON(201, product)
}

func getProducts(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var products []Product
	client, _ := mongodb.Get("main")

	filter := bson.M{}
	if err := client.Find(r.Context(), "products", filter, &products); err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.JSON(200, products)
}

func getProduct(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		ctx.JSONError(400, fmt.Errorf("invalid id"))
		return
	}

	var product Product
	client, _ := mongodb.Get("main")

	if err := client.FindByID(r.Context(), "products", objectID, &product); err != nil {
		ctx.JSONError(404, fmt.Errorf("product not found"))
		return
	}

	ctx.JSON(200, product)
}

func updateProduct(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		ctx.JSONError(400, fmt.Errorf("invalid id"))
		return
	}

	var product Product
	if err := ctx.Bind(&product); err != nil {
		ctx.JSONError(400, err)
		return
	}

	product.UpdatedAt = time.Now()
	update := bson.M{
		"$set": bson.M{
			"name":        product.Name,
			"description": product.Description,
			"price":       product.Price,
			"category":    product.Category,
			"stock":       product.Stock,
			"updated_at":  product.UpdatedAt,
		},
	}

	client, _ := mongodb.Get("main")
	result, err := client.UpdateByID(r.Context(), "products", objectID, update)
	if err != nil {
		ctx.JSONError(500, err)
		return
	}

	if result.MatchedCount == 0 {
		ctx.JSONError(404, fmt.Errorf("product not found"))
		return
	}

	product.ID = objectID
	ctx.JSON(200, product)
}

func deleteProduct(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		ctx.JSONError(400, fmt.Errorf("invalid id"))
		return
	}

	client, _ := mongodb.Get("main")
	result, err := client.DeleteByID(r.Context(), "products", objectID)
	if err != nil {
		ctx.JSONError(500, err)
		return
	}

	if result.DeletedCount == 0 {
		ctx.JSONError(404, fmt.Errorf("product not found"))
		return
	}

	ctx.NoContent()
}

func searchProducts(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	category := ctx.Query("category")

	var products []Product
	client, _ := mongodb.Get("main")

	filter := bson.M{}
	if category != "" {
		filter["category"] = category
	}

	if err := client.Find(r.Context(), "products", filter, &products); err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.JSON(200, products)
}
