package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/database"
	"github.com/polymatx/goframe/pkg/middleware"
)

type User struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Name      string    `json:"name"`
	Email     string    `json:"email" gorm:"uniqueIndex"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func main() {
	ctx := context.Background()

	// Register database
	database.Register(database.Config{
		Name:            "main",
		Driver:          database.PostgreSQL,
		Host:            "localhost",
		Port:            5432,
		User:            "goframe",
		Password:        "goframe",
		Database:        "goframe",
		MaxOpenConns:    100,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
	})

	if err := database.Initialize(ctx); err != nil {
		panic(err)
	}

	// Auto migrate
	conn, _ := database.Get("main")
	conn.DB().AutoMigrate(&User{})

	// Create app
	a := app.New(&app.Config{
		Name: "database-example",
		Port: ":8080",
	})

	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())

	// Routes
	api := a.Group("/api/v1")
	api.POST("/users", createUser)
	api.GET("/users", getUsers)
	api.GET("/users/{id}", getUser)
	api.PUT("/users/{id}", updateUser)
	api.DELETE("/users/{id}", deleteUser)

	fmt.Println("Database example running on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  POST   /api/v1/users")
	fmt.Println("  GET    /api/v1/users")
	fmt.Println("  GET    /api/v1/users/:id")
	fmt.Println("  PUT    /api/v1/users/:id")
	fmt.Println("  DELETE /api/v1/users/:id")

	a.StartWithGracefulShutdown()
}

func createUser(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var user User
	if err := ctx.Bind(&user); err != nil {
		ctx.JSONError(400, err)
		return
	}

	conn, _ := database.Get("main")
	if err := conn.DB().Create(&user).Error; err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.JSON(201, user)
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var users []User
	conn, _ := database.Get("main")

	query := conn.DB()

	// Pagination
	if page := ctx.Query("page"); page != "" {
		if p, _ := strconv.Atoi(page); p > 0 {
			query = query.Offset((p - 1) * 10).Limit(10)
		}
	}

	if err := query.Find(&users).Error; err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.JSON(200, users)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")

	var user User
	conn, _ := database.Get("main")

	if err := conn.DB().First(&user, id).Error; err != nil {
		ctx.JSONError(404, fmt.Errorf("user not found"))
		return
	}

	ctx.JSON(200, user)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")

	conn, _ := database.Get("main")

	var user User
	if err := conn.DB().First(&user, id).Error; err != nil {
		ctx.JSONError(404, fmt.Errorf("user not found"))
		return
	}

	if err := ctx.Bind(&user); err != nil {
		ctx.JSONError(400, err)
		return
	}

	if err := conn.DB().Save(&user).Error; err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.JSON(200, user)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")

	conn, _ := database.Get("main")
	if err := conn.DB().Delete(&User{}, id).Error; err != nil {
		ctx.JSONError(500, err)
		return
	}

	ctx.NoContent()
}
