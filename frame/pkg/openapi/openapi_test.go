package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/meta"
	"github.com/gin-gonic/gin"
)

type TestUserReq struct {
	meta.Meta `path:"/users" method:"POST" description:"创建用户"`
	Name      string `json:"name" source:"body" binding:"required" description:"用户名"`
	Email     string `json:"email" source:"body" binding:"required" description:"邮箱"`
	Age       int    `json:"age" source:"body" description:"年龄"`
}

type TestListReq struct {
	meta.Meta `path:"/users" method:"GET" description:"用户列表"`
	Page      int    `source:"query" name:"page" d:"1" description:"页码"`
	PageSize  int    `source:"query" name:"page_size" d:"20" description:"每页数量"`
	Keyword   string `source:"query" name:"keyword" description:"搜索关键词"`
}

type TestGetReq struct {
	meta.Meta `path:"/users/:id" method:"GET" description:"获取用户详情"`
	ID        int `source:"path" name:"id" description:"用户ID"`
}

type TestController struct {
	meta.Meta `path:"/api/v1" group:"User"`
}

func (c *TestController) Create(req *TestUserReq) {}
func (c *TestController) List(req *TestListReq)   {}
func (c *TestController) Get(req *TestGetReq)     {}

func TestGenerator_RegisterController(t *testing.T) {
	gen := NewGenerator(Info{
		Title:   "Test API",
		Version: "1.0.0",
	})

	gen.AddBearerAuth()
	gen.RegisterController(&TestController{})

	doc := gen.Generate()

	if doc.OpenAPI != "3.0.3" {
		t.Errorf("expected openapi 3.0.3, got %s", doc.OpenAPI)
	}

	if doc.Info.Title != "Test API" {
		t.Errorf("expected title 'Test API', got '%s'", doc.Info.Title)
	}

	// 验证路径被注册
	if len(doc.Paths) == 0 {
		t.Fatal("expected paths to be generated")
	}

	// 验证 POST /api/v1/users
	usersPath, ok := doc.Paths["/api/v1/users"]
	if !ok {
		t.Fatal("expected /api/v1/users path")
	}

	if usersPath.Post == nil {
		t.Fatal("expected POST operation on /api/v1/users")
	}

	if usersPath.Post.Summary != "创建用户" {
		t.Errorf("expected summary '创建用户', got '%s'", usersPath.Post.Summary)
	}

	if usersPath.Post.RequestBody == nil {
		t.Fatal("expected request body for POST")
	}

	// 验证 GET /api/v1/users
	if usersPath.Get == nil {
		t.Fatal("expected GET operation on /api/v1/users")
	}

	if len(usersPath.Get.Parameters) < 2 {
		t.Errorf("expected at least 2 query parameters, got %d", len(usersPath.Get.Parameters))
	}

	// 验证 GET /api/v1/users/:id
	userIDPath, ok := doc.Paths["/api/v1/users/:id"]
	if !ok {
		t.Fatal("expected /api/v1/users/:id path")
	}
	if userIDPath.Get == nil {
		t.Fatal("expected GET operation on /api/v1/users/:id")
	}

	// 验证 path 参数
	hasPathParam := false
	for _, p := range userIDPath.Get.Parameters {
		if p.In == "path" && p.Name == "id" {
			hasPathParam = true
			if !p.Required {
				t.Error("path parameter should be required")
			}
		}
	}
	if !hasPathParam {
		t.Error("expected path parameter 'id'")
	}

	// 验证 security scheme
	if doc.Components.SecuritySchemes["bearerAuth"] == nil {
		t.Error("expected bearerAuth security scheme")
	}

	// 验证 tags
	if len(doc.Tags) == 0 {
		t.Error("expected tags")
	}
}

func TestGenerator_JSON(t *testing.T) {
	gen := NewGenerator(Info{Title: "Test", Version: "1.0.0"})
	gen.RegisterController(&TestController{})

	data, err := gen.JSON()
	if err != nil {
		t.Fatalf("JSON() error: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if doc["openapi"] != "3.0.3" {
		t.Errorf("expected openapi 3.0.3 in JSON")
	}
}

func TestGenerator_Handler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gen := NewGenerator(Info{Title: "Test", Version: "1.0.0"})
	gen.RegisterController(&TestController{})

	r := gin.New()
	r.GET("/openapi.json", gen.Handler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/openapi.json", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected application/json, got %s", contentType)
	}
}

func TestGenerator_SwaggerUIHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gen := NewGenerator(Info{Title: "Test API", Version: "1.0.0"})

	r := gin.New()
	r.GET("/swagger", gen.SwaggerUIHandler("/openapi.json"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/swagger", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty HTML body")
	}
}
