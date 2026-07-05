package random

import (
	"regexp"
	"testing"
	"time"
)

// readID reads one value from the ID channel with a timeout so a broken
// generator fails the test instead of hanging it.
func readID(t *testing.T) string {
	t.Helper()
	select {
	case id := <-ID:
		return id
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ID from generator")
		return ""
	}
}

func TestID_Format(t *testing.T) {
	hexPattern := regexp.MustCompile(`^[0-9a-f]{64}$`)

	for i := 0; i < 10; i++ {
		id := readID(t)
		if !hexPattern.MatchString(id) {
			t.Fatalf("ID %q is not a 64-char lowercase hex string (sha256)", id)
		}
	}
}

func TestID_Uniqueness(t *testing.T) {
	const n = 1000
	seen := make(map[string]bool, n)

	for i := 0; i < n; i++ {
		id := readID(t)
		if seen[id] {
			t.Fatalf("duplicate ID generated after %d reads: %q", i, id)
		}
		seen[id] = true
	}
}

func TestID_ConcurrentReaders(t *testing.T) {
	const readers = 8
	const perReader = 50

	results := make(chan string, readers*perReader)
	for i := 0; i < readers; i++ {
		go func() {
			for j := 0; j < perReader; j++ {
				results <- <-ID
			}
		}()
	}

	seen := make(map[string]bool, readers*perReader)
	for i := 0; i < readers*perReader; i++ {
		select {
		case id := <-results:
			if id == "" {
				t.Fatal("received empty ID")
			}
			if seen[id] {
				t.Fatalf("duplicate ID across concurrent readers: %q", id)
			}
			seen[id] = true
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for concurrent IDs")
		}
	}
}
