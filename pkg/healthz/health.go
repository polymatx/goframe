package healthz

import (
	"context"
	"sync"
)

type Healthy interface {
	Health(ctx context.Context) error
}

var (
	all  []Healthy
	lock sync.RWMutex
)

// Register add a new health check service to system
func Register(checker ...Healthy) {
	lock.Lock()
	defer lock.Unlock()

	all = append(all, checker...)
}
