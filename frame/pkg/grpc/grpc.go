package grpc

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/registry"
	"github.com/LandcLi/landc-go/tools/generate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// ServerConfig gRPC 服务器配置
type ServerConfig struct {
	Address            string
	MaxRecvMsgSize     int
	MaxSendMsgSize     int
	UnaryInterceptors  []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor
}

// Server gRPC 服务器封装
type Server struct {
	server *grpc.Server
	config ServerConfig
}

// NewServer 创建 gRPC 服务器
func NewServer(config ServerConfig) *Server {
	var opts []grpc.ServerOption

	if config.MaxRecvMsgSize > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(config.MaxRecvMsgSize))
	}
	if config.MaxSendMsgSize > 0 {
		opts = append(opts, grpc.MaxSendMsgSize(config.MaxSendMsgSize))
	}
	if len(config.UnaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(config.UnaryInterceptors...))
	}
	if len(config.StreamInterceptors) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(config.StreamInterceptors...))
	}

	return &Server{
		server: grpc.NewServer(opts...),
		config: config,
	}
}

// GetServer 获取底层 grpc.Server 用于注册服务
func (s *Server) GetServer() *grpc.Server {
	return s.server
}

// Run 启动 gRPC 服务器
func (s *Server) Run() error {
	addr := s.config.Address
	if addr == "" {
		addr = ":50051"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	return s.server.Serve(ln)
}

// GracefulStop 优雅停止
func (s *Server) GracefulStop() {
	s.server.GracefulStop()
}

// Stop 强制停止
func (s *Server) Stop() {
	s.server.Stop()
}

// --- 客户端 ---

// ClientConfig gRPC 客户端配置
type ClientConfig struct {
	Target             string
	Insecure           bool
	Timeout            time.Duration
	UnaryInterceptors  []grpc.UnaryClientInterceptor
	StreamInterceptors []grpc.StreamClientInterceptor
}

// Dial 建立 gRPC 连接
func Dial(config ClientConfig) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	if config.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if len(config.UnaryInterceptors) > 0 {
		opts = append(opts, grpc.WithChainUnaryInterceptor(config.UnaryInterceptors...))
	}
	if len(config.StreamInterceptors) > 0 {
		opts = append(opts, grpc.WithChainStreamInterceptor(config.StreamInterceptors...))
	}

	ctx := context.Background()
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	conn, err := grpc.DialContext(ctx, config.Target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", config.Target, err)
	}
	return conn, nil
}

// DialWithRegistry 通过服务注册中心建立连接
func DialWithRegistry(resolver *registry.Resolver, serviceName string, config ClientConfig) (*grpc.ClientConn, error) {
	instance, err := resolver.Resolve(serviceName)
	if err != nil {
		return nil, err
	}
	config.Target = instance.Endpoint()
	return Dial(config)
}

// --- 拦截器 ---

type traceIDKey struct{}

// TraceUnaryServerInterceptor 链路追踪（服务端 Unary 拦截器）
func TraceUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		traceID := ""
		if values := md.Get("x-trace-id"); len(values) > 0 {
			traceID = values[0]
		}
		if traceID == "" {
			traceID = generate.UUID()
		}

		ctx = context.WithValue(ctx, traceIDKey{}, traceID)
		_ = grpc.SetHeader(ctx, metadata.Pairs("x-trace-id", traceID))

		return handler(ctx, req)
	}
}

// TraceUnaryClientInterceptor 链路追踪（客户端 Unary 拦截器）
func TraceUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		traceID, _ := ctx.Value(traceIDKey{}).(string)
		if traceID == "" {
			traceID = generate.UUID()
		}
		ctx = metadata.AppendToOutgoingContext(ctx, "x-trace-id", traceID)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// RecoveryUnaryServerInterceptor panic recovery
func RecoveryUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic recovered in gRPC handler %s: %v", info.FullMethod, r)
			}
		}()
		return handler(ctx, req)
	}
}

// TimeoutUnaryServerInterceptor 超时控制
func TimeoutUnaryServerInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return handler(ctx, req)
	}
}

// GetTraceID 从 context 获取 TraceID
func GetTraceID(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDKey{}).(string)
	return traceID
}

// --- 连接池 ---

// ConnPool gRPC 连接池
type ConnPool struct {
	conns    map[string]*grpc.ClientConn
	mu       sync.RWMutex
	config   ClientConfig
	resolver *registry.Resolver
}

// NewConnPool 创建连接池
func NewConnPool(config ClientConfig, resolver *registry.Resolver) *ConnPool {
	return &ConnPool{
		conns:    make(map[string]*grpc.ClientConn),
		config:   config,
		resolver: resolver,
	}
}

// Get 获取连接
func (p *ConnPool) Get(serviceName string) (*grpc.ClientConn, error) {
	p.mu.RLock()
	conn, ok := p.conns[serviceName]
	p.mu.RUnlock()
	if ok {
		return conn, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, ok := p.conns[serviceName]; ok {
		return conn, nil
	}

	var err error
	if p.resolver != nil {
		conn, err = DialWithRegistry(p.resolver, serviceName, p.config)
	} else {
		cfg := p.config
		cfg.Target = serviceName
		conn, err = Dial(cfg)
	}
	if err != nil {
		return nil, err
	}

	p.conns[serviceName] = conn
	return conn, nil
}

// Close 关闭所有连接
func (p *ConnPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for name, conn := range p.conns {
		if err := conn.Close(); err != nil {
			lastErr = err
		}
		delete(p.conns, name)
	}
	return lastErr
}
