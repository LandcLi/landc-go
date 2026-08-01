package di

import (
	"fmt"
	"reflect"
	"sync"
)

type (
	Registry struct {
		container *Container
	}

	// lazyHolder 懒加载单例容器，持有工厂函数，首次 resolve 时执行并缓存实例
	lazyHolder struct {
		once     sync.Once
		instance interface{}
		factory  func() interface{}
	}
)

// resolve 执行工厂函数并返回单例实例（线程安全）
func (h *lazyHolder) resolve() interface{} {
	h.once.Do(func() {
		h.instance = h.factory()
	})
	return h.instance
}

func NewRegistry(container *Container) *Registry {
	if container == nil {
		container = GetGlobalContainer()
	}
	return &Registry{container: container}
}

func (r *Registry) Register(name string, service interface{}, overwrite bool) error {
	return r.container.Register(name, service, overwrite)
}

func (r *Registry) Get(name string) (interface{}, error) {
	return r.container.Get(name)
}

func (r *Registry) MustGet(name string) interface{} {
	return r.container.MustGet(name)
}

func (r *Registry) Has(name string) bool {
	return r.container.Has(name)
}

// RegisterInterface 注册接口的实现（泛型版本）
func RegisterInterface[T any](name string, impl T, overwrite bool) error {
	return Register(name, impl, overwrite)
}

// GetInterface 获取接口的实现（泛型版本，带类型检查）
func GetInterface[T any](name string) (T, error) {
	var zero T
	service, err := Get(name)
	if err != nil {
		return zero, err
	}

	impl, ok := service.(T)
	if !ok {
		return zero, fmt.Errorf("service %s does not implement the expected type", name)
	}

	return impl, nil
}

// MustGetInterface 获取接口的实现（泛型版本，失败时 panic）
func MustGetInterface[T any](name string) T {
	impl, err := GetInterface[T](name)
	if err != nil {
		panic(err)
	}
	return impl
}

// RegisterSingleton 注册单例（立即实例化）
func RegisterSingleton[T any](name string, factory func() T, overwrite bool) error {
	container := GetGlobalContainer()

	if !overwrite && container.Has(name) {
		return fmt.Errorf("service %s already registered and overwrite is false", name)
	}

	instance := factory()
	return Register(name, instance, overwrite)
}

// RegisterLazySingleton 注册懒加载单例（首次获取时实例化，线程安全）
func RegisterLazySingleton[T any](name string, factory func() T, overwrite bool) error {
	container := GetGlobalContainer()

	if !overwrite && container.Has(name) {
		return fmt.Errorf("service %s already registered and overwrite is false", name)
	}

	holder := &lazyHolder{
		factory: func() interface{} {
			return factory()
		},
	}

	return container.Register(name, holder, overwrite)
}

// CallMethod 通过反射调用实例的方法
func CallMethod(instance interface{}, methodName string, args ...interface{}) ([]interface{}, error) {
	val := reflect.ValueOf(instance)
	method := val.MethodByName(methodName)

	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	methodType := method.Type()

	if len(args) != methodType.NumIn() {
		return nil, fmt.Errorf("method %s expects %d arguments, got %d", methodName, methodType.NumIn(), len(args))
	}

	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	results := method.Call(in)

	out := make([]interface{}, len(results))
	for i, result := range results {
		out[i] = result.Interface()
	}

	return out, nil
}
