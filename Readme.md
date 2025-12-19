# GoFrame

A production-ready Go web framework with batteries included. Built for modern web applications with comprehensive tooling, database support, messaging, caching, and more.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Features

### Core Framework
- **HTTP Server** - Fast HTTP server with middleware support (Gorilla Mux)
- **Authentication** - JWT, Basic Auth, API Key authentication
- **Request/Response** - JSON/XML/Form binding and rendering
- **Validation** - Built-in input validation
- **Router Groups** - Organized routing with middleware chains
- **WebSocket** - Real-time communication with hub pattern
- **IoC Container** - Dependency injection container

### Middleware
- **Recovery** - Panic recovery with stack traces
- **Logger** - HTTP request logging
- **CORS** - Cross-origin resource sharing
- **Compression** - Gzip compression
- **Rate Limiting** - Per-IP rate limiting
- **Metrics** - Prometheus metrics

### Database Support
- **SQL Databases** - MySQL, PostgreSQL, SQLite (GORM)
- **MongoDB** - NoSQL document database
- **Redis** - Caching and pub/sub (standalone/cluster)
- **Elasticsearch** - Full-text search

### Messaging & Events
- **RabbitMQ** - Message queue and pub/sub
- **MQTT** - IoT messaging protocol


## Installation

```bash
go get github.com/polymatx/goframe
```

## Quick Start

### Basic HTTP Server

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
    a.Use(middleware.DefaultCORS())

    a.Router().HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        ctx := app.NewContext(w, r)
        ctx.JSON(200, map[string]string{"message": "Hello, World!"})
    }).Methods("GET")

    a.StartWithGracefulShutdown()
}
```

### With Database

```go
package main

import (
    "context"
    "net/http"

    "github.com/polymatx/goframe/pkg/app"
    "github.com/polymatx/goframe/pkg/database"
    "github.com/polymatx/goframe/pkg/middleware"
)

