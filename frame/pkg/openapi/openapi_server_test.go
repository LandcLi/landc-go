package openapi

import (
	"os"
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/meta"
	"github.com/LandcLi/landc-go/frame/pkg/web"
	"github.com/gin-gonic/gin"
)

type (
	DocController struct{}

	DocRequest struct {
		meta.Meta `path:"/doc/login" method:"POST" description:"登录"`
		Username  string `json:"username" binding:"required"`
	}

	DocResponse struct {
		Token string `json:"token"`
	}
)

func (c *DocController) Login(req *DocRequest) (*DocResponse, error) {
	return &DocResponse{Token: "t"}, nil
}

func pathKeys(m map[string]*PathItem) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestRegisterServerUsesFinalRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := web.NewServer(&web.ServerConfig{Addr: ":8080"})

	// 注册时通过选项加前缀，验证文档使用最终生效路由
	if err := s.RegisterHandler(&DocController{}, web.WithPrefix("/api/v2")); err != nil {
		t.Fatalf("register handler: %v", err)
	}

	g := NewGenerator(Info{Title: "Demo", Version: "1.0"})
	g.RegisterServer(s)

	doc := g.Generate()
	item := doc.Paths["/api/v2/doc/login"]
	if item == nil {
		t.Fatalf("expected path /api/v2/doc/login, got %v", pathKeys(doc.Paths))
	}
	if item.Post == nil {
		t.Fatal("expected POST operation for login")
	}
	if item.Post.RequestBody == nil {
		t.Fatal("expected request body with required fields")
	}
	if item.Post.Summary != "登录" {
		t.Errorf("summary = %q, want 登录", item.Post.Summary)
	}
}

func TestWriteFile(t *testing.T) {
	g := NewGenerator(Info{Title: "T", Version: "1.0"})
	path := t.TempDir() + "/openapi.json"
	if err := g.WriteFile(path); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("generated file is empty")
	}
}
