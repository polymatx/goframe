package config

import (
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Initialize initializes the configuration system with the given application prefix
// Config file should be named: {prefix}_config.yaml
func Initialize(prefix string) error {
	viper.SetEnvPrefix(prefix)
	viper.AutomaticEnv()
	viper.SetConfigName(prefix + "_config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/" + prefix)

	if err := viper.ReadInConfig(); err != nil {
		logrus.Warnf("Config file not found, using environment variables only: %v", err)
	}

	viper.OnConfigChange(func(e fsnotify.Event) {
		logrus.Infof("Config file changed: %s", e.Name)
	})
	viper.WatchConfig()

	return nil
}

// SetDefault sets a default value and binds it to an environment variable
func SetDefault(key string, value interface{}) error {
	if err := viper.BindEnv(key); err != nil {
		return err
	}
	viper.SetDefault(key, value)
	return nil
}

// GetIntOrDefault returns an integer config value or a default if not set or zero
func GetIntOrDefault(key string, def int) int {
	n := viper.GetInt(key)
	if n != 0 {
		return n
	}
	return def
}

// GetStringOrDefault returns a string config value or a default if not set or empty
func GetStringOrDefault(key string, def string) string {
	s := viper.GetString(key)
	if s != "" {
		return s
	}
	return def
}

// GetBoolOrDefault returns a boolean config value or a default if not set
func GetBoolOrDefault(key string, def bool) bool {
	if viper.IsSet(key) {
		return viper.GetBool(key)
	}
	return def
}
