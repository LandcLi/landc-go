package gin

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	defaultCSRFCookieName = "_csrf_token"
	defaultCSRFHeaderName = "X-CSRF-Token"
)

// CSRFOption CSRF 中间件配置选项。
type CSRFOption func(*csrfConfig)

type csrfConfig struct {
	cookieName string
	headerName string
	secure     bool
}

// WithCSRFCookieName 自定义 CSRF cookie 名称。
func WithCSRFCookieName(name string) CSRFOption {
	return func(c *csrfConfig) {
		c.cookieName = name
	}
}

// WithCSRFHeaderName 自定义 CSRF 请求头名称。
func WithCSRFHeaderName(name string) CSRFOption {
	return func(c *csrfConfig) {
		c.headerName = name
	}
}

// WithCSRFSecure 设置 Secure 标志（HTTPS 时启用）。
func WithCSRFSecure(secure bool) CSRFOption {
	return func(c *csrfConfig) {
		c.secure = secure
	}
}

// CSRF 基于 Double Submit Cookie 模式的 CSRF 防护中间件。
//
// 对于 GET/HEAD/OPTIONS 请求，生成 token 并通过 cookie 下发。
// 对于其他请求，验证请求头中的 token 与 cookie 中的 token 一致。
//
// 用法:
//
//	r.Use(ginmw.CSRF())
//	r.Use(ginmw.CSRF(ginmw.WithCSRFCookieName("_myapp_csrf")))
func CSRF(opts ...CSRFOption) gin.HandlerFunc {
	cfg := &csrfConfig{
		cookieName: defaultCSRFCookieName,
		headerName: defaultCSRFHeaderName,
		secure:     false,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(c *gin.Context) {
		// 安全方法：下发 token
		if c.Request.Method == http.MethodGet ||
			c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodOptions {
			token := generateCSRFToken()
			c.SetCookie(cfg.cookieName, token, 3600, "/", "", cfg.secure, true)
			c.Set("_csrf_token", token)
			c.Next()
			return
		}

		// 变更方法：验证 token
		cookieToken, err := c.Cookie(cfg.cookieName)
		if err != nil || cookieToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    40300,
				"message": "CSRF token missing in cookie",
			})
			return
		}

		headerToken := c.GetHeader(cfg.headerName)
		if headerToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    40300,
				"message": "CSRF token missing in header",
			})
			return
		}

		if cookieToken != headerToken {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    40300,
				"message": "CSRF token mismatch",
			})
			return
		}

		c.Next()
	}
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
