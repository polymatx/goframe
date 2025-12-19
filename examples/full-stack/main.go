package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/auth"
	"github.com/polymatx/goframe/pkg/cache"
	"github.com/polymatx/goframe/pkg/database"
	"github.com/polymatx/goframe/pkg/middleware"
	"github.com/polymatx/goframe/pkg/util"
)

type User struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Username  string    `json:"username" gorm:"uniqueIndex"`
	Password  string    `json:"-"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

var jwtManager *auth.JWTManager

func main() {
	ctx := context.Background()

	// Database
	database.Register(database.Config{
		Name:     "main",
		Driver:   database.PostgreSQL,
		Host:     "localhost",
		Port:     5432,
		User:     "goframe",
		Password: "goframe",
		Database: "goframe",
	})
	database.Initialize(ctx)

	conn, _ := database.Get("main")
	conn.DB().AutoMigrate(&User{})

	// Cache
	cache.Register(cache.Config{
		Name:  "main",
		Addrs: []string{"localhost:6379"},
		Mode:  cache.ModeStandalone,
	})
	cache.Initialize(ctx)

	// JWT
	jwtManager = auth.NewJWTManager("secret-key", 24*time.Hour)

	// App
	a := app.New(&app.Config{
		Name: "full-stack-example",
		Port: ":8080",
	})

	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())
	a.Use(middleware.RateLimit(100, 10)) // 100 req/s, burst 10
	a.Use(middleware.Metrics())

	// Public routes
	public := a.Group("/api/v1")
	public.POST("/register", register)
	public.POST("/login", login)

	// Protected routes
	protected := a.Group("/api/v1", auth.BearerAuth(jwtManager))
	protected.GET("/profile", getProfile)
	protected.GET("/users", getUsers)

	// Metrics
	a.Router().Handle("/metrics", middleware.MetricsHandler()).Methods("GET")

	fmt.Println("Full-stack example running on :8080")
	fmt.Println("Public endpoints:")
	fmt.Println("  POST /api/v1/register")
	fmt.Println("  POST /api/v1/login")
	fmt.Println("Protected endpoints (requires JWT):")
	fmt.Println("  GET  /api/v1/profile")
	fmt.Println("  GET  /api/v1/users")
	fmt.Println("Metrics:")
	fmt.Println("  GET  /metrics")

	a.StartWithGracefulShutdown()
}

func register(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	if err := ctx.Bind(&req); err != nil {
		ctx.JSONError(400, err)
		return
	}

	// Hash password
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSONError(500, err)
		return
	}

	user := User{
		Username: req.Username,
		Password: hashedPassword,
		Email:    req.Email,
	}

	conn, _ := database.Get("main")
	if err := conn.DB().Create(&user).Error; err != nil {
		ctx.JSONError(400, fmt.Errorf("username already exists"))
		return
	}

	ctx.JSON(201, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

func login(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := ctx.Bind(&req); err != nil {
		ctx.JSONError(400, err)
		return
	}

	conn, _ := database.Get("main")
	var user User
	if err := conn.DB().Where("username = ?", req.Username).First(&user).Error; err != nil {
		ctx.JSONError(401, fmt.Errorf("invalid credentials"))
		return
	}

	// Check password
	if !util.CheckPassword(req.Password, user.Password) {
		ctx.JSONError(401, fmt.Errorf("invalid credentials"))
		return
	}

	// Generate JWT
	token, _ := jwtManager.GenerateToken(
		fmt.Sprintf("%d", user.ID),
		user.Username,
		"user",
		nil,
	)

	// Cache user session
	mgr, _ := cache.Get("main")
	mgr.SetJSON(r.Context(), fmt.Sprintf("session:%d", user.ID), user, time.Hour)

	ctx.JSON(200, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

func getProfile(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	claims, _ := auth.GetClaims(r.Context())

	conn, _ := database.Get("main")
	var user User
	if err := conn.DB().First(&user, claims.UserID).Error; err != nil {
		ctx.JSONError(404, fmt.Errorf("user not found"))
		return
	}

	ctx.JSON(200, user)
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	// Check cache first
	mgr, _ := cache.Get("main")
	var users []User

	err := mgr.GetJSON(r.Context(), "users:all", &users)
	if err == nil {
		ctx.JSON(200, map[string]interface{}{
			"source": "cache",
			"users":  users,
		})
		return
	}

	// Get from database
	conn, _ := database.Get("main")
	if err := conn.DB().Find(&users).Error; err != nil {
		ctx.JSONError(500, err)
		return
	}

	// Cache for 5 minutes
	mgr.SetJSON(r.Context(), "users:all", users, 5*time.Minute)

	ctx.JSON(200, map[string]interface{}{
		"source": "database",
		"users":  users,
	})
}
