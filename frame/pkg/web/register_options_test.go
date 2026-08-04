package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/meta"
	"github.com/gin-gonic/gin"
)

type (
	OptionController struct{}

	OptionRequest struct {
		meta.Meta `path:"/option/login" method:"POST" description:"用户登录"`
		Username  string `json:"username" binding:"required"`
		Password  string `json:"password" binding:"required"`
	}

	OptionResponse struct {
		Username string `json:"username"`
	}
)

func (c *OptionController) Login(req *OptionRequest) (*OptionResponse, error) {
	return &OptionResponse{Username: req.Username}, nil
}

func (c *OptionController) Ping() (map[string]string, error) {
	return map[string]string{"pong": "true"}, nil
}

func newOptionServer(t *testing.T) (*Server, *gin.Engine, *[]string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	server := NewServer(&ServerConfig{Addr: ":8080"})

	order := make([]string, 0)
	audit := func(name string) gin.HandlerFunc {
		return func(c *gin.Context) { order = append(order, name); c.Next() }
	}

	err := server.RegisterHandler(&OptionController{},
		WithPrefix("/api/v2"),
		WithMethodHTTPMethod("Ping", "PUT"),
		WithMethodPath("Ping", "/ping"),
		WithMethodMiddleware("Login", audit("login-mw-1"), audit("login-mw-2")),
		WithControllerMiddleware(audit("ctrl-mw")),
	)
	if err != nil {
		t.Fatalf("register handler: %v", err)
	}
	engine := server.Engine()
	return server, engine, &order
}

func postLogin(router *gin.Engine) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("POST", "/api/v2/option/login",
		strings.NewReader(`{"username":"u","password":"p"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestRegisterOptionsPrefixAndOverrides(t *testing.T) {
	_, router, _ := newOptionServer(t)

	t.Run("POST login under prefix", func(t *testing.T) {
		if w := postLogin(router); w.Code != http.StatusOK {
			t.Errorf("POST /api/v2/option/login = %d, want 200", w.Code)
		}
	})

	t.Run("Ping moved and method overridden", func(t *testing.T) {
		// 编译期默认 GET /ping，被覆盖为 PUT /api/v2/ping
		req, _ := http.NewRequest("PUT", "/api/v2/ping", http.NoBody)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("PUT /api/v2/ping = %d, want 200", w.Code)
		}

		// 旧路径与方法不应再存在
		req2, _ := http.NewRequest("GET", "/api/v2/ping", http.NoBody)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		if w2.Code == http.StatusOK {
			t.Error("old GET /api/v2/ping should not be registered")
		}
	})
}

func TestRegisterOptionsMiddlewareOrder(t *testing.T) {
	_, router, order := newOptionServer(t)

	*order = nil
	postLogin(router)

	want := []string{"ctrl-mw", "login-mw-1", "login-mw-2"}
	if len(*order) != len(want) {
		t.Fatalf("middleware order = %v, want %v", *order, want)
	}
	for i := range want {
		if (*order)[i] != want[i] {
			t.Fatalf("middleware order = %v, want %v", *order, want)
		}
	}
}

func TestRoutesQuery(t *testing.T) {
	server, _, _ := newOptionServer(t)

	routes := server.Routes()
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2: %+v", len(routes), routes)
	}

	var login, ping *RouteInfo
	for i := range routes {
		switch routes[i].HandlerName {
		case "Login":
			login = &routes[i]
		case "Ping":
			ping = &routes[i]
		}
	}

	if login == nil || login.Method != "POST" || login.Path != "/api/v2/option/login" {
		t.Errorf("login route = %+v", login)
	}
	if login.Description != "用户登录" {
		t.Errorf("login description = %q, want 用户登录", login.Description)
	}
	if ping == nil || ping.Method != "PUT" || ping.Path != "/api/v2/ping" {
		t.Errorf("ping route = %+v", ping)
	}
}

func TestRoutesQueryNoSideEffect(t *testing.T) {
	server, _, _ := newOptionServer(t)

	before := server.Routes()
	after := server.Routes()
	if len(before) != len(after) {
		t.Fatal("Routes() must be side-effect free")
	}
}
