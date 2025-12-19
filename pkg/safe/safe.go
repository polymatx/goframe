package safe

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/polymatx/goframe/pkg/xlog"
	"github.com/sirupsen/logrus"
)

// GoRoutine runs a function in a goroutine with automatic panic recovery
func GoRoutine(ctx context.Context, fn func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				xlog.GetWithField(ctx, "panic", err).
					WithField("stack", string(debug.Stack())).
					Error("Recovered from panic in goroutine")
			}
		}()
		fn()
	}()
}

// ContinuesGoRoutine runs a function repeatedly with a delay, handling panics
// The function returns the duration to wait before the next iteration
// Use the cancel function to stop the loop
func ContinuesGoRoutine(ctx context.Context, fn func(context.CancelFunc) time.Duration) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				xlog.GetWithField(ctx, "panic", err).
					WithField("stack", string(debug.Stack())).
					Error("Recovered from panic in continuous goroutine")
				cancel()
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				delay := fn(cancel)
				if delay == 0 {
					return
				}
				time.Sleep(delay)
			}
		}
	}()

	return ctx
}

// Try attempts to execute a function with retries on panic/error
// maxDuration is the maximum time to keep retrying
func Try(fn func() error, maxDuration time.Duration) error {
	start := time.Now()
	attempt := 0

	for {
		attempt++

		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = recoverToError(r)
					logrus.WithFields(logrus.Fields{
						"attempt": attempt,
						"panic":   r,
						"stack":   string(debug.Stack()),
					}).Error("Panic in Try function")
				}
			}()
			err = fn()
		}()

		if err == nil {
			return nil
		}

		if time.Since(start) >= maxDuration {
			logrus.WithField("attempts", attempt).Error("Try function exceeded max duration")
			return err
		}

		// Exponential backoff with max 30 seconds
		backoff := time.Duration(attempt) * time.Second
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}

		logrus.WithFields(logrus.Fields{
			"attempt": attempt,
			"backoff": backoff,
			"error":   err,
		}).Warn("Retrying after error")

		time.Sleep(backoff)
	}
}

func recoverToError(r interface{}) error {
	switch v := r.(type) {
	case error:
		return v
	case string:
		return &PanicError{Message: v}
	default:
		return &PanicError{Message: "unknown panic"}
	}
}

// PanicError represents an error from a recovered panic
type PanicError struct {
	Message string
}

func (e *PanicError) Error() string {
	return "panic: " + e.Message
}
