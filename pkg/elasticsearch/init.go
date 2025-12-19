package elasticsearch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/polymatx/goframe/pkg/assert"
	"github.com/polymatx/goframe/pkg/safe"
	"github.com/polymatx/goframe/pkg/xlog"
	"github.com/sirupsen/logrus"
)

var (
	clients             = make(map[string]*Client)
	clientLock          = &sync.RWMutex{}
	once                = &sync.Once{}
	elasticConnExpected = make([]elasticConfig, 0)

	all  map[string][]Initializer
	lock sync.RWMutex
)

type elasticConfig struct {
	name     string
	url      string
	username string
	password string
}

// Initializer interface for post-connection initialization
type Initializer interface {
	Initialize()
}

// RegisterElasticSearch registers Elasticsearch connection
func RegisterElasticSearch(name, url, username, password string) {
	elasticConnExpected = append(elasticConnExpected, elasticConfig{
		name:     name,
		url:      url,
		username: username,
		password: password,
	})
}

// Initialize initializes all Elasticsearch connections
func Initialize(ctx context.Context) error {
	var initErr error
	once.Do(func() {
		_ = safe.Try(func() error {
			for _, cfg := range elasticConnExpected {
				opts := []elastic.ClientOptionFunc{
					elastic.SetURL(cfg.url),
					elastic.SetSniff(false),
					elastic.SetHealthcheck(false),
				}

				if cfg.username != "" && cfg.password != "" {
					opts = append(opts, elastic.SetBasicAuth(cfg.username, cfg.password))
				}

				client, err := elastic.NewClient(opts...)
				if err != nil {
					xlog.GetWithError(ctx, errors.New("connect to elasticsearch failed")).Error(err)
					initErr = err
					return err
				}

				_, _, err = client.Ping(cfg.url).Do(ctx)
				if err != nil {
					xlog.GetWithError(ctx, errors.New("ping to elasticsearch failed")).Error(err)
					initErr = err
					return err
				}

				clientLock.Lock()
				clients[cfg.name] = NewClient(client)
				clientLock.Unlock()

				logrus.Infof("successfully connected to elasticsearch: %s", cfg.url)
			}
			return nil
		}, 30*time.Second)
	})
	return initErr
}

// GetElasticSearchConnection returns Elasticsearch client by name
func GetElasticSearchConnection(name string) (*Client, error) {
	clientLock.RLock()
	defer clientLock.RUnlock()

	client, ok := clients[name]
	if !ok {
		return nil, fmt.Errorf("elasticsearch connection '%s' not found", name)
	}

	return client, nil
}

// MustGetElasticClient returns client or panics
func MustGetElasticClient(name string) *Client {
	clientLock.RLock()
	defer clientLock.RUnlock()
	val, ok := clients[name]
	assert.True(ok)
	assert.NotNil(val)
	return val
}

// RegisterElastic (deprecated, use RegisterElasticSearch)
func RegisterElastic(cnt, host string, port int) error {
	url := fmt.Sprintf("http://%s:%d", host, port)
	RegisterElasticSearch(cnt, url, "", "")
	return nil
}

// Register an initializer to run after elastic is loaded
func Register(cnt string, m ...Initializer) {
	lock.Lock()
	if all == nil {
		all = make(map[string][]Initializer)
	}
	all[cnt] = append(all[cnt], m...)
	lock.Unlock()
}
