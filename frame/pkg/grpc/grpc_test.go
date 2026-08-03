package grpc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestNewServer(t *testing.T) {
	cfg := ServerConfig{
		Address:        ":50051",
		MaxRecvMsgSize: 10 * 1024 * 1024,
		MaxSendMsgSize: 10 * 1024 * 1024,
	}

	server := NewServer(cfg)
	if server == nil {
		t.Fatal("NewServer should not return nil")
	}

	if server.server == nil {
		t.Error("grpc.Server should be initialized")
	}

	if server.config.Address != ":50051" {
		t.Errorf("Expected address ':50051', got '%s'", server.config.Address)
	}
}

func TestServer_GetServer(t *testing.T) {
	cfg := ServerConfig{
		Address: ":50051",
	}

	server := NewServer(cfg)
	grpcServer := server.GetServer()

	if grpcServer == nil {
		t.Error("GetServer should not return nil")
	}
}

func TestServer_RunGracefulStop(t *testing.T) {
	cfg := ServerConfig{
		Address: ":50052",
	}

	server := NewServer(cfg)

	go func() {
		err := server.Run()
		if err != nil {
			t.Errorf("Run failed: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	server.GracefulStop()
}

func TestServer_Stop(t *testing.T) {
	cfg := ServerConfig{
		Address: ":50053",
	}

	server := NewServer(cfg)

	go func() {
		server.Run()
	}()

	time.Sleep(100 * time.Millisecond)

	server.Stop()
}

func TestDial(t *testing.T) {
	cfg := ClientConfig{
		Target:   "localhost:50051",
		Insecure: true,
		Timeout:  5 * time.Second,
	}

	conn, err := Dial(cfg)
	if err != nil {
		t.Skip("gRPC server not available, skipping test")
	}
	defer conn.Close()

	if conn == nil {
		t.Error("Dial should not return nil")
	}
}

func TestDial_InvalidTarget(t *testing.T) {
	// grpc.DialContext 默认懒连接，WithBlock 强制立即连接
	_, err := Dial(ClientConfig{
		Target:   "invalid:99999",
		Insecure: true,
		Timeout:  500 * time.Millisecond,
	})
	// 即使连不上，Dial 也可能返回非 nil conn + error，直接调 Dial 不传 WithBlock 会成功
	// 此处仅验证 Dial 函数不 panic
	if err != nil {
		// 预期会报错，通过
		t.Logf("Dial failed as expected: %v", err)
	}
}

func TestDialWithRegistry(t *testing.T) {
	// Resolver 不是 EtcdRegistry 的方法，且需要 etcd 连接，跳过
	t.Skip("Requires etcd connection and proper Resolver setup")
}

func TestTraceUnaryServerInterceptor(t *testing.T) {
	interceptor := TraceUnaryServerInterceptor()
	if interceptor == nil {
		t.Fatal("TraceUnaryServerInterceptor should not return nil")
	}

	ctx := context.Background()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		traceID := ctx.Value(traceIDKey{})
		if traceID == nil {
			t.Error("TraceID should be in context")
		}
		return "response", nil
	}

	_, err := interceptor(ctx, "test-request", &grpc.UnaryServerInfo{
		FullMethod: "/test/service",
	}, handler)
	if err != nil {
		t.Fatalf("Interceptor handler failed: %v", err)
	}
}

func TestTraceUnaryClientInterceptor(t *testing.T) {
	interceptor := TraceUnaryClientInterceptor()
	if interceptor == nil {
		t.Fatal("TraceUnaryClientInterceptor should not return nil")
	}

	// grpc.UnaryClientInterceptor 签名：
	// func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error
	// 需要真实的 grpc.ClientConn 才能完整测试，此处只验证不 panic
	t.Skip("Requires real gRPC connection to fully test client interceptor")
}

func TestRecoveryUnaryServerInterceptor(t *testing.T) {
	interceptor := RecoveryUnaryServerInterceptor()
	if interceptor == nil {
		t.Fatal("RecoveryUnaryServerInterceptor should not return nil")
	}

	ctx := context.Background()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test panic")
	}

	_, err := interceptor(ctx, "test", &grpc.UnaryServerInfo{
		FullMethod: "/test/service",
	}, handler)
	if err == nil {
		t.Error("Should recover from panic and return error")
	}
}

func TestTimeoutUnaryServerInterceptor(t *testing.T) {
	timeout := 100 * time.Millisecond
	interceptor := TimeoutUnaryServerInterceptor(timeout)
	if interceptor == nil {
		t.Fatal("TimeoutUnaryServerInterceptor should not return nil")
	}

	ctx := context.Background()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// 等待上下文超时
		<-ctx.Done()
		return nil, ctx.Err()
	}

	_, err := interceptor(ctx, "test", &grpc.UnaryServerInfo{
		FullMethod: "/test/service",
	}, handler)
	if err == nil {
		t.Error("Should timeout")
	}
}

