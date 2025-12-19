# GoFrame Documentation

Complete documentation for the GoFrame web framework.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Application Core](#application-core)
3. [Routing](#routing)
4. [Middleware](#middleware)
5. [Request & Response](#request--response)
6. [Authentication](#authentication)
7. [Database](#database)
8. [MongoDB](#mongodb)
9. [Caching](#caching)
10. [Messaging](#messaging)
11. [WebSocket](#websocket)
12. [IoC Container](#ioc-container)
13. [Utilities](#utilities)
14. [CLI Tool](#cli-tool)
15. [Deployment](#deployment)

---

## Getting Started

### Installation

```bash
go get github.com/polymatx/goframe
```

### Basic Application

```go
package main

import (
    "net/http"
    "github.com/polymatx/goframe/pkg/app"
    "github.com/polymatx/goframe/pkg/middleware"
)

func main() {
    a := app.New(&app.Config{
        Name: "my-app",
        Port: ":8080",
    })

    a.Use(middleware.Recovery())
    a.Use(middleware.Logger())

    a.Router().HandleFunc("/", homeHandler).Methods("GET")

    a.StartWithGracefulShutdown()
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    ctx := app.NewContext(w, r)
    ctx.JSON(200, map[string]string{"message": "Hello, World!"})
}
```

---

## Application Core

### Creating an App

```go
a := app.New(&app.Config{
    Name:            "my-app",
    Port:            ":8080",
    ReadTimeout:     15 * time.Second,
    WriteTimeout:    15 * time.Second,
    ShutdownTimeout: 10 * time.Second,
})
```

### Starting the Server

```go
// With graceful shutdown
a.StartWithGracefulShutdown()

// Manual control
ctx, cancel := context.WithCancel(context.Background())
go a.Start(ctx)
// ... later
cancel()
```

### Accessing the Router

```go
router := a.Router() // Returns *mux.Router
```

### IoC Container

```go
container := a.Container() // Returns *container.Container
```

---

## Routing

### Basic Routes

```go
router := a.Router()

router.HandleFunc("/users", getUsers).Methods("GET")
router.HandleFunc("/users", createUser).Methods("POST")
router.HandleFunc("/users/{id}", getUser).Methods("GET")
router.HandleFunc("/users/{id}", updateUser).Methods("PUT")
router.HandleFunc("/users/{id}", deleteUser).Methods("DELETE")
```

### Route Groups

```go
// Public routes
public := a.Group("/api/v1")
public.POST("/register", registerHandler)
public.POST("/login", loginHandler)

// Protected routes with middleware
protected := a.Group("/api/v1", auth.BearerAuth(jwtManager))
protected.GET("/profile", profileHandler)
protected.GET("/users", usersHandler)

// Nested groups
admin := protected.Group("/admin", adminMiddleware)
admin.GET("/stats", statsHandler)
```

### Route Parameters

```go
func getUser(w http.ResponseWriter, r *http.Request) {
    ctx := app.NewContext(w, r)
    id := ctx.Param("id") // Get route parameter
    
    ctx.JSON(200, map[string]string{"id": id})
}
```

### Query Parameters

```go
func searchUsers(w http.ResponseWriter, r *http.Request) {
    ctx := app.NewContext(w, r)
    
    query := ctx.Query("q")
    page := ctx.Query("page")
    limit := ctx.Query("limit")
    
    // ... search logic
}
```

---

## Middleware

### Built-in Middleware

#### Recovery

```go
a.Use(middleware.Recovery())
```

#### Logger

```go
a.Use(middleware.Logger())
```

#### CORS

```go
// Default CORS (allows all origins)
a.Use(middleware.DefaultCORS())

// Custom CORS
a.Use(middleware.CORS(&middleware.CORSConfig{
    AllowedOrigins:   []string{"https://example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
    MaxAge:           86400,
}))
```

#### Compression

```go
a.Use(middleware.Compress())
```

#### Rate Limiting

```go
// 100 requests per second per IP, with burst of 10
a.Use(middleware.RateLimit(100, 10))
```

#### Metrics

```go
a.Use(middleware.Metrics())

// Expose metrics endpoint
a.Router().Handle("/metrics", middleware.MetricsHandler()).Methods("GET")
```

### Custom Middleware

```go
func customMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Before request
            log.Println("Before")
            
            next.ServeHTTP(w, r)
            
            // After request
            log.Println("After")
        })
    }
}

a.Use(customMiddleware())
```

---

## Request & Response

### Context Helper

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := app.NewContext(w, r)
    
    // Get parameters
    id := ctx.Param("id")
    query := ctx.Query("search")
    header := ctx.Header("Authorization")
    
    // Get client IP
    ip := ctx.ClientIP()
    
    // Bind request body
    var data MyStruct
    ctx.Bind(&data)
    
    // Send responses
    ctx.JSON(200, data)
    ctx.String(200, "Hello")
    ctx.NoContent()
    ctx.JSONError(400, errors.New("bad request"))
}
```

### Request Binding

```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=18"`
}

func createUser(w http.ResponseWriter, r *http.Request) {
    ctx := app.NewContext(w, r)
    
    var req CreateUserRequest
    if err := ctx.Bind(&req); err != nil {
        ctx.JSONError(400, err)
        return
    }
    
    // req is now populated and validated
    ctx.JSON(201, req)
}
```

### Response Rendering

```go
// JSON
ctx.JSON(200, map[string]string{"status": "ok"})

// String
ctx.String(200, "Hello, World!")

// No Content
ctx.NoContent() // 204 status

// Error
ctx.JSONError(400, errors.New("invalid input"))

// Redirect
ctx.Redirect(302, "/new-location")
```

---

## Authentication

### JWT Authentication

```go
import "github.com/polymatx/goframe/pkg/auth"

// Create JWT manager
jwtManager := auth.NewJWTManager("your-secret-key", 24*time.Hour)

// Generate token
token, err := jwtManager.GenerateToken(
    "user-123",           // User ID
    "john@example.com",   // Username
    "user",               // Role
    map[string]interface{}{"premium": true}, // Custom claims
)

// Protect routes
protected := a.Group("/api", auth.BearerAuth(jwtManager))
protected.GET("/profile", profileHandler)

// Get claims in handler
func profileHandler(w http.ResponseWriter, r *http.Request) {
    claims, _ := auth.GetClaims(r.Context())
    
    userID := claims.UserID
    username := claims.Username
    role := claims.Role
    extra := claims.Extra // Custom claims map
}
```

### Basic Authentication

```go
validator := func(username, password string) bool {
    // Check credentials
    return username == "admin" && password == "secret"
}

a.Use(auth.BasicAuth(validator))
```

### API Key Authentication

```go
validator := func(apiKey string) bool {
    // Validate API key
    return apiKey == "valid-api-key"
}

a.Use(auth.APIKeyAuth("X-API-Key", validator))
```

---

## Database

### Configuration

```go
import "github.com/polymatx/goframe/pkg/database"

// Register database
database.Register(database.Config{
    Name:            "main",
    Driver:          database.PostgreSQL, // or MySQL, SQLite
    Host:            "localhost",
    Port:            5432,
    User:            "user",
    Password:        "pass",
    Database:        "mydb",
    MaxOpenConns:    100,
    MaxIdleConns:    10,
    ConnMaxLifetime: time.Hour,
})

// Initialize
ctx := context.Background()
database.Initialize(ctx)

// Get connection
conn, _ := database.Get("main")
db := conn.DB() // Returns *gorm.DB
```

### Models

```go
type User struct {
    ID        uint      `gorm:"primarykey"`
    Name      string    `gorm:"not null"`
    Email     string    `gorm:"uniqueIndex"`
    Age       int
    CreatedAt time.Time
    UpdatedAt time.Time
}

// Auto migrate
conn, _ := database.Get("main")
conn.DB().AutoMigrate(&User{})
```

### CRUD Operations

```go
// Create
user := User{Name: "John", Email: "john@example.com", Age: 30}
db.Create(&user)

// Read
var user User
db.First(&user, 1) // Find by ID
db.Where("email = ?", "john@example.com").First(&user)

// Update
db.Model(&user).Update("age", 31)
db.Model(&user).Updates(User{Name: "John Doe", Age: 31})

// Delete
db.Delete(&user, 1)

// Query
var users []User
db.Find(&users)
db.Where("age > ?", 18).Find(&users)
db.Order("created_at desc").Limit(10).Find(&users)
```

---

## MongoDB

### Configuration

```go
import "github.com/polymatx/goframe/pkg/mongodb"

mongodb.Register(mongodb.Config{
    Name:     "main",
    URI:      "mongodb://localhost:27017",
    Database: "mydb",
})

mongodb.Initialize(ctx)

client, _ := mongodb.Get("main")
```

### Operations

```go
// Insert
result, _ := client.InsertOne(ctx, "users", user)

// Find One
var user User
client.FindOne(ctx, "users", bson.M{"email": "john@example.com"}, &user)

// Find Many
var users []User
client.Find(ctx, "users", bson.M{}, &users)

// Update
update := bson.M{"$set": bson.M{"age": 31}}
client.UpdateOne(ctx, "users", filter, update)

// Delete
client.DeleteOne(ctx, "users", bson.M{"_id": id})

// Aggregate
pipeline := []bson.M{
    {"$match": bson.M{"age": bson.M{"$gte": 18}}},
    {"$group": bson.M{"_id": "$city", "count": bson.M{"$sum": 1}}},
}
var results []bson.M
client.Aggregate(ctx, "users", pipeline, &results)
```

### Transactions

```go
err := client.Transaction(ctx, func(sessCtx mongo.SessionContext) error {
    // All operations in this function are part of the transaction
    client.InsertOne(sessCtx, "users", user1)
    client.InsertOne(sessCtx, "users", user2)
    return nil
})
```

---

## Caching

### Redis Configuration

```go
import "github.com/polymatx/goframe/pkg/cache"

// Standalone
cache.Register(cache.Config{
    Name:  "main",
    Addrs: []string{"localhost:6379"},
    Mode:  cache.ModeStandalone,
})

// Cluster
cache.Register(cache.Config{
    Name:  "main",
    Addrs: []string{"node1:6379", "node2:6379", "node3:6379"},
    Mode:  cache.ModeCluster,
})

cache.Initialize(ctx)
mgr, _ := cache.Get("main")
```

### Operations

```go
// Set/Get
mgr.Set(ctx, "key", "value", 5*time.Minute)
value, _ := mgr.Get(ctx, "key")

// JSON
mgr.SetJSON(ctx, "user:1", user, time.Hour)
var user User
mgr.GetJSON(ctx, "user:1", &user)

// Delete
mgr.Del(ctx, "key")

// Increment/Decrement
mgr.Incr(ctx, "counter")
mgr.IncrBy(ctx, "counter", 5)

// Hashes
mgr.HSet(ctx, "user:1", "name", "John", "age", 30)
value, _ := mgr.HGet(ctx, "user:1", "name")
all, _ := mgr.HGetAll(ctx, "user:1")

// Lists
mgr.LPush(ctx, "queue", "item1", "item2")
item, _ := mgr.RPop(ctx, "queue")

// Sets
mgr.SAdd(ctx, "tags", "go", "web", "api")
members, _ := mgr.SMembers(ctx, "tags")

// Sorted Sets
mgr.ZAdd(ctx, "leaderboard", &cache.Z{Score: 100, Member: "player1"})
players, _ := mgr.ZRange(ctx, "leaderboard", 0, 9)
```

---

## Messaging

### RabbitMQ

```go
import "github.com/polymatx/goframe/pkg/rabbit"

// Register
rabbit.RegisterRabbitMq("main", "localhost", 5672, "user", "pass", "/")
rabbit.Initialize(ctx)

conn, _ := rabbit.GetConnection("main")

// Publish
conn.Publish(ctx, "queue_name", []byte("message"))
conn.PublishJSON(ctx, "queue_name", data)

// Consume
handler := func(body []byte) error {
    // Process message
    return nil
}
conn.Consume(ctx, "queue_name", handler)
```

### MQTT

```go
import "github.com/polymatx/goframe/pkg/mqtt"

// Register
mqtt.RegisterMqtt("main", "tcp://localhost:1883", "client-id", "user", "pass")
mqtt.Initialize(ctx)

client, _ := mqtt.GetMqttConnection("main")

// Publish
client.Publish(ctx, "topic/sensors", []byte("data"))

// Subscribe
handler := func(topic string, payload []byte) error {
    // Handle message
    return nil
}
client.Subscribe(ctx, "topic/#", handler)
```

---

## WebSocket

### Server Setup

```go
import "github.com/polymatx/goframe/pkg/websocket"

hub := websocket.NewHub()
go hub.Run()

a.Router().HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("user")
    hub.Upgrade(w, r, userID)
})

// Broadcast to all connections
hub.Broadcast([]byte("Hello everyone!"))

// Get connection count
count := hub.ConnectionCount()
```

---

## IoC Container

### Registration

```go
container := a.Container()

// Bind instance
container.Bind("config", myConfig)

// Bind factory (new instance each time)
container.BindFactory("service", func(c *container.Container) (interface{}, error) {
    return NewService(), nil
})

// Singleton (one instance)
container.Singleton("database", func(c *container.Container) (interface{}, error) {
    return database.Get("main")
})
```

### Resolution

```go
// Resolve
service, err := container.Resolve("service")

// Must resolve (panics on error)
service := container.MustResolve("service")

// Check existence
if container.Has("service") {
    // ...
}
```

### Auto-Injection

```go
type UserHandler struct {
    DB          *database.Connection `inject:"database"`
    CacheServ   *CacheService        `inject:"cacheService"`
}

handler := &UserHandler{}
container.Inject(handler)
// Fields are now populated
```

---

## Utilities

### Crypto

```go
import "github.com/polymatx/goframe/pkg/util"

// Hash password
hash, _ := util.HashPassword("password123")
valid := util.CheckPassword("password123", hash)

// Generate token
token, _ := util.RandomToken()

// UUID
id, _ := util.UUIDv4()

// Hashing
md5 := util.MD5("text")
sha1 := util.SHA1("text")
sha256 := util.SHA256("text")

// Base64
encoded := util.Base64Encode([]byte("data"))
decoded := util.Base64Decode(encoded)
```

### String Utilities

```go
// Case conversion
snake := util.CamelToSnake("MyVariable") // "my_variable"
camel := util.SnakeToCamel("my_variable") // "MyVariable"

// Text manipulation
truncated := util.Truncate("long text...", 10) // "long text..."
capitalized := util.Capitalize("hello")         // "Hello"
clean := util.RemoveSpaces("  text  ")          // "text"
```

### Slice Utilities

```go
// Contains
exists := util.Contains([]int{1, 2, 3}, 2) // true

// Filter
evens := util.Filter([]int{1, 2, 3, 4}, func(n int) bool {
    return n%2 == 0
}) // [2, 4]

// Map
doubled := util.Map([]int{1, 2, 3}, func(n int) int {
    return n * 2
}) // [2, 4, 6]

// Unique
unique := util.Unique([]int{1, 2, 2, 3}) // [1, 2, 3]

// Chunk
chunks := util.Chunk([]int{1, 2, 3, 4, 5}, 2) // [[1,2], [3,4], [5]]
```

### Validation

```go
import "github.com/polymatx/goframe/pkg/validator"

validator.IsEmail("test@example.com")
validator.IsURL("https://example.com")
validator.IsPhone("+1234567890")
validator.IsStrongPassword("MyP@ss123")
validator.IsIPv4("192.168.1.1")
validator.IsJSON(`{"key": "value"}`)
```

---

## CLI Tool

### Installation

```bash
go install ./cmd/goframe
```

### Commands

```bash
# Create new project
goframe new myapp

# Generate model
goframe gen model User

# Generate handler
goframe gen handler user

# Generate CRUD (model + handler)
goframe gen crud Product

# Generate middleware
goframe gen middleware Auth

# Development server with hot reload
goframe serve

# Build production binary
goframe build

# Database migrations
goframe migrate
```

---

## Deployment

### Environment Variables

```bash
APP_ENV=production
LOG_LEVEL=info
PORT=8080
DB_HOST=localhost
DB_PORT=5432
REDIS_ADDR=localhost:6379
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o app ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/app .
EXPOSE 8080
CMD ["./app"]
```

### Docker Compose

```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
      - mongodb
    environment:
      DB_HOST: postgres
      REDIS_ADDR: redis:6379
      MONGO_URI: mongodb://mongodb:27017
```

### Systemd Service

```ini
[Unit]
Description=GoFrame App
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/myapp
ExecStart=/opt/myapp/bin/app
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

---

## Best Practices

1. **Error Handling** - Always check errors and return appropriate HTTP status codes
2. **Validation** - Validate all user inputs using struct tags
3. **Logging** - Use structured logging for better debugging
4. **Context** - Pass context through all database/cache operations
5. **Graceful Shutdown** - Always use `StartWithGracefulShutdown()`
6. **Middleware Order** - Recovery → Logger → CORS → Custom
7. **Database** - Use connection pooling and prepared statements
8. **Caching** - Cache frequently accessed data with appropriate TTL
9. **Security** - Use HTTPS, validate JWT tokens, sanitize inputs
10. **Testing** - Write unit tests for handlers and services

---

For more examples, see the `examples/` directory in the repository.
