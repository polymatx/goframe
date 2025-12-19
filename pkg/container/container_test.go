package container

import (
	"errors"
	"sync"
	"testing"
)

func TestContainer_New(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("expected container to be non-nil")
	}
}

func TestContainer_Bind(t *testing.T) {
	c := New()

	t.Run("bind service", func(t *testing.T) {
		err := c.Bind("myService", "hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("duplicate binding returns error", func(t *testing.T) {
		c := New()
		_ = c.Bind("service", "value1")
		err := c.Bind("service", "value2")
		if err == nil {
			t.Error("expected error for duplicate binding")
		}
	})
}

func TestContainer_Resolve(t *testing.T) {
	c := New()

	t.Run("resolve bound service", func(t *testing.T) {
		_ = c.Bind("greeting", "hello")

		result, err := c.Resolve("greeting")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "hello" {
			t.Errorf("expected 'hello', got '%v'", result)
		}
	})

	t.Run("resolve non-existent service", func(t *testing.T) {
		_, err := c.Resolve("nonexistent")
		if err == nil {
			t.Error("expected error for non-existent service")
		}
	})
}

func TestContainer_BindFactory(t *testing.T) {
	c := New()

	counter := 0
	factory := func(c *Container) (interface{}, error) {
		counter++
		return counter, nil
	}

	err := c.BindFactory("counter", factory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Factory should be called each time
	result1, _ := c.Resolve("counter")
	result2, _ := c.Resolve("counter")

	if result1 == result2 {
		t.Error("expected factory to create new instances")
	}
}

func TestContainer_Singleton(t *testing.T) {
	c := New()

	counter := 0
	factory := func(c *Container) (interface{}, error) {
		counter++
		return counter, nil
	}

	err := c.Singleton("singleton", factory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Singleton should return same instance
	result1, _ := c.Resolve("singleton")
	result2, _ := c.Resolve("singleton")

	if result1 != result2 {
		t.Error("expected singleton to return same instance")
	}

	if counter != 1 {
		t.Errorf("expected factory to be called once, called %d times", counter)
	}
}

func TestContainer_Singleton_Concurrent(t *testing.T) {
	c := New()

	counter := 0
	var mu sync.Mutex

	factory := func(c *Container) (interface{}, error) {
		mu.Lock()
		counter++
		mu.Unlock()
		return counter, nil
	}

	_ = c.Singleton("concurrent", factory)

	// Resolve concurrently
	var wg sync.WaitGroup
	results := make(chan interface{}, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, _ := c.Resolve("concurrent")
			results <- result
		}()
	}

	wg.Wait()
	close(results)

	// All results should be the same
	var first interface{}
	for result := range results {
		if first == nil {
			first = result
		} else if result != first {
			t.Error("singleton returned different instances concurrently")
		}
	}

	// Factory should only be called once
	if counter != 1 {
		t.Errorf("expected factory to be called once, called %d times", counter)
	}
}

func TestContainer_Has(t *testing.T) {
	c := New()

	if c.Has("service") {
		t.Error("expected Has to return false for non-existent service")
	}

	_ = c.Bind("service", "value")

	if !c.Has("service") {
		t.Error("expected Has to return true for bound service")
	}
}

func TestContainer_Remove(t *testing.T) {
	c := New()

	_ = c.Bind("service", "value")
	c.Remove("service")

	if c.Has("service") {
		t.Error("expected service to be removed")
	}
}

func TestContainer_Clear(t *testing.T) {
	c := New()

	_ = c.Bind("service1", "value1")
	_ = c.Bind("service2", "value2")

	c.Clear()

	if c.Has("service1") || c.Has("service2") {
		t.Error("expected all services to be cleared")
	}
}

func TestContainer_MustResolve(t *testing.T) {
	c := New()
	_ = c.Bind("service", "value")

	t.Run("resolves existing service", func(t *testing.T) {
		result := c.MustResolve("service")
		if result != "value" {
			t.Errorf("expected 'value', got '%v'", result)
		}
	})

	t.Run("panics for non-existent service", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for non-existent service")
			}
		}()
		c.MustResolve("nonexistent")
	})
}

func TestContainer_FactoryError(t *testing.T) {
	c := New()

	expectedErr := errors.New("factory error")
	factory := func(c *Container) (interface{}, error) {
		return nil, expectedErr
	}

	_ = c.BindFactory("failing", factory)

	_, err := c.Resolve("failing")
	if err != expectedErr {
		t.Errorf("expected factory error, got %v", err)
	}
}
