package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/cache"
	"github.com/LandcLi/landc-go/frame/pkg/meta"
	"github.com/LandcLi/landc-go/frame/pkg/resource"
	"github.com/gin-gonic/gin"
)

type LibraryController struct{}

type LibraryCheckRequest struct {
	meta.Meta `path:"/library/check" method:"GET"`
}

type LibraryCheckResponse struct {
	HasScope bool   `json:"has_scope"`
	ScopeDB  string `json:"scope_db"`
}

// Check 第一参数为 context.Context 接口，框架应传入携带资源作用域的请求 ctx。
func (c *LibraryController) Check(ctx context.Context, req *LibraryCheckRequest) (*LibraryCheckResponse, error) {
	s, ok := resource.FromContext(ctx)
	if !ok {
		return &LibraryCheckResponse{}, nil
	}
	return &LibraryCheckResponse{HasScope: true, ScopeDB: s.DB}, nil
}

func TestRegisterLibraryScopeInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 注册命名缓存（本地实现，无需网络）
	cache.InitNamedCacheWithLocal("lum", 100)

	s := NewServer(&ServerConfig{Addr: ":8080"})
	err := s.RegisterLibrary(&LibraryController{},
		WithScope(resource.Scope{Name: "lum", Cache: "lum"}),
	)
	if err != nil {
		t.Fatalf("RegisterLibrary: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/library/check", http.NoBody)
	s.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"has_scope":true`) {
		t.Errorf("scope not injected into request ctx: %s", w.Body.String())
	}
}

func TestRegisterLibraryScopeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := NewServer(&ServerConfig{Addr: ":8080"})

	// 引用未注册的命名 DB → 注册期报错（不静默回退）
	err := s.RegisterLibrary(&LibraryController{},
		WithScope(resource.Scope{Name: "lum", DB: "not-exist"}),
	)
	if err == nil {
		t.Fatal("RegisterLibrary should fail for unregistered named db")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("error should mention not registered, got: %v", err)
	}
}

// CacheController 演示嵌入服务通过 GetCacheFrom(ctx) 使用命名缓存。
type CacheController struct{}

type CachePutRequest struct {
	meta.Meta `path:"/cache/put" method:"POST"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

type CacheGetRequest struct {
	meta.Meta `path:"/cache/get" method:"GET"`
	Key       string `json:"key" source:"query"`
}

type CacheValueResponse struct {
	Value string `json:"value"`
}

func (c *CacheController) Put(ctx context.Context, req *CachePutRequest) (*CacheValueResponse, error) {
	cc := cache.GetCacheFrom(ctx)
	if cc == nil {
		return &CacheValueResponse{}, nil
	}
	return &CacheValueResponse{}, cc.Set(ctx, req.Key, req.Value, 0)
}

func (c *CacheController) Get(ctx context.Context, req *CacheGetRequest) (*CacheValueResponse, error) {
	cc := cache.GetCacheFrom(ctx)
	if cc == nil {
		return &CacheValueResponse{}, nil
	}
	v, _ := cc.Get(ctx, req.Key)
	return &CacheValueResponse{Value: v}, nil
}

func TestLibraryScopeResourceIsolation(t *testing.T) {
	// A6 集成验证：作用域路由命中命名缓存，普通路由命中全局缓存，数据隔离。
	gin.SetMode(gin.TestMode)
	cache.InitNamedCacheWithLocal("lum", 100)
	cache.InitGlobalCacheWithLocal(1000)

	s := NewServer(&ServerConfig{Addr: ":8080"})
	ctrl := &CacheController{}

	// 命名作用域路由（嵌入服务形态）
	if err := s.RegisterLibrary(ctrl,
		WithScope(resource.Scope{Name: "lum", Cache: "lum"}),
		WithRegisterOptions(WithPrefix("/lum")),
	); err != nil {
		t.Fatalf("RegisterLibrary: %v", err)
	}
	// 普通全局路由（宿主自身形态）
	if err := s.RegisterHandler(ctrl, WithPrefix("/global")); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}

	postJSON := func(path, body string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.Engine().ServeHTTP(w, req)
		return w
	}
	get := func(path string) string {
		req, _ := http.NewRequest("GET", path, http.NoBody)
		w := httptest.NewRecorder()
		s.Engine().ServeHTTP(w, req)
		return w.Body.String()
	}

	// 写入两个缓存：同名 key，不同值
	if w := postJSON("/lum/cache/put", `{"key":"k","value":"from-lum"}`); w.Code != http.StatusOK {
		t.Fatalf("lum put = %d: %s", w.Code, w.Body.String())
	}
	if w := postJSON("/global/cache/put", `{"key":"k","value":"from-global"}`); w.Code != http.StatusOK {
		t.Fatalf("global put = %d: %s", w.Code, w.Body.String())
	}

	// 读取：应各自命中自己的缓存实例（数据隔离）
	if got := get("/lum/cache/get?key=k"); !strings.Contains(got, `"value":"from-lum"`) {
		t.Errorf("lum get = %s, want from-lum", got)
	}
	if got := get("/global/cache/get?key=k"); !strings.Contains(got, `"value":"from-global"`) {
		t.Errorf("global get = %s, want from-global", got)
	}
}

func TestRegisterLibraryNoScopeBackwardCompat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := NewServer(&ServerConfig{Addr: ":8080"})

	// 不指定作用域 → 行为与 RegisterHandler 一致（scope 未注入，HasScope=false）
	if err := s.RegisterLibrary(&LibraryController{}); err != nil {
		t.Fatalf("RegisterLibrary: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/library/check", http.NoBody)
	s.Engine().ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), `"has_scope":false`) {
		t.Errorf("no scope expected, got: %s", w.Body.String())
	}
}
