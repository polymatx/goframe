package container

import (
	"fmt"
	"reflect"
	"sync"
)

// Container is a dependency injection container
type Container struct {
	services   map[string]interface{}
	factories  map[string]func(*Container) (interface{}, error)
	singletons map[string]interface{}
	mu         sync.RWMutex
}

// New creates a new Container
func New() *Container {
	return &Container{
		services:   make(map[string]interface{}),
		factories:  make(map[string]func(*Container) (interface{}, error)),
		singletons: make(map[string]interface{}),
	}
}

// Bind binds a service to the container
func (c *Container) Bind(name string, service interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.services[name]; exists {
		return fmt.Errorf("service '%s' already bound", name)
	}

	c.services[name] = service
	return nil
}

// BindFactory binds a factory function
func (c *Container) BindFactory(name string, factory func(*Container) (interface{}, error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.factories[name]; exists {
		return fmt.Errorf("factory '%s' already bound", name)
	}

	c.factories[name] = factory
	return nil
}

// Singleton binds a singleton service
func (c *Container) Singleton(name string, factory func(*Container) (interface{}, error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.singletons[name]; exists {
		return fmt.Errorf("singleton '%s' already bound", name)
	}

	c.factories[name] = factory
	c.singletons[name] = nil // Mark as singleton
	return nil
}

// Resolve resolves a service from the container
func (c *Container) Resolve(name string) (interface{}, error) {
	c.mu.RLock()

	// Check if it's a singleton and already instantiated
	if _, isSingleton := c.singletons[name]; isSingleton {
		if c.singletons[name] != nil {
			service := c.singletons[name]
			c.mu.RUnlock()
			return service, nil
		}
	}

	// Check direct binding
	if service, exists := c.services[name]; exists {
		c.mu.RUnlock()
		return service, nil
	}

	// Check factory
	factory, hasFactory := c.factories[name]
	_, isSingleton := c.singletons[name]
	c.mu.RUnlock()

	if !hasFactory {
		return nil, fmt.Errorf("service '%s' not found", name)
	}

	// For singletons, use double-checked locking to prevent race conditions
	if isSingleton {
		c.mu.Lock()
		// Double-check if another goroutine created it while we waited for the lock
		if c.singletons[name] != nil {
			service := c.singletons[name]
			c.mu.Unlock()
			return service, nil
		}

		// Create instance from factory while holding the lock
		service, err := factory(c)
		if err != nil {
			c.mu.Unlock()
			return nil, err
		}

		c.singletons[name] = service
		c.mu.Unlock()
		return service, nil
	}

	// For non-singletons, just create a new instance
	return factory(c)
}

// MustResolve resolves or panics
func (c *Container) MustResolve(name string) interface{} {
	service, err := c.Resolve(name)
	if err != nil {
		panic(err)
	}
	return service
}

// Has checks if service exists
func (c *Container) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, hasService := c.services[name]
	_, hasFactory := c.factories[name]
	return hasService || hasFactory
}

// Remove removes a service
func (c *Container) Remove(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.services, name)
	delete(c.factories, name)
	delete(c.singletons, name)
}

// Clear clears all services
func (c *Container) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services = make(map[string]interface{})
	c.factories = make(map[string]func(*Container) (interface{}, error))
	c.singletons = make(map[string]interface{})
}

// Inject injects dependencies into struct fields
func (c *Container) Inject(target interface{}) error {
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	typ := elem.Type()
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		typeField := typ.Field(i)

		// Check for inject tag
		tag := typeField.Tag.Get("inject")
		if tag == "" {
			continue
		}

		if !field.CanSet() {
			continue
		}

		// Resolve service
		service, err := c.Resolve(tag)
		if err != nil {
			return fmt.Errorf("failed to inject '%s': %w", tag, err)
		}

		// Set field
		serviceVal := reflect.ValueOf(service)
		if !serviceVal.Type().AssignableTo(field.Type()) {
			return fmt.Errorf("service '%s' type mismatch", tag)
		}

		field.Set(serviceVal)
	}

	return nil
}

// Call invokes a function with dependency injection
func (c *Container) Call(fn interface{}) ([]interface{}, error) {
	fnVal := reflect.ValueOf(fn)
	if fnVal.Kind() != reflect.Func {
		return nil, fmt.Errorf("argument must be a function")
	}

	fnType := fnVal.Type()
	args := make([]reflect.Value, fnType.NumIn())

	// Resolve parameters
	for i := 0; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)
		paramName := paramType.String()

		service, err := c.Resolve(paramName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve parameter %d (%s): %w", i, paramName, err)
		}

		args[i] = reflect.ValueOf(service)
	}

	// Call function
	results := fnVal.Call(args)

	// Convert results
	out := make([]interface{}, len(results))
	for i, result := range results {
		out[i] = result.Interface()
	}

	return out, nil
}

// Global container instance
var global = New()

// Bind binds to global container
func Bind(name string, service interface{}) error {
	return global.Bind(name, service)
}

// BindFactory binds factory to global container
func BindFactory(name string, factory func(*Container) (interface{}, error)) error {
	return global.BindFactory(name, factory)
}

// Singleton binds singleton to global container
func Singleton(name string, factory func(*Container) (interface{}, error)) error {
	return global.Singleton(name, factory)
}

// Resolve resolves from global container
func Resolve(name string) (interface{}, error) {
	return global.Resolve(name)
}

// MustResolve resolves from global container or panics
func MustResolve(name string) interface{} {
	return global.MustResolve(name)
}

// Has checks global container
func Has(name string) bool {
	return global.Has(name)
}

// Inject injects into target using global container
func Inject(target interface{}) error {
	return global.Inject(target)
}

// Call calls function with global container
func Call(fn interface{}) ([]interface{}, error) {
	return global.Call(fn)
}
