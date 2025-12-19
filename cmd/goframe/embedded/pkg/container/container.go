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
	c.singletons[name] = nil
	return nil
}

// Resolve resolves a service from the container
func (c *Container) Resolve(name string) (interface{}, error) {
	c.mu.RLock()

	if _, isSingleton := c.singletons[name]; isSingleton {
		if c.singletons[name] != nil {
			service := c.singletons[name]
			c.mu.RUnlock()
			return service, nil
		}
	}

	if service, exists := c.services[name]; exists {
		c.mu.RUnlock()
		return service, nil
	}

	factory, hasFactory := c.factories[name]
	_, isSingleton := c.singletons[name]
	c.mu.RUnlock()

	if !hasFactory {
		return nil, fmt.Errorf("service '%s' not found", name)
	}

	if isSingleton {
		c.mu.Lock()
		if c.singletons[name] != nil {
			service := c.singletons[name]
			c.mu.Unlock()
			return service, nil
		}

		service, err := factory(c)
		if err != nil {
			c.mu.Unlock()
			return nil, err
		}

		c.singletons[name] = service
		c.mu.Unlock()
		return service, nil
	}

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

		tag := typeField.Tag.Get("inject")
		if tag == "" {
			continue
		}

		if !field.CanSet() {
			continue
		}

		service, err := c.Resolve(tag)
		if err != nil {
			return fmt.Errorf("failed to inject '%s': %w", tag, err)
		}

		serviceVal := reflect.ValueOf(service)
		if !serviceVal.Type().AssignableTo(field.Type()) {
			return fmt.Errorf("service '%s' type mismatch", tag)
		}

		field.Set(serviceVal)
	}

	return nil
}
