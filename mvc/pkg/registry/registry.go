package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ServiceInstance 服务实例
type ServiceInstance struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Version  string            `json:"version,omitempty"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Weight   int               `json:"weight,omitempty"`
}

// Endpoint 获取完整地址
func (s *ServiceInstance) Endpoint() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

// Registry 服务注册中心接口
type Registry interface {
	Register(ctx context.Context, instance *ServiceInstance) error
	Deregister(ctx context.Context, instance *ServiceInstance) error
	GetService(ctx context.Context, name string) ([]*ServiceInstance, error)
	Watch(ctx context.Context, name string) (Watcher, error)
	Close() error
}

// Watcher 服务变化监听器
type Watcher interface {
	Next() ([]*ServiceInstance, error)
	Stop()
}

// EtcdRegistryConfig etcd 注册中心配置
type EtcdRegistryConfig struct {
	Endpoints   []string
	DialTimeout time.Duration
	Username    string
	Password    string
	Prefix      string // key 前缀（默认 /services/）
	TTL         int64  // 租约 TTL 秒数（默认 15）
}

// EtcdRegistry etcd 实现的服务注册中心
type EtcdRegistry struct {
	client *clientv3.Client
	prefix string
	ttl    int64
	leases map[string]clientv3.LeaseID
	mu     sync.Mutex
}

// NewEtcdRegistry 创建 etcd 注册中心
func NewEtcdRegistry(config EtcdRegistryConfig) (*EtcdRegistry, error) {
	if config.DialTimeout <= 0 {
		config.DialTimeout = 5 * time.Second
	}
	if config.Prefix == "" {
		config.Prefix = "/services/"
	}
	if config.TTL <= 0 {
		config.TTL = 15
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		DialTimeout: config.DialTimeout,
		Username:    config.Username,
		Password:    config.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}

	return &EtcdRegistry{
		client: client,
		prefix: config.Prefix,
		ttl:    config.TTL,
		leases: make(map[string]clientv3.LeaseID),
	}, nil
}

// Register 注册服务实例（带自动续约）
func (r *EtcdRegistry) Register(ctx context.Context, instance *ServiceInstance) error {
	lease, err := r.client.Grant(ctx, r.ttl)
	if err != nil {
		return fmt.Errorf("failed to create lease: %w", err)
	}

	data, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("failed to marshal instance: %w", err)
	}

	key := r.instanceKey(instance)
	_, err = r.client.Put(ctx, key, string(data), clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	keepAlive, err := r.client.KeepAlive(ctx, lease.ID)
	if err != nil {
		return fmt.Errorf("failed to keep alive: %w", err)
	}
	go func() {
		for range keepAlive {
		}
	}()

	r.mu.Lock()
	r.leases[key] = lease.ID
	r.mu.Unlock()

	return nil
}

// Deregister 注销服务实例
func (r *EtcdRegistry) Deregister(ctx context.Context, instance *ServiceInstance) error {
	key := r.instanceKey(instance)

	r.mu.Lock()
	leaseID, ok := r.leases[key]
	if ok {
		delete(r.leases, key)
	}
	r.mu.Unlock()

	if ok {
		r.client.Revoke(ctx, leaseID)
	}
	_, err := r.client.Delete(ctx, key)
	return err
}

// GetService 获取服务的所有实例
func (r *EtcdRegistry) GetService(ctx context.Context, name string) ([]*ServiceInstance, error) {
	prefix := r.servicePrefix(name)
	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get service %s: %w", name, err)
	}

	instances := make([]*ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var instance ServiceInstance
		if err := json.Unmarshal(kv.Value, &instance); err != nil {
			continue
		}
		instances = append(instances, &instance)
	}
	return instances, nil
}

// Watch 监听服务变化
func (r *EtcdRegistry) Watch(ctx context.Context, name string) (Watcher, error) {
	prefix := r.servicePrefix(name)
	watchCtx, cancel := context.WithCancel(ctx)
	wChan := r.client.Watch(watchCtx, prefix, clientv3.WithPrefix())

	return &etcdWatcher{
		registry: r,
		name:     name,
		ctx:      watchCtx,
		cancel:   cancel,
		wChan:    wChan,
	}, nil
}

// Close 关闭连接
func (r *EtcdRegistry) Close() error {
	r.mu.Lock()
	for _, leaseID := range r.leases {
		r.client.Revoke(context.Background(), leaseID)
	}
	r.leases = make(map[string]clientv3.LeaseID)
	r.mu.Unlock()
	return r.client.Close()
}

func (r *EtcdRegistry) instanceKey(instance *ServiceInstance) string {
	return fmt.Sprintf("%s%s/%s", r.prefix, instance.Name, instance.ID)
}

func (r *EtcdRegistry) servicePrefix(name string) string {
	return fmt.Sprintf("%s%s/", r.prefix, name)
}

// etcdWatcher
type etcdWatcher struct {
	registry *EtcdRegistry
	name     string
	ctx      context.Context
	cancel   context.CancelFunc
	wChan    clientv3.WatchChan
}

func (w *etcdWatcher) Next() ([]*ServiceInstance, error) {
	select {
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	case resp, ok := <-w.wChan:
		if !ok {
			return nil, fmt.Errorf("watcher closed")
		}
		if resp.Err() != nil {
			return nil, resp.Err()
		}
		return w.registry.GetService(w.ctx, w.name)
	}
}

func (w *etcdWatcher) Stop() {
	w.cancel()
}

// --- 负载均衡 ---

// Balancer 负载均衡器接口
type Balancer interface {
	Pick(instances []*ServiceInstance) *ServiceInstance
}

// RoundRobinBalancer 轮询负载均衡
type RoundRobinBalancer struct {
	counter uint64
	mu      sync.Mutex
}

func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{}
}

func (b *RoundRobinBalancer) Pick(instances []*ServiceInstance) *ServiceInstance {
	if len(instances) == 0 {
		return nil
	}
	b.mu.Lock()
	idx := b.counter % uint64(len(instances))
	b.counter++
	b.mu.Unlock()
	return instances[idx]
}

// WeightedBalancer 加权负载均衡
type WeightedBalancer struct{}

func NewWeightedBalancer() *WeightedBalancer {
	return &WeightedBalancer{}
}

func (b *WeightedBalancer) Pick(instances []*ServiceInstance) *ServiceInstance {
	if len(instances) == 0 {
		return nil
	}

	totalWeight := 0
	for _, inst := range instances {
		w := inst.Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += w
	}

	target := int(time.Now().UnixNano() % int64(totalWeight))
	cumulative := 0
	for _, inst := range instances {
		w := inst.Weight
		if w <= 0 {
			w = 1
		}
		cumulative += w
		if cumulative > target {
			return inst
		}
	}
	return instances[0]
}

// --- Resolver 服务解析器 ---

// Resolver 服务解析器（缓存 + 自动更新）
type Resolver struct {
	registry Registry
	balancer Balancer
	cache    map[string][]*ServiceInstance
	mu       sync.RWMutex
	watchers map[string]Watcher
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewResolver 创建服务解析器
func NewResolver(registry Registry, balancer Balancer) *Resolver {
	ctx, cancel := context.WithCancel(context.Background())
	return &Resolver{
		registry: registry,
		balancer: balancer,
		cache:    make(map[string][]*ServiceInstance),
		watchers: make(map[string]Watcher),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Resolve 解析服务地址（返回一个实例）
func (r *Resolver) Resolve(name string) (*ServiceInstance, error) {
	instances, err := r.ResolveAll(name)
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, fmt.Errorf("no instances found for service: %s", name)
	}
	return r.balancer.Pick(instances), nil
}

// ResolveAll 获取服务的所有实例
func (r *Resolver) ResolveAll(name string) ([]*ServiceInstance, error) {
	r.mu.RLock()
	instances, ok := r.cache[name]
	r.mu.RUnlock()

	if ok && len(instances) > 0 {
		return instances, nil
	}

	instances, err := r.registry.GetService(r.ctx, name)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.cache[name] = instances
	r.mu.Unlock()

	r.startWatch(name)
	return instances, nil
}

func (r *Resolver) startWatch(name string) {
	r.mu.RLock()
	_, exists := r.watchers[name]
	r.mu.RUnlock()
	if exists {
		return
	}

	watcher, err := r.registry.Watch(r.ctx, name)
	if err != nil {
		return
	}

	r.mu.Lock()
	r.watchers[name] = watcher
	r.mu.Unlock()

	go func() {
		for {
			instances, err := watcher.Next()
			if err != nil {
				return
			}
			r.mu.Lock()
			r.cache[name] = instances
			r.mu.Unlock()
		}
	}()
}

// Close 关闭解析器
func (r *Resolver) Close() {
	r.cancel()
	r.mu.Lock()
	for _, w := range r.watchers {
		w.Stop()
	}
	r.watchers = make(map[string]Watcher)
	r.mu.Unlock()
}
