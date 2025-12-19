package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/polymatx/goframe/pkg/safe"
	"github.com/polymatx/goframe/pkg/xlog"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	connections     = make(map[string]*Connection)
	connectionsLock = &sync.RWMutex{}
	once            = sync.Once{}
	pendingConns    = make([]connectionConfig, 0)
	initializers    = make(map[string][]Initializer)
	initializerLock = &sync.RWMutex{}
)

// Initializer interface for post-connection initialization
type Initializer interface {
	Initialize()
}

// Connection represents a database connection manager
type Connection struct {
	db *gorm.DB
}

type connectionConfig struct {
	name     string
	host     string
	port     int
	user     string
	password string
	database string
}

// GetDB returns the underlying GORM database instance
func (c *Connection) GetDB() *gorm.DB {
	return c.db
}

// GetSqlDB returns the underlying sql.DB instance
func (c *Connection) GetSqlDB() *sql.DB {
	sqlDB, _ := c.db.DB()
	return sqlDB
}

// Begin starts a transaction
func (c *Connection) Begin() *Connection {
	return &Connection{db: c.db.Begin()}
}

// Commit commits the transaction
func (c *Connection) Commit() error {
	return c.db.Commit().Error
}

// Rollback rolls back the transaction
func (c *Connection) Rollback() error {
	return c.db.Rollback().Error
}

// WithContext returns a new Connection with the given context
func (c *Connection) WithContext(ctx context.Context) *Connection {
	return &Connection{db: c.db.WithContext(ctx)}
}

// RegisterMysql registers a MySQL connection to be initialized later
func RegisterMysql(name, host, user, password, database string, port int) {
	pendingConns = append(pendingConns, connectionConfig{
		name:     name,
		host:     host,
		port:     port,
		user:     user,
		password: password,
		database: database,
	})
}

// Initialize establishes all registered database connections
func Initialize(ctx context.Context) error {
	var initErr error

	once.Do(func() {
		initErr = safe.Try(func() error {
			for _, cfg := range pendingConns {
				if err := connectDatabase(ctx, cfg); err != nil {
					return err
				}
			}
			return nil
		}, 30*time.Second)
	})

	return initErr
}

func connectDatabase(ctx context.Context, cfg connectionConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.user,
		cfg.password,
		cfg.host,
		cfg.port,
		cfg.database,
	)

	// Configure GORM
	gormConfig := &gorm.Config{}

	// Enable SQL logging in development mode
	if viper.GetBool("develop_mode") {
		gormConfig.Logger = logger.New(
			logrus.StandardLogger(),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: false,
				Colorful:                  true,
			},
		)
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		xlog.GetWithError(ctx, err).Errorf("Failed to connect to database: %s", dsn)
		return err
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// Set connection pool settings
	maxIdleConns := viper.GetInt(fmt.Sprintf("%s_max_idle_conns", cfg.name))
	if maxIdleConns == 0 {
		maxIdleConns = 10
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)

	maxOpenConns := viper.GetInt(fmt.Sprintf("%s_max_open_conns", cfg.name))
	if maxOpenConns == 0 {
		maxOpenConns = 100
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)

	connMaxLifetime := viper.GetDuration(fmt.Sprintf("%s_conn_max_lifetime", cfg.name))
	if connMaxLifetime == 0 {
		connMaxLifetime = time.Hour
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		xlog.GetWithError(ctx, err).Errorf("Failed to ping database: %s", dsn)
		return err
	}

	// Store connection
	connectionsLock.Lock()
	connections[cfg.name] = &Connection{db: db}
	connectionsLock.Unlock()

	// Run post-initialization hooks
	initializerLock.RLock()
	inits, exists := initializers[cfg.name]
	initializerLock.RUnlock()

	if exists {
		for _, init := range inits {
			init.Initialize()
		}
	}

	logrus.Infof("Successfully connected to MySQL: %s@%s:%d/%s",
		cfg.user, cfg.host, cfg.port, cfg.database)

	return nil
}

// RegisterInitializer registers an initializer to run after database connection
func RegisterInitializer(connectionName string, init Initializer) {
	initializerLock.Lock()
	defer initializerLock.Unlock()

	if initializers[connectionName] == nil {
		initializers[connectionName] = make([]Initializer, 0)
	}
	initializers[connectionName] = append(initializers[connectionName], init)
}

// MustGetConnection returns a database connection or panics if not found
func MustGetConnection(ctx context.Context, name string) *Connection {
	connectionsLock.RLock()
	defer connectionsLock.RUnlock()

	conn, exists := connections[name]
	if !exists {
		panic(fmt.Sprintf("database connection '%s' not found", name))
	}

	if conn == nil {
		panic(fmt.Sprintf("database connection '%s' is nil", name))
	}

	return conn.WithContext(ctx)
}

// GetConnection returns a database connection or an error if not found
func GetConnection(ctx context.Context, name string) (*Connection, error) {
	connectionsLock.RLock()
	defer connectionsLock.RUnlock()

	conn, exists := connections[name]
	if !exists {
		return nil, fmt.Errorf("database connection '%s' not found", name)
	}

	if conn == nil {
		return nil, fmt.Errorf("database connection '%s' is nil", name)
	}

	return conn.WithContext(ctx), nil
}

// Close closes all database connections
func Close() error {
	connectionsLock.Lock()
	defer connectionsLock.Unlock()

	var errs []error
	for name, conn := range connections {
		if conn != nil {
			if sqlDB := conn.GetSqlDB(); sqlDB != nil {
				if err := sqlDB.Close(); err != nil {
					errs = append(errs, fmt.Errorf("failed to close connection '%s': %w", name, err))
				}
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

// Legacy compatibility - kept for backward compatibility
func MustGetMysqlConn(ctx context.Context, name string) *Connection {
	return MustGetConnection(ctx, name)
}

// GetConn is kept for backward compatibility
func (c *Connection) GetConn() *gorm.DB {
	return c.GetDB()
}