type User struct {
    ID    uint   `json:"id" gorm:"primarykey"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

var db *database.Connection

func main() {
    ctx := context.Background()

    // Setup database
    database.Register(database.Config{
        Name:     "main",
        Driver:   database.PostgreSQL,
        Host:     "localhost",
        Port:     5432,
        User:     "user",
        Password: "pass",
        Database: "mydb",
    })
    database.Initialize(ctx)

    db, _ = database.Get("main")
    db.AutoMigrate(&User{})

    // Create app
    a := app.New(&app.Config{Name: "app", Port: ":8080"})
    a.Use(middleware.Recovery())
    a.Use(middleware.Logger())

    // Routes
    api := a.Group("/api/v1")
    api.POST("/users", createUser)
    api.GET("/users", getUsers)

    a.StartWithGracefulShutdown()
}

func createUser(w http.ResponseWriter, r *http.Request) {
    ctx := app.NewContext(w, r)
    var user User
    if err := ctx.Bind(&user); err != nil {
        ctx.JSONError(400, err)
        return
    }
    db.DB().Create(&user)
    ctx.JSON(201, user)
}

func getUsers(w http.ResponseWriter, r *http.Request) {
    ctx := app.NewContext(w, r)
    var users []User
    db.DB().Find(&users)
    ctx.JSON(200, users)
}
```

## Documentation

Full documentation available at [docs/DOCUMENTATION.md](docs/DOCUMENTATION.md)

## Project Structure

```
goframe/
├── cmd/
│   └── goframe/          # CLI tool
├── pkg/
│   ├── app/              # Application core
│   ├── auth/             # Authentication (JWT, Basic, API Key)
│   ├── binding/          # Request binding
│   ├── cache/            # Redis caching
│   ├── container/        # IoC dependency injection
│   ├── database/         # SQL databases (GORM)
│   ├── elasticsearch/    # Elasticsearch client
│   ├── middleware/       # HTTP middleware
│   ├── mongodb/          # MongoDB client
│   ├── mqtt/             # MQTT client
│   ├── rabbit/           # RabbitMQ client
│   ├── render/           # Response rendering
│   ├── util/             # Utilities (crypto, strings, slices)
│   ├── validator/        # Input validation
│   └── websocket/        # WebSocket support
├── examples/             # Example applications
├── build/                # Docker and deployment
└── docs/                 # Documentation
```

## CLI Tool

Install the CLI tool:

```bash
go install ./cmd/goframe
```

### Available Commands

```bash
# Create new project
goframe new myapp

# Generate code
goframe gen model User
goframe gen handler user
goframe gen crud Product
goframe gen middleware Auth

# Development
goframe serve              # Start with hot reload
goframe build              # Build production binary

# Database
goframe migrate            # Run migrations
```

## Docker

### Development

```bash
# Start all services (Postgres, Redis, MongoDB, RabbitMQ)
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

### Production

```bash
docker build -f build/Dockerfile -t myapp:latest .
docker run -p 8080:8080 myapp:latest
```

## Examples

All examples are in the `examples/` directory:

- **basic** - Simple HTTP server
- **rest-api** - REST API with CRUD operations
- **database** - PostgreSQL database operations
- **mongodb** - MongoDB operations
- **cache** - Redis caching
- **websocket-chat** - Real-time WebSocket chat
- **rabbitmq** - Message queue example
- **mqtt** - MQTT pub/sub example
- **elasticsearch** - Search operations
- **full-stack** - Complete app (DB + Cache + Auth)
- **ioc-container** - Dependency injection example

Run examples:

```bash
make run-basic
make run-database
make run-mongodb
make run-websocket
make run-full-stack
```

## Configuration

### Database

```go
// PostgreSQL
database.Register(database.Config{
    Name:     "main",
    Driver:   database.PostgreSQL,
    Host:     "localhost",
    Port:     5432,
    User:     "user",
    Password: "pass",
    Database: "mydb",
})

// MySQL
database.Register(database.Config{
    Driver:   database.MySQL,
    Host:     "localhost",
    Port:     3306,
    // ...
})
```

### MongoDB

```go
mongodb.Register(mongodb.Config{
    Name:     "main",
    URI:      "mongodb://localhost:27017",
    Database: "mydb",
})
```

### Redis Cache

```go
cache.Register(cache.Config{
    Name:  "main",
    Addrs: []string{"localhost:6379"},
    Mode:  cache.ModeStandalone, // or cache.ModeCluster
})
```

### RabbitMQ

```go
rabbit.RegisterRabbitMq("main", "localhost", 5672, "user", "pass", "/")
```

### MQTT

```go
mqtt.RegisterMqtt("main", "tcp://localhost:1883", "client-id", "user", "pass")
```

### Elasticsearch

```go
elasticsearch.RegisterElasticSearch("main", "http://localhost:9200", "user", "pass")
```

## Middleware

```go
// Recovery
a.Use(middleware.Recovery())

// Logging
a.Use(middleware.Logger())

// CORS
a.Use(middleware.DefaultCORS())

// Rate limiting (requests per second, burst size)
a.Use(middleware.RateLimit(100, 10)) // 100 req/s with burst of 10

// Compression
a.Use(middleware.Compress())

// Metrics
a.Use(middleware.Metrics())

// Authentication
api := a.Group("/api", auth.BearerAuth(jwtManager))
```

## Authentication

### JWT

```go
jwtManager := auth.NewJWTManager("secret-key", 24*time.Hour)

// Generate token
token, _ := jwtManager.GenerateToken("user-id", "username", "role", nil)

// Protect routes
protected := a.Group("/api", auth.BearerAuth(jwtManager))
```

### Basic Auth

```go
validator := func(username, password string) bool {
    return username == "admin" && password == "secret"
}
a.Use(auth.BasicAuth(validator))
```

## IoC Container

```go
// Register services
container := a.Container()

container.Singleton("database", func(c *container.Container) (interface{}, error) {
    return database.Get("main")
})

container.Singleton("userService", func(c *container.Container) (interface{}, error) {
    db, _ := c.Resolve("database")
    return NewUserService(db), nil
})

// Resolve services
service, _ := container.Resolve("userService")

// Auto-inject
type Handler struct {
    UserService *UserService `inject:"userService"`
}
container.Inject(&handler)
```

## Testing

```bash
# Run tests
make test

# With coverage
make test-coverage
```

## Metrics

Prometheus metrics available at `/metrics`:

```go
a.Use(middleware.Metrics())
a.Router().Handle("/metrics", middleware.MetricsHandler())
```

Metrics include:
- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request duration

## WebSocket

```go
hub := websocket.NewHub()
go hub.Run()

a.Router().HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
    hub.Upgrade(w, r, "user-id")
})

// Broadcast message
hub.Broadcast([]byte("Hello, everyone!"))
```

## Deployment

### Environment Variables

```bash
APP_ENV=production
LOG_LEVEL=info
PORT=8080
```

### Systemd Service

```ini
[Unit]
Description=GoFrame Application
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/myapp
ExecStart=/opt/myapp/bin/app
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## Contributing

Contributions are welcome! Please read our contributing guidelines.

## License

MIT License - see [LICENSE](LICENSE) file for details

## Acknowledgments

Built with these amazing libraries:
- [Gorilla Mux](https://github.com/gorilla/mux) - HTTP router
- [GORM](https://gorm.io) - ORM library
- [Redis](https://github.com/go-redis/redis) - Redis client
- [MongoDB Driver](https://github.com/mongodb/mongo-go-driver) - MongoDB driver
- [Elasticsearch](https://github.com/olivere/elastic) - Elasticsearch client
- [RabbitMQ](https://github.com/streadway/amqp) - AMQP client
- [Paho MQTT](https://github.com/eclipse/paho.mqtt.golang) - MQTT client

## Support

- Documentation: [docs/DOCUMENTATION.md](docs/DOCUMENTATION.md)
- Issues: [GitHub Issues](https://github.com/polymatx/goframe/issues)
- Discussions: [GitHub Discussions](https://github.com/polymatx/goframe/discussions)

---

Made with ❤️ by [Polymatx](https://polymatx.dev)
