package example

import (
	"net/http"

	ginApi "github.com/LandcLi/landc-go/api/middleware/gin"

	"github.com/gin-gonic/gin"
)

// GinExample Gin框架使用示例
func GinExample() {
	r := gin.Default()

	// 使用默认配置的中间件
	r.Use(ginApi.GinMiddleware())

	// 定义路由
	r.GET("/api/hello", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"hello": "world"})
	})

	r.Run(":8080")
}

// GinExampleWithConfig Gin框架使用自定义配置示例
func GinExampleWithConfig() {
	r := gin.Default()

	// 使用自定义配置
	config := ginApi.NewGinMiddlewareConfig().
		WithEnableHeaders(true)

	r.Use(ginApi.GinMiddlewareWithConfig(config))

	r.GET("/api/hello", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"hello": "world"})
	})

	r.Run(":8080")
}
