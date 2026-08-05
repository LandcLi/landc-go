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
