package bredis

import (
	"context"
	"strings"

	"github.com/polymatx/goframe/pkg/cache"
)

// Manager is deprecated. Use cache.Manager instead.
type Manager = cache.Manager

// Initializer is deprecated.
type Initializer interface {
	Initialize()
}

// RegisterRedis is deprecated. Use cache.Register instead.
func RegisterRedis(cnt, host, password, kind string, database int) error {
	return cache.Register(cache.Config{
		Name:     cnt,
		Addrs:    strings.Split(host, ","),
		Password: password,
		DB:       database,
		Mode:     cache.Mode(kind),
	})
}

// Initialize is deprecated. Use cache.Initialize instead.
func Initialize(ctx context.Context) {
	_ = cache.Initialize(ctx)
}

// MustGetRedisConn is deprecated. Use cache.Get instead.
func MustGetRedisConn(cnt string) Manager {
	return *cache.MustGet(cnt)
}

// GetRedisConn is deprecated. Use cache.Get instead.
func GetRedisConn(cnt string) (Manager, error) {
	mgr, err := cache.Get(cnt)
	if err != nil {
		return Manager{}, err
	}
	return *mgr, nil
}

// Close is deprecated. Use cache.Close instead.
func Close() error {
	return cache.Close()
}

// Register is deprecated.
func Register(cnt string, m ...Initializer) {
	// No-op for backward compatibility
}