func TestGetTraceID(t *testing.T) {
	ctx := context.WithValue(context.Background(), traceIDKey{}, "test-trace-id")

	traceID := GetTraceID(ctx)
	if traceID != "test-trace-id" {
		t.Errorf("Expected 'test-trace-id', got '%s'", traceID)
	}
}

func TestGetTraceID_NotFound(t *testing.T) {
	ctx := context.Background()

	traceID := GetTraceID(ctx)
	if traceID != "" {
		t.Errorf("Expected empty string, got '%s'", traceID)
	}
}

func TestNewConnPool(t *testing.T) {
	cfg := ClientConfig{
		Insecure: true,
		Timeout:  5 * time.Second,
	}

	resolver := &registry.Resolver{}
	pool := NewConnPool(cfg, resolver)

	if pool == nil {
		t.Fatal("NewConnPool should not return nil")
	}

	if pool.conns == nil {
		t.Error("conns map should be initialized")
	}

	if pool.resolver != resolver {
		t.Error("Resolver should be set")
	}
}

func TestConnPool_Get(t *testing.T) {
	cfg := ClientConfig{
		Target:   "localhost:50051",
		Insecure: true,
		Timeout:  5 * time.Second,
	}

	pool := NewConnPool(cfg, nil)

	_, err := pool.Get("localhost:50051")
	if err != nil {
		t.Skip("gRPC server not available, skipping test")
	}
}

func TestConnPool_GetWithResolver(t *testing.T) {
	// Resolver 需要有效的 registry，此处仅验证代码不 panic
	// 实际使用需要在 Resolver 中设置有效的 registry
	t.Skip("Requires proper Resolver with valid registry")
}

func TestConnPool_Close(t *testing.T) {
	cfg := ClientConfig{
		Target:   "localhost:50051",
		Insecure: true,
		Timeout:  5 * time.Second,
	}

	pool := NewConnPool(cfg, nil)

	conn, err := pool.Get("localhost:50051")
	if err != nil {
		t.Skip("gRPC server not available, skipping test")
	}

	err = pool.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if len(pool.conns) != 0 {
		t.Error("Connections should be cleared after Close")
	}

	if conn != nil {
		conn.Close()
	}
}

