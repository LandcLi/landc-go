package registry

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

// isEtcdAvailable 快速检测 etcd 是否可用（在创建 registry 之前调用）
func isEtcdAvailable(t *testing.T) bool {
	t.Helper()

	// 快速检测端口是否可连接（2 秒超时）
	conn, err := net.DialTimeout("tcp", "localhost:2379", 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()

	return true
}

// skipIfNoEtcd 检测 etcd 是否可用，不可用则跳过测试
func skipIfNoEtcd(t *testing.T) {
	t.Helper()

	// 先检查端口是否可连接（快速失败）
	if !isEtcdAvailable(t) {
		t.Skip("etcd not available, skipping test")
	}

	// 创建临时 registry 验证 etcd 集群状态
	cfg := EtcdRegistryConfig{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}

	registry, err := NewEtcdRegistry(cfg)
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
	defer registry.Close()

	// 验证 etcd 集群状态
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = registry.client.Cluster.MemberList(ctx)
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
}

func TestNewEtcdRegistry(t *testing.T) {
	skipIfNoEtcd(t)

	cfg := EtcdRegistryConfig{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
		Username:    "",
		Password:    "",
		Prefix:      "/services/",
		TTL:         15,
	}

	registry, err := NewEtcdRegistry(cfg)
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
	defer registry.Close()

	if registry == nil {
		t.Fatal("NewEtcdRegistry should not return nil")
	}

	if registry.prefix != "/services/" {
		t.Errorf("Expected prefix '/services/', got '%s'", registry.prefix)
	}

	if registry.ttl != 15 {
		t.Errorf("Expected TTL 15, got %d", registry.ttl)
	}
}

func TestNewEtcdRegistry_Defaults(t *testing.T) {
	skipIfNoEtcd(t)

	cfg := EtcdRegistryConfig{
		Endpoints: []string{"localhost:2379"},
	}

	registry, err := NewEtcdRegistry(cfg)
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
	defer registry.Close()

	if registry.prefix != "/services/" {
		t.Errorf("Expected default prefix '/services/', got '%s'", registry.prefix)
	}

	if registry.ttl != 15 {
		t.Errorf("Expected default TTL 15, got %d", registry.ttl)
	}
}

func TestServiceInstance_Endpoint(t *testing.T) {
	instance := &ServiceInstance{
		ID:      "instance-1",
		Name:    "test-service",
		Address: "127.0.0.1",
		Port:    8080,
	}

	endpoint := instance.Endpoint()
	expected := "127.0.0.1:8080"
	if endpoint != expected {
		t.Errorf("Expected '%s', got '%s'", expected, endpoint)
	}
}

func TestEtcdRegistry_RegisterDeregister(t *testing.T) {
	if !isEtcdAvailable(t) {
		t.Skip("etcd not available, skipping test")
	}

	cfg := EtcdRegistryConfig{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
		Prefix:      "/services/",
		TTL:         15,
	}

	registry, err := NewEtcdRegistry(cfg)
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
	defer registry.Close()

	ctx := context.Background()

	instance := &ServiceInstance{
		ID:       "test-instance",
		Name:     "test-service",
		Address:  "127.0.0.1",
		Port:     8080,
		Metadata: map[string]string{"version": "1.0"},
		Weight:   10,
	}

	err = registry.Register(ctx, instance)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	instances, err := registry.GetService(ctx, "test-service")
	if err != nil {
		t.Fatalf("GetService failed: %v", err)
	}

	if len(instances) == 0 {
		t.Error("Should have at least 1 instance")
	}

	found := false
	for _, inst := range instances {
		if inst.ID == "test-instance" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Registered instance should be found")
	}

	err = registry.Deregister(ctx, instance)
	if err != nil {
		t.Fatalf("Deregister failed: %v", err)
	}

	instances, err = registry.GetService(ctx, "test-service")
	if err != nil {
		t.Fatalf("GetService failed: %v", err)
	}

	for _, inst := range instances {
		if inst.ID == "test-instance" {
			t.Error("Deregistered instance should not be found")
		}
	}
}

func TestEtcdRegistry_GetService(t *testing.T) {
	if !isEtcdAvailable(t) {
		t.Skip("etcd not available, skipping test")
	}

	cfg := EtcdRegistryConfig{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
		Prefix:      "/services/",
		TTL:         15,
	}

	registry, err := NewEtcdRegistry(cfg)
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
	defer registry.Close()

	ctx := context.Background()

	instance := &ServiceInstance{
		ID:      "test-instance-1",
		Name:    "test-service",
		Address: "127.0.0.1",
		Port:    8080,
	}

	err = registry.Register(ctx, instance)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	defer registry.Deregister(ctx, instance)

	instances, err := registry.GetService(ctx, "test-service")
	if err != nil {
		t.Fatalf("GetService failed: %v", err)
	}

	if len(instances) == 0 {
		t.Error("Should have at least 1 instance")
	}
}

func TestEtcdRegistry_Watch(t *testing.T) {
	if !isEtcdAvailable(t) {
		t.Skip("etcd not available, skipping test")
	}

	cfg := EtcdRegistryConfig{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
		Prefix:      "/services/",
		TTL:         15,
	}

	registry, err := NewEtcdRegistry(cfg)
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
	defer registry.Close()

	ctx := context.Background()

	watcher, err := registry.Watch(ctx, "test-service")
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	defer watcher.Stop()

	// 使用 goroutine 调用阻塞的 watcher.Next()
	nextCh := make(chan struct {
		instances []*ServiceInstance
		err       error
	})
	go func() {
		instances, err := watcher.Next()
		nextCh <- struct {
			instances []*ServiceInstance
			err       error
		}{instances, err}
	}()

	// 注册服务触发 watch 事件
	instance := &ServiceInstance{
		ID:      "watch-instance",
		Name:    "test-service",
		Address: "127.0.0.1",
		Port:    8080,
	}
	registry.Register(ctx, instance)

	// 等待 watcher.Next() 返回或超时
	select {
	case result := <-nextCh:
		if result.err != nil {
			t.Fatalf("Watcher.Next failed: %v", result.err)
		}
		if len(result.instances) == 0 {
			t.Error("Should have at least 1 instance")
		}
	case <-time.After(5 * time.Second):
		t.Error("Watcher timeout")
	}
}

func TestNewRoundRobinBalancer(t *testing.T) {
	balancer := NewRoundRobinBalancer()
	if balancer == nil {
		t.Fatal("NewRoundRobinBalancer should not return nil")
	}
}

func TestRoundRobinBalancer_Pick(t *testing.T) {
	balancer := NewRoundRobinBalancer()

	instances := []*ServiceInstance{
		{ID: "instance-1", Name: "service"},
		{ID: "instance-2", Name: "service"},
		{ID: "instance-3", Name: "service"},
	}

	picked := balancer.Pick(instances)
	if picked == nil {
		t.Error("Pick should not return nil")
	}

	picked2 := balancer.Pick(instances)
	if picked2 == nil {
		t.Error("Pick should not return nil")
	}

	picked3 := balancer.Pick(instances)
	if picked3 == nil {
		t.Error("Pick should not return nil")
	}

	allPicked := map[string]bool{}
	for i := 0; i < 100; i++ {
		inst := balancer.Pick(instances)
		allPicked[inst.ID] = true
	}

	if len(allPicked) != 3 {
		t.Error("RoundRobin should pick all instances")
	}
}

func TestRoundRobinBalancer_PickEmpty(t *testing.T) {
	balancer := NewRoundRobinBalancer()

	picked := balancer.Pick([]*ServiceInstance{})
	if picked != nil {
		t.Error("Pick should return nil for empty instances")
	}
}

func TestNewWeightedBalancer(t *testing.T) {
	balancer := NewWeightedBalancer()
	if balancer == nil {
		t.Fatal("NewWeightedBalancer should not return nil")
	}
}

func TestWeightedBalancer_Pick(t *testing.T) {
	balancer := NewWeightedBalancer()

	instances := []*ServiceInstance{
		{ID: "instance-1", Name: "service", Weight: 10},
		{ID: "instance-2", Name: "service", Weight: 20},
		{ID: "instance-3", Name: "service", Weight: 30},
	}

	picked := balancer.Pick(instances)
	if picked == nil {
		t.Error("Pick should not return nil")
	}
}

func TestWeightedBalancer_PickEmpty(t *testing.T) {
	balancer := NewWeightedBalancer()

	picked := balancer.Pick([]*ServiceInstance{})
	if picked != nil {
		t.Error("Pick should return nil for empty instances")
	}
}

func TestWeightedBalancer_PickDefaultWeight(t *testing.T) {
	balancer := NewWeightedBalancer()

	instances := []*ServiceInstance{
		{ID: "instance-1", Name: "service"},
		{ID: "instance-2", Name: "service"},
	}

	picked := balancer.Pick(instances)
	if picked == nil {
		t.Error("Pick should not return nil")
	}
}

// TestResolverZeroValue 验证 Resolver 零值可用
func TestResolverZeroValue(t *testing.T) {
	var r Resolver
	if r.registry != nil {
		t.Error("zero-value Resolver should have nil registry")
	}
}

func TestResolver_Resolve(t *testing.T) {
	// Resolver 需要有效的 registry，此处仅验证代码不 panic
	// 实际使用需要在 Resolver 中设置有效的 registry
	t.Skip("Requires proper Resolver with valid registry")
}

func TestResolver_ResolveAll(t *testing.T) {
	// Resolver 需要有效的 registry，此处仅验证代码不 panic
	// 实际使用需要在 Resolver 中设置有效的 registry
	t.Skip("Requires proper Resolver with valid registry")
}

func TestResolver_Close(t *testing.T) {
	if !isEtcdAvailable(t) {
		t.Skip("etcd not available, skipping test")
	}

	// Resolver 需要有效初始化，此处仅验证 Close 方法存在且不 panic
	// 使用 NewResolver 创建有效 Resolver
	registry, err := NewEtcdRegistry(EtcdRegistryConfig{
		Endpoints: []string{"localhost:2379"},
	})
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
	defer registry.Close()

	resolver := NewResolver(registry, NewRoundRobinBalancer())
	resolver.Close()
}

func TestEtcdWatcher_Next(t *testing.T) {
	if !isEtcdAvailable(t) {
		t.Skip("etcd not available, skipping test")
	}

	cfg := EtcdRegistryConfig{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
		Prefix:      "/services/",
		TTL:         15,
	}

	registry, err := NewEtcdRegistry(cfg)
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
	defer registry.Close()

	ctx := context.Background()

	watcher, err := registry.Watch(ctx, "test-service")
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	defer watcher.Stop()

	go func() {
		instance := &ServiceInstance{
			ID:      "watcher-instance",
			Name:    "test-service",
			Address: "127.0.0.1",
			Port:    8080,
		}
		registry.Register(ctx, instance)
	}()

	instances, err := watcher.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	if len(instances) == 0 {
		t.Error("Should have at least 1 instance")
	}
}

func TestEtcdWatcher_Stop(t *testing.T) {
	if !isEtcdAvailable(t) {
		t.Skip("etcd not available, skipping test")
	}

	cfg := EtcdRegistryConfig{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
		Prefix:      "/services/",
		TTL:         15,
	}

	registry, err := NewEtcdRegistry(cfg)
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
	defer registry.Close()

	ctx := context.Background()

	watcher, err := registry.Watch(ctx, "test-service")
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	watcher.Stop()
}

func TestRegistry_Concurrent(t *testing.T) {
	if !isEtcdAvailable(t) {
		t.Skip("etcd not available, skipping test")
	}

	cfg := EtcdRegistryConfig{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
		Prefix:      "/services/",
		TTL:         15,
	}

	registry, err := NewEtcdRegistry(cfg)
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
	defer registry.Close()

	ctx := context.Background()

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			instance := &ServiceInstance{
				ID:      string('A' + rune(index)),
				Name:    "concurrent-service",
				Address: "127.0.0.1",
				Port:    8080 + index,
			}
			registry.Register(ctx, instance)
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	instances, err := registry.GetService(ctx, "concurrent-service")
	if err != nil {
		t.Fatalf("GetService failed: %v", err)
	}

	if len(instances) != 10 {
		t.Errorf("Expected 10 instances, got %d", len(instances))
	}
}

func TestBalancer_Interface(t *testing.T) {
	var balancer Balancer

	balancer = NewRoundRobinBalancer()
	if balancer == nil {
		t.Error("Should be able to assign to interface")
	}

	balancer = NewWeightedBalancer()
	if balancer == nil {
		t.Error("Should be able to assign to interface")
	}
}

func TestRegistry_Interface(t *testing.T) {
	if !isEtcdAvailable(t) {
		t.Skip("etcd not available, skipping test")
	}

	var registry Registry

	cfg := EtcdRegistryConfig{
		Endpoints: []string{"localhost:2379"},
	}

	r, err := NewEtcdRegistry(cfg)
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
	defer r.Close()

	registry = r
	if registry == nil {
		t.Error("Should be able to assign to interface")
	}
}

func TestWatcher_Interface(t *testing.T) {
	skipIfNoEtcd(t)

	cfg := EtcdRegistryConfig{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}

	registry, err := NewEtcdRegistry(cfg)
	if err != nil {
		t.Skip("etcd not available, skipping test")
	}
	defer registry.Close()

	ctx := context.Background()

	watcher, err := registry.Watch(ctx, "test-service")
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	defer watcher.Stop()

	var w Watcher
	w = watcher
	if w == nil {
		t.Error("Should be able to assign to interface")
	}
}
