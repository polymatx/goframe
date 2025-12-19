package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Driver represents supported database drivers
type Driver string

const (
	// MySQL driver
	MySQL Driver = "mysql"
	// PostgreSQL driver
	PostgreSQL Driver = "postgres"
	// SQLite driver
	SQLite Driver = "sqlite"
)

// Config holds database connection configuration
type Config struct {
	Name     string // Connection name
	Driver   Driver // Database driver
	Host     string // Database host
	Port     int    // Database port
	User     string // Database user
	Password string // Database password
	Database string // Database name
	DSN      string // Custom DSN (overrides other fields if set)

	// Connection pool settings
	MaxIdleConns    int           // Maximum number of idle connections
	MaxOpenConns    int           // Maximum number of open connections
	ConnMaxLifetime time.Duration // Maximum lifetime of a connection
	ConnMaxIdleTime time.Duration // Maximum idle time of a connection

	// GORM settings
	LogLevel                    logger.LogLevel // Log level for SQL queries
	SkipDefaultTx               bool            // Skip default transaction for single operations
	PrepareStmt                 bool            // Prepare statements and cache them
	DisableForeignKeyConstraint bool            // Disable foreign key constraints
}

// Connection represents a database connection manager
type Connection struct {
	db     *gorm.DB
	config Config
	mu     sync.RWMutex
}

var (
	connections     = make(map[string]*Connection)
	connectionsLock sync.RWMutex
	once            sync.Once
	configs         []Config
)

// Register adds a database configuration to be initialized later
func Register(config Config) error {
	if config.Name == "" {
		return fmt.Errorf("database config name cannot be empty")
	}

	if config.Driver == "" {
		return fmt.Errorf("database driver cannot be empty")
	}

	// Set defaults
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 10
	}
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 100
	}
	if config.ConnMaxLifetime == 0 {
		config.ConnMaxLifetime = time.Hour
	}
	if config.ConnMaxIdleTime == 0 {
		config.ConnMaxIdleTime = 10 * time.Minute
	}

	configs = append(configs, config)
	return nil
}

// Initialize establishes all registered database connections
func Initialize(ctx context.Context) error {
	var initErr error

	once.Do(func() {
		for _, config := range configs {
			if err := connect(ctx, config); err != nil {
				initErr = err
				return
			}
		}
	})

	return initErr
}

func connect(ctx context.Context, config Config) error {
	var dialector gorm.Dialector
	var dsn string

	// Build DSN based on driver
	if config.DSN != "" {
		dsn = config.DSN
	} else {
		switch config.Driver {
		case MySQL:
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				config.User, config.Password, config.Host, config.Port, config.Database)
		case PostgreSQL:
			dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
				config.Host, config.Port, config.User, config.Password, config.Database)
		case SQLite:
			dsn = config.Database // For SQLite, database is the file path
		default:
			return fmt.Errorf("unsupported driver: %s", config.Driver)
		}
	}

	// Create dialector
	switch config.Driver {
	case MySQL:
		dialector = mysql.Open(dsn)
	case PostgreSQL:
		dialector = postgres.Open(dsn)
	case SQLite:
		dialector = sqlite.Open(dsn)
	default:
		return fmt.Errorf("unsupported driver: %s", config.Driver)
	}

	// Configure GORM
	gormConfig := &gorm.Config{
		SkipDefaultTransaction:                   config.SkipDefaultTx,
		PrepareStmt:                              config.PrepareStmt,
		DisableForeignKeyConstraintWhenMigrating: config.DisableForeignKeyConstraint,
	}

	// Set log level
	if config.LogLevel == 0 {
		if viper.GetBool("develop_mode") {
			config.LogLevel = logger.Info
		} else {
			config.LogLevel = logger.Warn
		}
	}

	gormConfig.Logger = logger.New(
		logrus.StandardLogger(),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  config.LogLevel,
			IgnoreRecordNotFoundError: false,
			Colorful:                  viper.GetBool("develop_mode"),
		},
	)

	// Open connection
	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database '%s': %w", config.Name, err)
	}

	// Get underlying sql.DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB for '%s': %w", config.Name, err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test connection
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database '%s': %w", config.Name, err)
	}

	// Store connection
	conn := &Connection{
		db:     db,
		config: config,
	}

	connectionsLock.Lock()
	connections[config.Name] = conn
	connectionsLock.Unlock()

	logrus.Infof("Successfully connected to %s database: %s", config.Driver, config.Name)

	return nil
}

// Get returns a database connection by name
func Get(name string) (*Connection, error) {
	connectionsLock.RLock()
	defer connectionsLock.RUnlock()

	conn, exists := connections[name]
	if !exists {
		return nil, fmt.Errorf("database connection '%s' not found", name)
	}

	if conn == nil {
		return nil, fmt.Errorf("database connection '%s' is nil", name)
	}

	return conn, nil
}

// MustGet returns a database connection or panics if not found
// Deprecated: Use Get instead and handle errors properly
func MustGet(name string) *Connection {
	conn, err := Get(name)
	if err != nil {
		panic(err)
	}
	return conn
}

// DB returns the underlying GORM database instance
func (c *Connection) DB() *gorm.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db
}

// SqlDB returns the underlying sql.DB instance
func (c *Connection) SqlDB() (*sql.DB, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db.DB()
}

// WithContext returns a new GORM DB with the given context
func (c *Connection) WithContext(ctx context.Context) *gorm.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db.WithContext(ctx)
}

// Transaction executes a function within a database transaction
func (c *Connection) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db.WithContext(ctx).Transaction(fn)
}

// Begin starts a manual transaction
func (c *Connection) Begin(ctx context.Context) *gorm.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db.WithContext(ctx).Begin()
}

// AutoMigrate runs auto migration for given models
func (c *Connection) AutoMigrate(models ...interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db.AutoMigrate(models...)
}

// Close closes all database connections
func Close() error {
	connectionsLock.Lock()
	defer connectionsLock.Unlock()

	var errs []error
	for name, conn := range connections {
		if conn == nil || conn.db == nil {
			continue
		}

		sqlDB, err := conn.db.DB()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get sql.DB for '%s': %w", name, err))
			continue
		}

		if err := sqlDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection '%s': %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

// Health checks if the database connection is healthy
func (c *Connection) Health(ctx context.Context) error {
	sqlDB, err := c.SqlDB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Stats returns database connection pool statistics
func (c *Connection) Stats() sql.DBStats {
	sqlDB, _ := c.SqlDB()
	if sqlDB == nil {
		return sql.DBStats{}
	}
	return sqlDB.Stats()
}
