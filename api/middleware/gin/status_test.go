package gin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	g "github.com/gin-gonic/gin"
)

// TestMiddlewarePreservesStatusCode 验证中间件不篡改 handler 写入的 HTTP 状态码。
//
// 历史缺陷：bodyLogWriter 只拦截 body、不记录状态码，而 gin 的 responseWriter
// 要等 Write 时才真正写出 header —— Write 被拦截后底层从未写出，中间件又无条件
// 写 200。结果 handler 返回 400/500 时，HTTP 状态码仍是 200，只有 body 里的
// 业务 code 是对的。网关、监控、按状态码判断的客户端会全部误判为成功。
func TestMiddlewarePreservesStatusCode(t *testing.T) {
	g.SetMode(g.TestMode)

	r := g.New()
	r.Use(GinMiddleware())
	r.POST("/ok", func(c *g.Context) {
		c.JSON(http.StatusOK, g.H{"code": 10000, "message": "success", "data": g.H{"id": 1}})
	})
	r.POST("/bad", func(c *g.Context) {
		c.JSON(http.StatusBadRequest, g.H{"code": 40000, "message": "bad request"})
	})
	r.POST("/boom", func(c *g.Context) {
		c.JSON(http.StatusInternalServerError, g.H{"code": 50000, "message": "internal"})
	})

	cases := []struct {
		path string
		want int
	}{
		{"/ok", http.StatusOK},
		{"/bad", http.StatusBadRequest},
		{"/boom", http.StatusInternalServerError},
	}

	for _, tc := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, tc.path, http.NoBody)
		r.ServeHTTP(w, req)

		if w.Code != tc.want {
			t.Errorf("%s: status = %d, want %d (body=%s)", tc.path, w.Code, tc.want, w.Body.String())
		}
	}
}

// TestMiddlewareKeepsSuccessData 确保修复状态码后成功响应的 data 没有被吞掉。
func TestMiddlewareKeepsSuccessData(t *testing.T) {
	g.SetMode(g.TestMode)

	r := g.New()
	r.Use(GinMiddleware())
	r.POST("/ok", func(c *g.Context) {
		c.JSON(http.StatusOK, g.H{"code": 10000, "message": "success", "data": g.H{"name": "研发中心"}})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/ok", http.NoBody))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"data"`) || !strings.Contains(body, "研发中心") {
		t.Errorf("响应丢失 data: %s", body)
	}
}
