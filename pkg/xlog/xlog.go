package xlog

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type contextKey int

const ctxKey contextKey = iota

var (
	specialLogger *logrus.Logger
	logLocation   *time.Location = time.UTC
)

type concurrentFields struct {
	fields logrus.Fields
	lock   sync.RWMutex
}

// GetSpecialLogger returns the special logger instance if configured
func GetSpecialLogger() *logrus.Logger {
	return specialLogger
}

// SetLogLocation sets the timezone for log timestamps
func SetLogLocation(location *time.Location) {
	logLocation = location
}

// CustomFormatter wraps logrus formatter with custom timezone
type CustomFormatter struct {
	logrus.Formatter
}

// Format implements logrus.Formatter interface with custom timezone
func (f CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Time = entry.Time.In(logLocation)
	return f.Formatter.Format(entry)
}

func watch(appName string) {
	logPath := viper.GetString("log_path")
	if logPath == "" {
		logPath = "./logs"
	}

	lastFileName := getLogFileName(appName, logPath, time.Now())

	for {
		time.Sleep(3 * time.Second)
		t := time.Now()
		fileName := getLogFileName(appName, logPath, t)

		if fileName != lastFileName {
			f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
			if err == nil {
				logrus.SetOutput(f)
				lastFileName = fileName
			}
		}
	}
}

func getLogFileName(appName, logPath string, t time.Time) string {
	return fmt.Sprintf("%s/%s/%04d-%02d-%02d.log", logPath, appName, t.Year(), t.Month(), t.Day())
}

// Initialize sets up the logging system
func Initialize(appName string) error {
	logrus.SetFormatter(CustomFormatter{
		Formatter: &logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		},
	})

	if viper.GetBool("develop_mode") {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	if viper.GetBool("log_to_file") {
		if err := setupFileLogging(appName); err != nil {
			return fmt.Errorf("failed to setup file logging: %w", err)
		}
	}

	if viper.GetBool("special_log_to_file") {
		if err := setupSpecialLogger(appName); err != nil {
			return fmt.Errorf("failed to setup special logger: %w", err)
		}
	}

	logrus.Info("Logging system initialized")
	return nil
}

func setupFileLogging(appName string) error {
	logPath := viper.GetString("log_path")
	if logPath == "" {
		logPath = "./logs"
	}

	t := time.Now()
	dirPath := fmt.Sprintf("%s/%s", logPath, appName)

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	fileName := getLogFileName(appName, logPath, t)
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	logrus.SetOutput(f)
	go watch(appName)

	return nil
}

func setupSpecialLogger(appName string) error {
	specialLogger = logrus.New()
	specialLogger.SetLevel(logrus.DebugLevel)
	specialLogger.SetFormatter(CustomFormatter{
		Formatter: &logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		},
	})

	logPath := viper.GetString("log_path")
	if logPath == "" {
		logPath = "./logs"
	}

	t := time.Now()
	dirPath := fmt.Sprintf("%s/%s", logPath, appName)

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	fileName := getLogFileName(appName, logPath, t)
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	specialLogger.SetOutput(f)
	return nil
}

// Get returns a log entry with fields from context
func Get(ctx context.Context) *logrus.Entry {
	fields, ok := ctx.Value(ctxKey).(*concurrentFields)
	entry := logrus.NewEntry(logrus.StandardLogger())
	if ok {
		fields.lock.RLock()
		defer fields.lock.RUnlock()
		return entry.WithFields(fields.fields)
	}
	return entry
}

// GetWithError is a shorthand for Get(ctx).WithError(err)
func GetWithError(ctx context.Context, err error) *logrus.Entry {
	return Get(ctx).WithError(err)
}

// GetWithField is a shorthand for Get(ctx).WithField()
func GetWithField(ctx context.Context, key string, val interface{}) *logrus.Entry {
	return Get(ctx).WithField(key, val)
}

// GetWithFields is a shorthand for Get(ctx).WithFields()
func GetWithFields(ctx context.Context, fields logrus.Fields) *logrus.Entry {
	return Get(ctx).WithFields(fields)
}

// SetField adds a field to the context for logging
func SetField(ctx context.Context, key string, val interface{}) context.Context {
	fields, ok := ctx.Value(ctxKey).(*concurrentFields)
	if !ok {
		fields = &concurrentFields{
			fields: make(logrus.Fields),
			lock:   sync.RWMutex{},
		}
	}
	fields.lock.Lock()
	defer fields.lock.Unlock()
	fields.fields[key] = val

	return context.WithValue(ctx, ctxKey, fields)
}

// SetFields adds multiple fields to the context for logging
func SetFields(ctx context.Context, fl logrus.Fields) context.Context {
	fields, ok := ctx.Value(ctxKey).(*concurrentFields)
	if !ok {
		fields = &concurrentFields{
			fields: make(logrus.Fields),
			lock:   sync.RWMutex{},
		}
	}
	fields.lock.Lock()
	defer fields.lock.Unlock()
	for key, val := range fl {
		fields.fields[key] = val
	}
	return context.WithValue(ctx, ctxKey, fields)
}
