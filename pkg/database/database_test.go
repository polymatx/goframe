package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	testConnName = "test-main"
	testDSNName  = "test-dsn"
)

// testUser exercises AutoMigrate plus the custom column types from types.go.
type testUser struct {
	ID       uint             `gorm:"primaryKey"`
	Name     string           `gorm:"size:64"`
	Nickname NullString       `gorm:"type:text"`
	Age      NullInt64        `gorm:"type:integer"`
	Score    NullFloat64      `gorm:"type:real"`
	Active   NullBool         `gorm:"type:boolean"`
	LastSeen NullTime         `gorm:"type:datetime"`
	TagIDs   Int64Slice       `gorm:"type:text"`
	Meta     GenericJSONField `gorm:"type:text"`
}

// TestMain registers and initializes all test connections exactly once:
// Initialize is guarded by sync.Once, so it can only ever connect the
// configs registered before its first call in a given process.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "goframe-database-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	fatal := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}

	if err := Register(Config{
		Name:     testConnName,
		Driver:   SQLite,
		Database: filepath.Join(dir, "main.db"),
		LogLevel: logger.Silent,
	}); err != nil {
		fatal("failed to register %s: %v", testConnName, err)
	}

	// Second connection configured through a custom DSN.
	if err := Register(Config{
		Name:     testDSNName,
		Driver:   SQLite,
		DSN:      filepath.Join(dir, "dsn.db"),
		LogLevel: logger.Silent,
	}); err != nil {
		fatal("failed to register %s: %v", testDSNName, err)
	}

	if err := Initialize(context.Background()); err != nil {
		fatal("failed to initialize databases: %v", err)
	}

	code := m.Run()

	if err := Close(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to close connections: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func mustConn(t *testing.T) *Connection {
	t.Helper()
	conn, err := Get(testConnName)
	if err != nil {
		t.Fatalf("failed to get connection %q: %v", testConnName, err)
	}
	return conn
}

func TestRegister_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{"empty name", Config{Driver: SQLite, Database: "ignored.db"}, true},
		{"empty driver", Config{Name: "no-driver"}, true},
		{"valid config", Config{Name: "register-valid", Driver: SQLite, Database: "unused.db"}, false},
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

func TestGet(t *testing.T) {
	t.Run("existing connections", func(t *testing.T) {
		for _, name := range []string{testConnName, testDSNName} {
			conn, err := Get(name)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", name, err)
			}
			if conn == nil {
				t.Fatalf("expected non-nil connection for %q", name)
			}
		}
	})

	t.Run("unknown name returns error", func(t *testing.T) {
		conn, err := Get("does-not-exist")
		if err == nil {
			t.Fatal("expected error for unknown connection")
		}
		if conn != nil {
			t.Errorf("expected nil connection, got %v", conn)
		}
	})
}

func TestMustGet(t *testing.T) {
	t.Run("returns existing connection", func(t *testing.T) {
		conn := MustGet(testConnName)
		if conn == nil {
			t.Fatal("expected non-nil connection")
		}
	})

	t.Run("panics for unknown connection", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for unknown connection")
			}
		}()
		MustGet("does-not-exist")
	})
}

