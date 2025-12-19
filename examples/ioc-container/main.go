package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/cache"
	"github.com/polymatx/goframe/pkg/container"
	"github.com/polymatx/goframe/pkg/database"
	"github.com/polymatx/goframe/pkg/middleware"
)

// Services
type UserRepository struct {
	db *database.Connection
}

func NewUserRepository(db *database.Connection) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetAll() ([]User, error) {
	var users []User
	err := r.db.DB().Find(&users).Error
	return users, err
}

type CacheService struct {
	cache *cache.Manager
}

func NewCacheService(cache *cache.Manager) *CacheService {
	return &CacheService{cache: cache}
}

func (s *CacheService) Get(ctx context.Context, key string) (string, error) {
	return s.cache.Get(ctx, key)
}

func (s *CacheService) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.cache.Set(ctx, key, value, ttl)
}

type UserService struct {
	repo  *UserRepository
	cache *CacheService
}

func NewUserService(repo *UserRepository, cache *CacheService) *UserService {
	return &UserService{
		repo:  repo,
		cache: cache,
	}
}

func (s *UserService) GetUsers(ctx context.Context) ([]User, error) {
	return s.repo.GetAll()
}

// Models
type User struct {
	ID    uint   `json:"id" gorm:"primarykey"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Handlers with dependency injection
type UserHandler struct {
	userService *UserService `inject:"userService"`
}

func main() {
	ctx := context.Background()

	// Setup database
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

	// Setup cache
	cache.Register(cache.Config{
		Name:  "main",
		Addrs: []string{"localhost:6379"},
		Mode:  cache.ModeStandalone,
	})
	cache.Initialize(ctx)

	// Auto migrate
	conn, _ := database.Get("main")
	conn.DB().AutoMigrate(&User{})

	// Create app
	a := app.New(&app.Config{
		Name: "ioc-example",
		Port: ":8080",
	})

	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())

	// Register services in IoC container
	appContainer := a.Container()

	// Register database
	appContainer.Singleton("database", func(c *container.Container) (interface{}, error) {
		return database.Get("main")
	})

	// Register cache
	appContainer.Singleton("cache", func(c *container.Container) (interface{}, error) {
		return cache.Get("main")
	})

	// Register repository
	appContainer.Singleton("userRepository", func(c *container.Container) (interface{}, error) {
		db, _ := c.Resolve("database")
		return NewUserRepository(db.(*database.Connection)), nil
	})

	// Register cache service
	appContainer.Singleton("cacheService", func(c *container.Container) (interface{}, error) {
		cacheManager, _ := c.Resolve("cache")
		return NewCacheService(cacheManager.(*cache.Manager)), nil
	})

	// Register user service
	appContainer.Singleton("userService", func(c *container.Container) (interface{}, error) {
		repo, _ := c.Resolve("userRepository")
		cacheService, _ := c.Resolve("cacheService")
		return NewUserService(
			repo.(*UserRepository),
			cacheService.(*CacheService),
		), nil
	})

	// Routes with manual injection
	api := a.Group("/api/v1")

	api.GET("/users", func(w http.ResponseWriter, r *http.Request) {
		ctx := app.NewContext(w, r)

		// Resolve service from container
		service, _ := appContainer.Resolve("userService")
		userService := service.(*UserService)

		users, err := userService.GetUsers(r.Context())
		if err != nil {
			ctx.JSONError(500, err)
			return
		}

		ctx.JSON(200, users)
	})

	api.POST("/users", func(w http.ResponseWriter, r *http.Request) {
		ctx := app.NewContext(w, r)

		var user User
		if err := ctx.Bind(&user); err != nil {
			ctx.JSONError(400, err)
			return
		}

		// Resolve database
		db, _ := appContainer.Resolve("database")
		conn := db.(*database.Connection)

		if err := conn.DB().Create(&user).Error; err != nil {
			ctx.JSONError(500, err)
			return
		}

		ctx.JSON(201, user)
	})

	// Auto-inject handler
	handler := &UserHandler{}
	if err := appContainer.Inject(handler); err != nil {
		panic(err)
	}

	api.GET("/users/with-injection", func(w http.ResponseWriter, r *http.Request) {
		ctx := app.NewContext(w, r)
		users, _ := handler.userService.GetUsers(r.Context())
		ctx.JSON(200, users)
	})

	fmt.Println("IoC Container example running on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  GET  /api/v1/users")
	fmt.Println("  POST /api/v1/users")
	fmt.Println("  GET  /api/v1/users/with-injection")

	a.StartWithGracefulShutdown()
}
