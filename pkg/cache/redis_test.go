package cache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

const testCacheName = "test-cache"

var (
	testCache *Manager
	testAddr  string
)

// TestMain starts the in-process fake Redis server and initializes the
// package-global registry exactly once: Initialize is guarded by sync.Once,
// so it can only ever connect the configs registered before its first call.
func TestMain(m *testing.M) {
	srv, err := startFakeRedis()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start fake redis: %v\n", err)
		os.Exit(1)
	}
	testAddr = srv.Addr()

	fatal := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
		srv.Close()
		os.Exit(1)
	}

	if err := Register(Config{
		Name:  testCacheName,
		Addrs: []string{testAddr},
		Mode:  ModeStandalone,
	}); err != nil {
		fatal("failed to register cache: %v", err)
	}

	if err := Initialize(context.Background()); err != nil {
		fatal("failed to initialize cache: %v", err)
	}

	testCache, err = Get(testCacheName)
	if err != nil {
		fatal("failed to get cache manager: %v", err)
	}

	code := m.Run()

	if err := Close(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to close cache connections: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	srv.Close()
	os.Exit(code)
}

func TestRegister_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{"empty name", Config{Addrs: []string{"127.0.0.1:6379"}}, true},
		{"no addresses", Config{Name: "no-addrs"}, true},
		{"invalid mode", Config{Name: "bad-mode", Addrs: []string{"127.0.0.1:6379"}, Mode: "sentinel"}, true},
		{"valid standalone", Config{Name: "reg-standalone", Addrs: []string{"127.0.0.1:6379"}, Mode: ModeStandalone}, false},
		{"valid cluster", Config{Name: "reg-cluster", Addrs: []string{"127.0.0.1:6379", "127.0.0.1:6380"}, Mode: ModeCluster}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Register(tt.config)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRegister_Defaults(t *testing.T) {
	if err := Register(Config{
		Name:  "reg-defaults",
		Addrs: []string{"127.0.0.1:6379"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var registered *Config
	for i := range configs {
		if configs[i].Name == "reg-defaults" {
			registered = &configs[i]
			break
		}
	}
	if registered == nil {
		t.Fatal("expected config to be registered")
	}
	if registered.Mode != ModeStandalone {
		t.Errorf("expected default mode %q, got %q", ModeStandalone, registered.Mode)
	}
	if registered.PoolSize != 10 {
		t.Errorf("expected default pool size 10, got %d", registered.PoolSize)
	}
	if registered.Timeout != 5*time.Second {
		t.Errorf("expected default timeout 5s, got %v", registered.Timeout)
	}
}

func TestGet(t *testing.T) {
	t.Run("existing manager", func(t *testing.T) {
		mgr, err := Get(testCacheName)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mgr == nil {
			t.Fatal("expected non-nil manager")
		}
		if mgr.Client() == nil {
			t.Error("expected non-nil underlying client")
		}
	})

	t.Run("unknown name returns error", func(t *testing.T) {
		mgr, err := Get("does-not-exist")
		if err == nil {
			t.Fatal("expected error for unknown cache connection")
		}
		if mgr != nil {
			t.Errorf("expected nil manager, got %v", mgr)
		}
	})
}

func TestMustGet(t *testing.T) {
	t.Run("returns existing manager", func(t *testing.T) {
		if MustGet(testCacheName) == nil {
			t.Fatal("expected non-nil manager")
		}
	})

	t.Run("panics for unknown manager", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for unknown cache connection")
			}
		}()
		MustGet("does-not-exist")
	})
}

func TestInitialize_RunsOnlyOnce(t *testing.T) {
	// Initialize already ran in TestMain; sync.Once means configs registered
	// afterwards are never connected, and re-running Initialize is a no-op.
	if err := Register(Config{
		Name:  "late-register",
		Addrs: []string{testAddr},
		Mode:  ModeStandalone,
	}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	if err := Initialize(context.Background()); err != nil {
		t.Fatalf("unexpected error from repeated Initialize: %v", err)
	}

	if _, err := Get("late-register"); err == nil {
		t.Error("expected late-registered connection to be unavailable: Initialize only connects configs registered before its first call")
	}
}

func TestLegacyHelpers(t *testing.T) {
	t.Run("GetRedisConn", func(t *testing.T) {
		mgr, err := GetRedisConn(testCacheName)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mgr != testCache {
			t.Error("expected same manager as Get")
		}
	})

	t.Run("MustGetRedisConn", func(t *testing.T) {
		if MustGetRedisConn(testCacheName) != testCache {
			t.Error("expected same manager as Get")
		}
	})

	t.Run("RegisterRedis splits addresses", func(t *testing.T) {
		err := RegisterRedis("legacy-cache", "127.0.0.1:7001,127.0.0.1:7002", "", "cluster", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var registered *Config
		for i := range configs {
			if configs[i].Name == "legacy-cache" {
				registered = &configs[i]
				break
			}
		}
		if registered == nil {
			t.Fatal("expected config to be registered")
		}
		if len(registered.Addrs) != 2 {
			t.Errorf("expected 2 addresses, got %v", registered.Addrs)
		}
		if registered.Mode != ModeCluster {
			t.Errorf("expected cluster mode, got %q", registered.Mode)
		}
	})

	t.Run("RegisterRedis rejects invalid mode", func(t *testing.T) {
		if err := RegisterRedis("legacy-bad", "127.0.0.1:7001", "", "bogus", 0); err == nil {
			t.Error("expected error for invalid mode")
		}
	})
}

func TestManager_Ping(t *testing.T) {
	if err := testCache.Ping(context.Background()); err != nil {
		t.Errorf("expected successful ping, got %v", err)
	}
}
