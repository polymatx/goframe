package safe

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func TestMain(m *testing.M) {
	// Recovery paths log stacks via logrus; keep test output quiet.
	// Hooks (used to assert recovery happened) still fire.
	logrus.SetOutput(io.Discard)
	m.Run()
}

// waitForLogMessage polls the test hook until an entry containing msg appears.
func waitForLogMessage(t *testing.T, hook *logrustest.Hook, msg string, timeout time.Duration) *logrus.Entry {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, entry := range hook.AllEntries() {
			if strings.Contains(entry.Message, msg) {
				return entry
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for log message %q", msg)
	return nil
}

func TestGoRoutine_RunsFunction(t *testing.T) {
	done := make(chan struct{})
	GoRoutine(context.Background(), func() {
		close(done)
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("function passed to GoRoutine was never executed")
	}
}

func TestGoRoutine_RecoversFromPanic(t *testing.T) {
	hook := logrustest.NewGlobal()
	defer hook.Reset()

	GoRoutine(context.Background(), func() {
		panic("boom in goroutine")
	})

	entry := waitForLogMessage(t, hook, "Recovered from panic in goroutine", 2*time.Second)

	if entry.Level != logrus.ErrorLevel {
		t.Errorf("recovery logged at level %v, want %v", entry.Level, logrus.ErrorLevel)
	}
	if got, ok := entry.Data["panic"]; !ok || got != "boom in goroutine" {
		t.Errorf("panic field = %v, want %q", got, "boom in goroutine")
	}
	if _, ok := entry.Data["stack"]; !ok {
		t.Error("expected stack field in recovery log entry")
	}
}

func TestContinuesGoRoutine_RepeatsUntilCancel(t *testing.T) {
	var count int32

	ctx := ContinuesGoRoutine(context.Background(), func(cancel context.CancelFunc) time.Duration {
		if atomic.AddInt32(&count, 1) >= 3 {
			cancel()
		}
		return time.Millisecond
	})

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context was not canceled")
	}

	if got := atomic.LoadInt32(&count); got < 3 {
		t.Errorf("function ran %d times, want at least 3", got)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Errorf("ctx.Err() = %v, want context.Canceled", ctx.Err())
	}
}

func TestContinuesGoRoutine_ZeroDelayStopsLoop(t *testing.T) {
	var count int32

	ctx := ContinuesGoRoutine(context.Background(), func(cancel context.CancelFunc) time.Duration {
		atomic.AddInt32(&count, 1)
		return 0
	})

	// Give the loop time to (incorrectly) run again if it did not stop.
	time.Sleep(100 * time.Millisecond)

	if got := atomic.LoadInt32(&count); got != 1 {
		t.Errorf("function ran %d times after returning 0, want exactly 1", got)
	}

	// Documents current behavior: returning 0 stops the loop but does NOT
	// cancel the returned context, so callers waiting on ctx.Done() block
	// forever. See final report; likely a bug.
	if ctx.Err() != nil {
		t.Logf("note: behavior changed, ctx now canceled on zero delay: %v", ctx.Err())
	}
}

func TestContinuesGoRoutine_PanicCancelsContext(t *testing.T) {
	hook := logrustest.NewGlobal()
	defer hook.Reset()

	ctx := ContinuesGoRoutine(context.Background(), func(cancel context.CancelFunc) time.Duration {
		panic("boom in loop")
	})

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context was not canceled after panic")
	}

	entry := waitForLogMessage(t, hook, "Recovered from panic in continuous goroutine", 2*time.Second)
	if got, ok := entry.Data["panic"]; !ok || got != "boom in loop" {
		t.Errorf("panic field = %v, want %q", got, "boom in loop")
	}
}

func TestContinuesGoRoutine_RespectsParentCancel(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())

	started := make(chan struct{})
	var once atomic.Bool
	ctx := ContinuesGoRoutine(parent, func(cancel context.CancelFunc) time.Duration {
		if once.CompareAndSwap(false, true) {
			close(started)
		}
		return time.Millisecond
	})

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("loop never started")
	}

	parentCancel()

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("child context not done after parent cancel")
	}
}

func TestTry(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		calls := 0
		err := Try(func() error {
			calls++
			return nil
		}, 0)
		if err != nil {
			t.Fatalf("Try returned error: %v", err)
		}
		if calls != 1 {
			t.Errorf("function called %d times, want 1", calls)
		}
	})

	t.Run("persistent error returned after max duration", func(t *testing.T) {
		wantErr := errors.New("persistent failure")
		calls := 0
		err := Try(func() error {
			calls++
			return wantErr
		}, 0) // zero max duration: give up after first failure
		if !errors.Is(err, wantErr) {
			t.Fatalf("Try returned %v, want %v", err, wantErr)
		}
		if calls != 1 {
			t.Errorf("function called %d times, want 1", calls)
		}
	})

	t.Run("string panic converted to PanicError", func(t *testing.T) {
		err := Try(func() error {
			panic("string panic")
		}, 0)
		var pe *PanicError
		if !errors.As(err, &pe) {
			t.Fatalf("Try returned %T (%v), want *PanicError", err, err)
		}
		if pe.Message != "string panic" {
			t.Errorf("PanicError.Message = %q, want %q", pe.Message, "string panic")
		}
	})

	t.Run("error panic returned as-is", func(t *testing.T) {
		wantErr := errors.New("panic error value")
		err := Try(func() error {
			panic(wantErr)
		}, 0)
		if !errors.Is(err, wantErr) {
			t.Errorf("Try returned %v, want %v", err, wantErr)
		}
	})

	t.Run("non-string non-error panic becomes unknown PanicError", func(t *testing.T) {
		err := Try(func() error {
			panic(42)
		}, 0)
		var pe *PanicError
		if !errors.As(err, &pe) {
			t.Fatalf("Try returned %T (%v), want *PanicError", err, err)
		}
		if pe.Message != "unknown panic" {
			t.Errorf("PanicError.Message = %q, want %q", pe.Message, "unknown panic")
		}
	})

	t.Run("retries until success", func(t *testing.T) {
		calls := 0
		err := Try(func() error {
			calls++
			if calls < 2 {
				return errors.New("transient")
			}
			return nil
		}, 10*time.Second)
		if err != nil {
			t.Fatalf("Try returned error: %v", err)
		}
		if calls != 2 {
			t.Errorf("function called %d times, want 2", calls)
		}
	})

	t.Run("recovers from panic then succeeds", func(t *testing.T) {
		calls := 0
		err := Try(func() error {
			calls++
			if calls < 2 {
				panic("transient panic")
			}
			return nil
		}, 10*time.Second)
		if err != nil {
			t.Fatalf("Try returned error: %v", err)
		}
		if calls != 2 {
			t.Errorf("function called %d times, want 2", calls)
		}
	})
}

func TestPanicError_Error(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{"simple message", "something broke", "panic: something broke"},
		{"empty message", "", "panic: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &PanicError{Message: tt.message}
			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
