package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewApp(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		app := New(nil)
		if app == nil {
			t.Fatal("expected app to be created")
		}
		if app.config.Port != ":8080" {
			t.Errorf("expected default port :8080, got %s", app.config.Port)
		}
		if app.config.Name != "goframe-app" {
			t.Errorf("expected default name goframe-app, got %s", app.config.Name)
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &Config{
			Name:            "test-app",
			Port:            ":3000",
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 15 * time.Second,
		}
		app := New(cfg)
		if app.config.Port != ":3000" {
			t.Errorf("expected port :3000, got %s", app.config.Port)
		}
		if app.config.Name != "test-app" {
			t.Errorf("expected name test-app, got %s", app.config.Name)
		}
	})
}

func TestApp_Router(t *testing.T) {
	app := New(nil)
	router := app.Router()
	if router == nil {
		t.Fatal("expected router to be non-nil")
	}
}

func TestApp_Container(t *testing.T) {
	app := New(nil)
	container := app.Container()
	if container == nil {
		t.Fatal("expected container to be non-nil")
	}

	// Verify app is bound to container
	if !container.Has("app") {
		t.Error("expected 'app' to be bound to container")
	}
}

func TestApp_Use(t *testing.T) {
	app := New(nil)

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	app.Use(middleware)

	if len(app.middleware) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(app.middleware))
	}
}

func TestApp_Group(t *testing.T) {
	app := New(nil)

	group := app.Group("/api")
	if group == nil {
		t.Fatal("expected group to be non-nil")
	}

	// Test nested group
	nested := group.Group("/v1")
	if nested == nil {
		t.Fatal("expected nested group to be non-nil")
	}
}

func TestRouteGroup_Methods(t *testing.T) {
	app := New(nil)
	group := app.Group("/api")

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// These should not panic
	group.GET("/get", handler)
	group.POST("/post", handler)
	group.PUT("/put", handler)
	group.DELETE("/delete", handler)
	group.PATCH("/patch", handler)
}

func TestContext(t *testing.T) {
	t.Run("NewContext", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?foo=bar", nil)
		w := httptest.NewRecorder()

		ctx := NewContext(w, req)
		if ctx == nil {
			t.Fatal("expected context to be non-nil")
		}
		if ctx.Request != req {
			t.Error("expected request to be set")
		}
		if ctx.Response != w {
			t.Error("expected response to be set")
		}
	})

	t.Run("Query", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?foo=bar&empty=", nil)
		w := httptest.NewRecorder()
		ctx := NewContext(w, req)

		if got := ctx.Query("foo"); got != "bar" {
			t.Errorf("expected 'bar', got '%s'", got)
		}
		if got := ctx.Query("missing"); got != "" {
			t.Errorf("expected empty string, got '%s'", got)
		}
	})

	t.Run("QueryDefault", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?foo=bar", nil)
		w := httptest.NewRecorder()
		ctx := NewContext(w, req)

		if got := ctx.QueryDefault("foo", "default"); got != "bar" {
			t.Errorf("expected 'bar', got '%s'", got)
		}
		if got := ctx.QueryDefault("missing", "default"); got != "default" {
			t.Errorf("expected 'default', got '%s'", got)
		}
	})

	t.Run("Header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Custom", "value")
		w := httptest.NewRecorder()
		ctx := NewContext(w, req)

		if got := ctx.Header("X-Custom"); got != "value" {
			t.Errorf("expected 'value', got '%s'", got)
		}
	})

	t.Run("SetHeader", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		ctx := NewContext(w, req)

		ctx.SetHeader("X-Response", "test")
		if got := w.Header().Get("X-Response"); got != "test" {
			t.Errorf("expected 'test', got '%s'", got)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		ctx := NewContext(w, req)

		data := map[string]string{"message": "hello"}
		err := ctx.JSON(200, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if w.Code != 200 {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/json;charset=UTF-8" {
			t.Errorf("expected JSON content type, got '%s'", ct)
		}
	})

	t.Run("String", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		ctx := NewContext(w, req)

		err := ctx.String(200, "Hello %s", "World")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if w.Code != 200 {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if body := w.Body.String(); body != "Hello World" {
			t.Errorf("expected 'Hello World', got '%s'", body)
		}
	})
}
