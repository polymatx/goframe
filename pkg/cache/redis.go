package cache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/polymatx/goframe/pkg/xlog"
	"github.com/sirupsen/logrus"
)

// Mode represents the Redis connection mode
type Mode string

const (
	// ModeStandalone represents a single Redis instance
	ModeStandalone Mode = "standalone"
	// ModeCluster represents a Redis cluster
	ModeCluster Mode = "cluster"
)

// Config holds Redis connection configuration
type Config struct {
	Name     string        // Connection name
	Addrs    []string      // Redis addresses
	Password string        // Password for authentication
	DB       int           // Database number (only for standalone)
	Mode     Mode          // Connection mode (standalone or cluster)
	PoolSize int           // Connection pool size
	Timeout  time.Duration // Connection timeout
}

// Manager provides Redis operations
type Manager struct {
	client redis.Cmdable
	config Config
}

var (
	once        sync.Once
	clients     = make(map[string]*Manager)
	clientsLock sync.RWMutex
	configs     []Config

	// ErrNotFound is returned when a cache key doesn't exist
	ErrNotFound = redis.Nil
)

// Register adds a Redis configuration to be initialized later
func Register(config Config) error {
	if config.Name == "" {
		return fmt.Errorf("cache config name cannot be empty")
	}

	if len(config.Addrs) == 0 {
		return fmt.Errorf("cache config must have at least one address")
	}

	if config.Mode == "" {
		config.Mode = ModeStandalone
	}

	if config.Mode != ModeStandalone && config.Mode != ModeCluster {
		return fmt.Errorf("invalid cache mode: %s", config.Mode)
	}

	if config.PoolSize == 0 {
		config.PoolSize = 10
	}

	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	configs = append(configs, config)
	return nil
}

// Initialize establishes all registered Redis connections
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
	var client redis.Cmdable

	if config.Mode == ModeCluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        config.Addrs,
			Password:     config.Password,
			PoolSize:     config.PoolSize,
			DialTimeout:  config.Timeout,
			ReadTimeout:  config.Timeout,
			WriteTimeout: config.Timeout,
		})
	} else {
		addr := config.Addrs[0]
		if len(config.Addrs) > 1 {
			logrus.Warnf("Multiple addresses provided for standalone mode, using first: %s", addr)
		}

		client = redis.NewClient(&redis.Options{
			Addr:         addr,
			Password:     config.Password,
			DB:           config.DB,
			PoolSize:     config.PoolSize,
			DialTimeout:  config.Timeout,
			ReadTimeout:  config.Timeout,
			WriteTimeout: config.Timeout,
		})
	}

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		xlog.GetWithError(ctx, err).Errorf("Failed to connect to Redis: %s", config.Name)
		return fmt.Errorf("failed to connect to redis '%s': %w", config.Name, err)
	}

	manager := &Manager{
		client: client,
		config: config,
	}

	clientsLock.Lock()
	clients[config.Name] = manager
	clientsLock.Unlock()

	logrus.Infof("Successfully connected to Redis (%s): %s", config.Mode, strings.Join(config.Addrs, ","))

	return nil
}

// Get returns a cache manager by name
func Get(name string) (*Manager, error) {
	clientsLock.RLock()
	defer clientsLock.RUnlock()

	manager, exists := clients[name]
	if !exists {
		return nil, fmt.Errorf("cache connection '%s' not found", name)
	}

	if manager == nil {
		return nil, fmt.Errorf("cache connection '%s' is nil", name)
	}

	return manager, nil
}

// MustGet returns a cache manager by name or panics if not found
// Deprecated: Use Get instead
func MustGet(name string) *Manager {
	manager, err := Get(name)
	if err != nil {
		panic(err)
	}
	return manager
}

// Client returns the underlying redis client
func (m *Manager) Client() redis.Cmdable {
	return m.client
}

// Close closes all Redis connections
func Close() error {
	clientsLock.Lock()
	defer clientsLock.Unlock()

	var errs []error
	for name, manager := range clients {
		if manager == nil || manager.client == nil {
			continue
		}

		var err error
		switch c := manager.client.(type) {
		case *redis.Client:
			err = c.Close()
		case *redis.ClusterClient:
			err = c.Close()
		}

		if err != nil {
			errs = append(errs, fmt.Errorf("failed to close redis connection '%s': %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing cache connections: %v", errs)
	}

	return nil
}

// Legacy compatibility - deprecated
// Deprecated: Use Register instead
func RegisterRedis(name string, addrs string, password string, mode string, database int) error {
	return Register(Config{
		Name:     name,
		Addrs:    strings.Split(addrs, ","),
		Password: password,
		DB:       database,
		Mode:     Mode(mode),
	})
}

// Deprecated: Use Get instead
func GetRedisConn(name string) (*Manager, error) {
	return Get(name)
}

// Deprecated: Use MustGet instead
func MustGetRedisConn(name string) *Manager {
	return MustGet(name)
}