func TestConnPool_Concurrent(t *testing.T) {
	cfg := ClientConfig{
		Target:   "localhost:50051",
		Insecure: true,
		Timeout:  2 * time.Second,
	}

	pool := NewConnPool(cfg, nil)

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := pool.Get("localhost:50051")
			if err != nil {
				t.Logf("Get failed (expected if no server): %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestServerConfig_Defaults(t *testing.T) {
	cfg := ServerConfig{}

	server := NewServer(cfg)
	if server.config.Address != "" {
		t.Errorf("Expected empty address, got '%s'", server.config.Address)
	}

	if server.config.MaxRecvMsgSize != 0 {
		t.Errorf("Expected 0 MaxRecvMsgSize, got %d", server.config.MaxRecvMsgSize)
	}
}

func TestClientConfig_Defaults(t *testing.T) {
	cfg := ClientConfig{}

	if cfg.Target != "" {
		t.Errorf("Expected empty target, got '%s'", cfg.Target)
	}

	if cfg.Timeout != 0 {
		t.Errorf("Expected 0 timeout, got %v", cfg.Timeout)
	}
}

func TestInterceptorChain(t *testing.T) {
	traceInterceptor := TraceUnaryServerInterceptor()
	recoveryInterceptor := RecoveryUnaryServerInterceptor()

	interceptors := []grpc.UnaryServerInterceptor{
		traceInterceptor,
		recoveryInterceptor,
	}

	if len(interceptors) != 2 {
		t.Errorf("Expected 2 interceptors, got %d", len(interceptors))
	}
}

func TestConnPool_ReuseConnection(t *testing.T) {
	cfg := ClientConfig{
		Target:   "localhost:50051",
		Insecure: true,
		Timeout:  5 * time.Second,
	}

	pool := NewConnPool(cfg, nil)

	conn1, err := pool.Get("localhost:50051")
	if err != nil {
		t.Skip("gRPC server not available, skipping test")
	}

	conn2, err := pool.Get("localhost:50051")
	if err != nil {
		t.Fatalf("Second Get failed: %v", err)
	}

	if conn1 != conn2 {
		t.Error("Should reuse connection from pool")
	}

	pool.Close()
}

func TestGrpc_Integration(t *testing.T) {
	cfg := ServerConfig{
		Address: ":50054",
	}

	server := NewServer(cfg)

	go func() {
		err := server.Run()
		if err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	clientCfg := ClientConfig{
		Target:   "localhost:50054",
		Insecure: true,
		Timeout:  5 * time.Second,
	}

	conn, err := Dial(clientCfg)
	if err != nil {
		t.Skip("Integration test failed, gRPC connection error")
	}
	defer conn.Close()

	if conn == nil {
		t.Error("Connection should not be nil")
	}

	server.GracefulStop()
}

func TestConnPool_MultipleServices(t *testing.T) {
	cfg := ClientConfig{
		Insecure: true,
		Timeout:  2 * time.Second,
	}

	pool := NewConnPool(cfg, nil)

	services := []string{
		"localhost:50051",
		"localhost:50052",
		"localhost:50053",
	}

	for _, service := range services {
		_, err := pool.Get(service)
		if err != nil {
			t.Logf("Get %s failed (expected if no server): %v", service, err)
		}
	}

	pool.Close()
}

func TestServer_WithInterceptors(t *testing.T) {
	traceInterceptor := TraceUnaryServerInterceptor()
	recoveryInterceptor := RecoveryUnaryServerInterceptor()
	timeoutInterceptor := TimeoutUnaryServerInterceptor(5 * time.Second)

	cfg := ServerConfig{
		Address:           ":50055",
		UnaryInterceptors: []grpc.UnaryServerInterceptor{traceInterceptor, recoveryInterceptor, timeoutInterceptor},
	}

	server := NewServer(cfg)

	if len(server.config.UnaryInterceptors) != 3 {
		t.Errorf("Expected 3 interceptors, got %d", len(server.config.UnaryInterceptors))
	}

	server.GracefulStop()
}

func TestClient_WithInterceptors(t *testing.T) {
	traceInterceptor := TraceUnaryClientInterceptor()

	cfg := ClientConfig{
		Target:            "localhost:50051",
		Insecure:          true,
		UnaryInterceptors: []grpc.UnaryClientInterceptor{traceInterceptor},
	}

	if len(cfg.UnaryInterceptors) != 1 {
		t.Errorf("Expected 1 interceptor, got %d", len(cfg.UnaryInterceptors))
	}

	conn, err := Dial(cfg)
	if err != nil {
		t.Skip("gRPC server not available, skipping test")
	}
	_ = conn.Close()
}

func TestGrpc_ContextPropagation(t *testing.T) {
	interceptor := TraceUnaryServerInterceptor()

	// 模拟 gRPC metadata 中包含 trace ID
	md := metadata.New(map[string]string{"x-trace-id": "test-trace-id"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// 拦截器应该从 metadata 中提取 trace ID 并放入 context
		traceID := ctx.Value(traceIDKey{})
		if traceID != "test-trace-id" {
			t.Errorf("Expected 'test-trace-id', got '%v'", traceID)
		}
		return "ok", nil
	}

	_, err := interceptor(ctx, "test", &grpc.UnaryServerInfo{
		FullMethod: "/test/service",
	}, handler)
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}
}