func TestInitialize_RunsOnlyOnce(t *testing.T) {
	// Initialize already ran in TestMain; sync.Once means configs registered
	// afterwards are never connected, and re-running Initialize is a no-op.
	dir := t.TempDir()
	if err := Register(Config{
		Name:     "late-register",
		Driver:   SQLite,
		Database: filepath.Join(dir, "late.db"),
		LogLevel: logger.Silent,
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

func TestConnection_Accessors(t *testing.T) {
	conn := mustConn(t)

	t.Run("DB", func(t *testing.T) {
		if conn.DB() == nil {
			t.Fatal("expected non-nil gorm.DB")
		}
	})

	t.Run("SqlDB", func(t *testing.T) {
		sqlDB, err := conn.SqlDB()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sqlDB == nil {
			t.Fatal("expected non-nil sql.DB")
		}
	})

	t.Run("WithContext", func(t *testing.T) {
		if conn.WithContext(context.Background()) == nil {
			t.Fatal("expected non-nil gorm.DB")
		}
	})

	t.Run("Stats reflects pool defaults", func(t *testing.T) {
		// Register defaults MaxOpenConns to 100 when unset.
		stats := conn.Stats()
		if stats.MaxOpenConnections != 100 {
			t.Errorf("expected MaxOpenConnections 100, got %d", stats.MaxOpenConnections)
		}
	})

	t.Run("Health", func(t *testing.T) {
		if err := conn.Health(context.Background()); err != nil {
			t.Errorf("expected healthy connection, got %v", err)
		}
	})
}

func TestConnection_AutoMigrateAndCRUD(t *testing.T) {
	conn := mustConn(t)
	db := conn.DB()

	if err := conn.AutoMigrate(&testUser{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	lastSeen := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)
	user := testUser{
		Name:     "alice",
		Nickname: NullString{Valid: true, String: "al"},
		Age:      NullInt64{Valid: true, Int64: 30},
		Score:    NullFloat64{Valid: true, Float64: 99.5},
		Active:   NullBool{Valid: true, Bool: true},
		LastSeen: NullTime{Valid: true, Time: lastSeen},
		TagIDs:   Int64Slice{1, 2, 3},
		Meta:     GenericJSONField{"role": "admin"},
	}

	t.Run("create", func(t *testing.T) {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create failed: %v", err)
		}
		if user.ID == 0 {
			t.Fatal("expected auto-assigned ID")
		}
	})

	t.Run("read back custom types", func(t *testing.T) {
		var got testUser
		if err := db.First(&got, user.ID).Error; err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if got.Name != "alice" {
			t.Errorf("expected name alice, got %q", got.Name)
		}
		if !got.Nickname.Valid || got.Nickname.String != "al" {
			t.Errorf("unexpected Nickname: %+v", got.Nickname)
		}
		if !got.Age.Valid || got.Age.Int64 != 30 {
			t.Errorf("unexpected Age: %+v", got.Age)
		}
		if !got.Score.Valid || got.Score.Float64 != 99.5 {
			t.Errorf("unexpected Score: %+v", got.Score)
		}
		if !got.Active.Valid || !got.Active.Bool {
			t.Errorf("unexpected Active: %+v", got.Active)
		}
		if !got.LastSeen.Valid || !got.LastSeen.Time.Equal(lastSeen) {
			t.Errorf("unexpected LastSeen: %+v", got.LastSeen)
		}
		if len(got.TagIDs) != 3 || got.TagIDs[0] != 1 || got.TagIDs[2] != 3 {
			t.Errorf("unexpected TagIDs: %v", got.TagIDs)
		}
		if got.Meta["role"] != "admin" {
			t.Errorf("unexpected Meta: %v", got.Meta)
		}
	})

	t.Run("null custom types roundtrip", func(t *testing.T) {
		nullUser := testUser{
			Name:   "bob",
			TagIDs: Int64Slice{},
			Meta:   GenericJSONField{},
		}
		if err := db.Create(&nullUser).Error; err != nil {
			t.Fatalf("create failed: %v", err)
		}

		var got testUser
		if err := db.First(&got, nullUser.ID).Error; err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if got.Nickname.Valid || got.Age.Valid || got.Score.Valid || got.Active.Valid || got.LastSeen.Valid {
			t.Errorf("expected all nullable fields invalid, got %+v", got)
		}
	})

	t.Run("update", func(t *testing.T) {
		if err := db.Model(&testUser{}).Where("id = ?", user.ID).
			Update("name", "alice-updated").Error; err != nil {
			t.Fatalf("update failed: %v", err)
		}
		var got testUser
		if err := db.First(&got, user.ID).Error; err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if got.Name != "alice-updated" {
			t.Errorf("expected updated name, got %q", got.Name)
		}
	})

	t.Run("delete", func(t *testing.T) {
		if err := db.Delete(&testUser{}, user.ID).Error; err != nil {
			t.Fatalf("delete failed: %v", err)
		}
		var got testUser
		err := db.First(&got, user.ID).Error
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Errorf("expected ErrRecordNotFound, got %v", err)
		}
	})
}

func TestConnection_Transaction(t *testing.T) {
	conn := mustConn(t)
	ctx := context.Background()

	if err := conn.AutoMigrate(&testUser{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	t.Run("commit persists changes", func(t *testing.T) {
		var id uint
		err := conn.Transaction(ctx, func(tx *gorm.DB) error {
			u := testUser{Name: "tx-commit", TagIDs: Int64Slice{}, Meta: GenericJSONField{}}
			if err := tx.Create(&u).Error; err != nil {
				return err
			}
			id = u.ID
			return nil
		})
		if err != nil {
			t.Fatalf("transaction failed: %v", err)
		}

		var got testUser
		if err := conn.DB().First(&got, id).Error; err != nil {
			t.Errorf("expected committed row, got error: %v", err)
		}
	})

	t.Run("error rolls back changes", func(t *testing.T) {
		sentinel := errors.New("boom")
		var id uint
		err := conn.Transaction(ctx, func(tx *gorm.DB) error {
			u := testUser{Name: "tx-rollback", TagIDs: Int64Slice{}, Meta: GenericJSONField{}}
			if err := tx.Create(&u).Error; err != nil {
				return err
			}
			id = u.ID
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected sentinel error, got %v", err)
		}

		var got testUser
		readErr := conn.DB().First(&got, id).Error
		if !errors.Is(readErr, gorm.ErrRecordNotFound) {
			t.Errorf("expected rolled-back row to be absent, got %v", readErr)
		}
	})

	t.Run("manual Begin and Rollback", func(t *testing.T) {
		tx := conn.Begin(ctx)
		if tx.Error != nil {
			t.Fatalf("begin failed: %v", tx.Error)
		}
		u := testUser{Name: "manual-tx", TagIDs: Int64Slice{}, Meta: GenericJSONField{}}
		if err := tx.Create(&u).Error; err != nil {
			t.Fatalf("create failed: %v", err)
		}
		if err := tx.Rollback().Error; err != nil {
			t.Fatalf("rollback failed: %v", err)
		}

		var got testUser
		readErr := conn.DB().First(&got, u.ID).Error
		if !errors.Is(readErr, gorm.ErrRecordNotFound) {
			t.Errorf("expected rolled-back row to be absent, got %v", readErr)
		}
	})
}
