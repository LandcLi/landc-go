//go:build goframe
// +build goframe

package example

import (
	"github.com/LandcLi/landc-go/api/middleware/goframe"

	"github.com/gogf/gf/v2/net/ghttp"
)

// GoFrameExample GoFrame框架使用示例
func GoFrameExample() {
	s := ghttp.GetServer()

	// 使用默认配置的中间件
	s.Use(goframe.GoFrameMiddleware())

	// 定义路由
	s.Group("/api", func(group *ghttp.RouterGroup) {
		group.GET("/hello", func(r *ghttp.Request) {
			r.Response.WriteJson(map[string]string{"hello": "world"})
		})
	})

	s.SetPort(8081)
	s.Run()
}
