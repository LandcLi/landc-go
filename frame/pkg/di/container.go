package di

import (
	"fmt"
	"sync"
)

type (
	Container struct {
		mu       sync.RWMutex
		services map[string]interface{}
	}

	Option func(*Container)

	// lazySingleton 懒加载单例接口，Container.Get 时自动解包执行工厂函数
	lazySingleton interface {
		resolve() interface{}
	}
)

var globalContainer *Container

func init() {
	globalContainer = NewContainer()
}

func NewContainer(opts ...Option) *Container {
	c := &Container{
		services: make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func GetGlobalContainer() *Container {
	return globalContainer
}

func Register(name string, service interface{}, overwrite bool) error {
	return globalContainer.Register(name, service, overwrite)
}

func Get(name string) (interface{}, error) {
	return globalContainer.Get(name)
}

func MustGet(name string) interface{} {
	return globalContainer.MustGet(name)
}

func Has(name string) bool {
	return globalContainer.Has(name)
}

func (c *Container) Register(name string, service interface{}, overwrite bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !overwrite {
		if _, exists := c.services[name]; exists {
			return fmt.Errorf("service %s already registered and overwrite is false", name)
		}
	}

	c.services[name] = service
	return nil
}

func (c *Container) Get(name string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	service, exists := c.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	// 懒加载单例：首次获取时执行工厂函数，返回真实实例
	if holder, ok := service.(lazySingleton); ok {
		return holder.resolve(), nil
	}

	return service, nil
}

func (c *Container) MustGet(name string) interface{} {
	service, err := c.Get(name)
	if err != nil {
		panic(err)
	}
	return service
}

func (c *Container) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.services[name]
	return exists
}
