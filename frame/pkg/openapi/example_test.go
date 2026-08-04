package openapi_test

import (
	"encoding/json"
	"fmt"

	"github.com/LandcLi/landc-go/frame/pkg/meta"
	"github.com/LandcLi/landc-go/frame/pkg/openapi"
	"github.com/LandcLi/landc-go/frame/pkg/web"
	"github.com/gin-gonic/gin"
)

type UserController struct{}

type LoginRequest struct {
	meta.Meta `path:"/user/login" method:"POST" description:"用户登录"`
	Username  string `json:"username" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (c *UserController) Login(req *LoginRequest) (*LoginResponse, error) {
	return &LoginResponse{Token: "token"}, nil
}

// 演示：从 web.Server 自动收集已注册控制器生成 OpenAPI 文档。
func ExampleGenerator_RegisterServer() {
	gin.SetMode(gin.TestMode)

	server := web.NewServer(&web.ServerConfig{Addr: ":8080"})
	_ = server.RegisterHandler(&UserController{}, web.WithPrefix("/api/v1"))

	gen := openapi.NewGenerator(openapi.Info{Title: "User Service", Version: "1.0"})
	gen.AddBearerAuth()
	gen.RegisterServer(server) // 自动收集，路径为最终生效路由

	doc := gen.Generate()
	item := doc.Paths["/api/v1/user/login"]
	b, _ := json.Marshal(item.Post)
	fmt.Printf("path=%s method=POST op=%s\n", "/api/v1/user/login", b)

	// Output:
	// path=/api/v1/user/login method=POST op={"tags":["User"],"summary":"用户登录","operationId":"Login","requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","properties":{"username":{"type":"string"}},"required":["username"]}}}},"responses":{"200":{"description":"Successful response","content":{"application/json":{"schema":{"$ref":"#/components/schemas/LoginResponse"}}}}}}
}
