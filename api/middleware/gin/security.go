package gin

import (
	"github.com/LandcLi/landc-go/tools/security"
	"github.com/gin-gonic/gin"
)

// SecurityHeaders 安全响应头中间件。
// 自动为每个响应添加安全相关的 HTTP 头。
//
// 用法:
//
//	r.Use(ginmw.SecurityHeaders(security.DefaultHeadersConfig))
func SecurityHeaders(cfg security.HeadersConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// X-Content-Type-Options: nosniff
		if v := cfg.XContentTypeOptionsHeader(); v != "" {
			c.Header("X-Content-Type-Options", v)
		}

		// X-Frame-Options: DENY
		if v := cfg.XFrameOptionsHeader(); v != "" {
			c.Header("X-Frame-Options", v)
		}

		// Strict-Transport-Security
		if v := cfg.HSTSHeader(); v != "" {
			c.Header("Strict-Transport-Security", v)
		}

		// Content-Security-Policy
		if cfg.CSP != "" {
			c.Header("Content-Security-Policy", cfg.CSP)
		}

		// Referrer-Policy
		if cfg.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", cfg.ReferrerPolicy)
		}

		c.Next()
	}
}
