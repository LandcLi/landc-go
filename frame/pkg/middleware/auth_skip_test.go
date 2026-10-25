package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestAuthSkipsProbeEndpoints 验证健康检查端点免鉴权。
//
// 编排系统（k8s / ECS / 负载均衡）的探针不会携带 Bearer Token。
// 一旦这些路径被鉴权拦下，探活就会永远失败，外在表现是 Pod 反复重启、
// 容器长期 unhealthy、发布卡在 rollout。这类故障排查成本极高，
// 因此用测试把行为钉死。
func TestAuthSkipsProbeEndpoints(t *testing.T) {
	r := gin.New()
	r.Use(Auth())

	ok := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
	for _, p := range []string{"/health", "/ready", "/metrics", "/livez", "/readyz", "/healthz"} {
		r.GET(p, ok)
	}
	r.GET("/api/protected", ok)

	probes := []string{"/health", "/ready", "/metrics", "/livez", "/readyz", "/healthz"}
	for _, p := range probes {
		t.Run("probe"+p, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, p, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("%s should bypass auth, got status %d body=%s", p, w.Code, w.Body.String())
			}
		})
	}

	t.Run("business path still requires token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("business path should require auth, got status %d", w.Code)
		}
	})
}

// TestAuthWithSkipExtra 验证可追加自定义免鉴权路径。
func TestAuthWithSkipExtra(t *testing.T) {
	r := gin.New()
	r.Use(AuthWithSkip("/api/public/webhook"))

	ok := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
	r.POST("/api/public/webhook", ok)
	r.GET("/health", ok)
	r.GET("/api/private", ok)

	cases := []struct {
		path   string
		method string
		want   int
	}{
		{"/api/public/webhook", http.MethodPost, http.StatusOK},
		{"/health", http.MethodGet, http.StatusOK}, // 默认跳过列表仍然生效
		{"/api/private", http.MethodGet, http.StatusUnauthorized},
	}

	for _, c := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(c.method, c.path, nil)
		r.ServeHTTP(w, req)
		if w.Code != c.want {
			t.Errorf("%s %s: want %d, got %d body=%s", c.method, c.path, c.want, w.Code, w.Body.String())
		}
	}
}
