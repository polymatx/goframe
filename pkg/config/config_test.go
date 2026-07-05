package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// resetViper isolates each test from the package-global viper state.
func resetViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
}

func writeConfigFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
}

func TestInitialize(t *testing.T) {
	t.Run("reads yaml config from working directory", func(t *testing.T) {
		resetViper(t)
		dir := t.TempDir()
		writeConfigFile(t, dir, "testapp_config.yaml",
			"app_name: goframe-test\nport: 9090\ndebug: true\n")
		t.Chdir(dir)

		if err := Initialize("testapp"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := viper.GetString("app_name"); got != "goframe-test" {
			t.Errorf("expected 'goframe-test', got %q", got)
		}
		if got := viper.GetInt("port"); got != 9090 {
			t.Errorf("expected 9090, got %d", got)
		}
		if got := viper.GetBool("debug"); !got {
			t.Error("expected debug to be true")
		}
	})

	t.Run("reads yaml config from config subdirectory", func(t *testing.T) {
		resetViper(t)
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}
		writeConfigFile(t, filepath.Join(dir, "config"), "myapp_config.yaml",
			"listen_addr: :8081\n")
		t.Chdir(dir)

		if err := Initialize("myapp"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := viper.GetString("listen_addr"); got != ":8081" {
			t.Errorf("expected ':8081', got %q", got)
		}
	})

	t.Run("missing config file is not an error", func(t *testing.T) {
		resetViper(t)
		t.Chdir(t.TempDir())

		if err := Initialize("nosuchapp"); err != nil {
			t.Fatalf("expected nil error when config file is missing, got %v", err)
		}
	})

	t.Run("environment variables override config file", func(t *testing.T) {
		resetViper(t)
		dir := t.TempDir()
		writeConfigFile(t, dir, "envapp_config.yaml",
			"app_name: from-file\nport: 1000\n")
		t.Chdir(dir)
		t.Setenv("ENVAPP_APP_NAME", "from-env")
		t.Setenv("ENVAPP_PORT", "2000")

		if err := Initialize("envapp"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := viper.GetString("app_name"); got != "from-env" {
			t.Errorf("expected env override 'from-env', got %q", got)
		}
		if got := viper.GetInt("port"); got != 2000 {
			t.Errorf("expected env override 2000, got %d", got)
		}
	})
}

func TestSetDefault(t *testing.T) {
	t.Run("default is returned when nothing else is set", func(t *testing.T) {
		resetViper(t)
		if err := SetDefault("greeting", "hello"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := viper.GetString("greeting"); got != "hello" {
			t.Errorf("expected 'hello', got %q", got)
		}
	})

	t.Run("environment variable overrides default", func(t *testing.T) {
		resetViper(t)
		t.Setenv("DB_HOST", "env-host")
		if err := SetDefault("db_host", "localhost"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := viper.GetString("db_host"); got != "env-host" {
			t.Errorf("expected 'env-host', got %q", got)
		}
	})

	t.Run("respects env prefix set by Initialize", func(t *testing.T) {
		resetViper(t)
		t.Chdir(t.TempDir())
		if err := Initialize("acme"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := SetDefault("timeout", 30); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := GetIntOrDefault("timeout", 1); got != 30 {
			t.Errorf("expected default 30, got %d", got)
		}

		t.Setenv("ACME_TIMEOUT", "45")
		if got := GetIntOrDefault("timeout", 1); got != 45 {
			t.Errorf("expected env override 45, got %d", got)
		}
	})
}

func TestGetIntOrDefault(t *testing.T) {
	resetViper(t)
	viper.Set("set_port", 8080)
	viper.Set("zero_port", 0)

	tests := []struct {
		name string
		key  string
		def  int
		want int
	}{
		{"returns value when set", "set_port", 1000, 8080},
		{"returns default when unset", "missing_port", 1000, 1000},
		{"returns default when value is zero", "zero_port", 1000, 1000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetIntOrDefault(tt.key, tt.def); got != tt.want {
				t.Errorf("expected %d, got %d", tt.want, got)
			}
		})
	}

	t.Run("env var override", func(t *testing.T) {
		resetViper(t)
		t.Chdir(t.TempDir())
		if err := Initialize("intapp"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Setenv("INTAPP_WORKERS", "16")
		if got := GetIntOrDefault("workers", 4); got != 16 {
			t.Errorf("expected 16 from env, got %d", got)
		}
	})
}

func TestGetStringOrDefault(t *testing.T) {
	resetViper(t)
	viper.Set("set_name", "service")
	viper.Set("empty_name", "")

	tests := []struct {
		name string
		key  string
		def  string
		want string
	}{
		{"returns value when set", "set_name", "fallback", "service"},
		{"returns default when unset", "missing_name", "fallback", "fallback"},
		{"returns default when value is empty", "empty_name", "fallback", "fallback"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetStringOrDefault(tt.key, tt.def); got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}

	t.Run("env var override", func(t *testing.T) {
		resetViper(t)
		t.Chdir(t.TempDir())
		if err := Initialize("strapp"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Setenv("STRAPP_REGION", "eu-west-1")
		if got := GetStringOrDefault("region", "us-east-1"); got != "eu-west-1" {
			t.Errorf("expected 'eu-west-1' from env, got %q", got)
		}
	})
}

func TestGetBoolOrDefault(t *testing.T) {
	resetViper(t)
	viper.Set("enabled_flag", true)
	viper.Set("disabled_flag", false)

	tests := []struct {
		name string
		key  string
		def  bool
		want bool
	}{
		{"returns true when set true", "enabled_flag", false, true},
		// Unlike the int/string variants, an explicitly-set false wins over the default.
		{"returns false when set false", "disabled_flag", true, false},
		{"returns default when unset", "missing_flag", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBoolOrDefault(tt.key, tt.def); got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}

	t.Run("env var override", func(t *testing.T) {
		resetViper(t)
		t.Chdir(t.TempDir())
		if err := Initialize("boolapp"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Setenv("BOOLAPP_FEATURE", "true")
		if got := GetBoolOrDefault("feature", false); !got {
			t.Error("expected true from env")
		}
	})
}
