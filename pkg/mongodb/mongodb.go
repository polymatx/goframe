package mongodb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	clients     = make(map[string]*Client)
	clientsLock = &sync.RWMutex{}
	once        = &sync.Once{}
	configs     = make([]Config, 0)
)

// Config holds MongoDB connection configuration
type Config struct {
	Name                   string
	URI                    string
	Database               string
	MaxPoolSize            uint64
	MinPoolSize            uint64
	ConnectTimeout         time.Duration
	SocketTimeout          time.Duration
	ServerSelectionTimeout time.Duration
}

// Client wraps mongo.Client with additional methods
type Client struct {
	client   *mongo.Client
	database *mongo.Database
	name     string
	dbName   string
}

// Register registers a MongoDB connection
func Register(cfg Config) {
	// Set defaults
	if cfg.MaxPoolSize == 0 {
		cfg.MaxPoolSize = 100
	}
	if cfg.MinPoolSize == 0 {
		cfg.MinPoolSize = 10
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.SocketTimeout == 0 {
		cfg.SocketTimeout = 30 * time.Second
	}
	if cfg.ServerSelectionTimeout == 0 {
		cfg.ServerSelectionTimeout = 10 * time.Second
	}

	configs = append(configs, cfg)
}

// Initialize initializes all MongoDB connections
func Initialize(ctx context.Context) error {
	var initErr error
	once.Do(func() {
		for _, cfg := range configs {
			clientOpts := options.Client().
				ApplyURI(cfg.URI).
				SetMaxPoolSize(cfg.MaxPoolSize).
				SetMinPoolSize(cfg.MinPoolSize).
				SetConnectTimeout(cfg.ConnectTimeout).
				SetSocketTimeout(cfg.SocketTimeout).
				SetServerSelectionTimeout(cfg.ServerSelectionTimeout)

			client, err := mongo.Connect(ctx, clientOpts)
			if err != nil {
				logrus.Errorf("Failed to connect to MongoDB %s: %v", cfg.Name, err)
				initErr = err
				return
			}

			// Ping to verify connection
			if err := client.Ping(ctx, readpref.Primary()); err != nil {
				logrus.Errorf("Failed to ping MongoDB %s: %v", cfg.Name, err)
				initErr = err
				return
			}

			clientsLock.Lock()
			clients[cfg.Name] = &Client{
				client:   client,
				database: client.Database(cfg.Database),
				name:     cfg.Name,
				dbName:   cfg.Database,
			}
			clientsLock.Unlock()

			logrus.Infof("Successfully connected to MongoDB: %s (database: %s)", cfg.Name, cfg.Database)
		}
	})
	return initErr
}

// Get returns MongoDB client by name
func Get(name string) (*Client, error) {
	clientsLock.RLock()
	defer clientsLock.RUnlock()

	client, ok := clients[name]
	if !ok {
		return nil, fmt.Errorf("mongodb connection '%s' not found", name)
	}

	return client, nil
}

// MustGet returns client or panics
func MustGet(name string) *Client {
	client, err := Get(name)
	if err != nil {
		panic(err)
	}
	return client
}

// Client returns the underlying mongo.Client
func (c *Client) Client() *mongo.Client {
	return c.client
}

// Database returns the database instance
func (c *Client) Database() *mongo.Database {
	return c.database
}

// Collection returns a collection
func (c *Client) Collection(name string) *mongo.Collection {
	return c.database.Collection(name)
}

// Close closes the MongoDB connection
func (c *Client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// Ping checks the connection
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx, readpref.Primary())
}

// StartSession starts a new session
func (c *Client) StartSession(opts ...*options.SessionOptions) (mongo.Session, error) {
	return c.client.StartSession(opts...)
}

// UseSession executes a function within a session
func (c *Client) UseSession(ctx context.Context, fn func(mongo.SessionContext) error) error {
	session, err := c.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, fn)
}

// Transaction executes operations in a transaction
func (c *Client) Transaction(ctx context.Context, fn func(mongo.SessionContext) error) error {
	session, err := c.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(sessCtx)
	})

	return err
}

// CloseAll closes all MongoDB connections
func CloseAll(ctx context.Context) error {
	clientsLock.Lock()
	defer clientsLock.Unlock()

	for name, client := range clients {
		if err := client.Close(ctx); err != nil {
			logrus.Errorf("Failed to close MongoDB connection %s: %v", name, err)
			return err
		}
		logrus.Infof("Closed MongoDB connection: %s", name)
	}

	return nil
}
