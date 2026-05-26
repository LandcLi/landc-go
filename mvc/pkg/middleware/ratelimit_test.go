package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests within limit", func(t *testing.T) {
		r := gin.New()
		r.Use(RateLimiter(RateLimiterConfig{Rate: 10, Burst: 10}))
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		// 前 10 个请求应该通过
		for i := 0; i < 10; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			r.ServeHTTP(w, req)
			if w.Code != 200 {
				t.Errorf("request %d: expected 200, got %d", i, w.Code)
			}
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		r := gin.New()
		r.Use(RateLimiter(RateLimiterConfig{Rate: 2, Burst: 2}))
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		// 消耗所有令牌
		for i := 0; i < 2; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			r.ServeHTTP(w, req)
		}

		// 第 3 个请求应该被限流
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		r.ServeHTTP(w, req)
		if w.Code != 429 {
			t.Errorf("expected 429, got %d", w.Code)
		}
	})

	t.Run("different IPs have separate limits", func(t *testing.T) {
		r := gin.New()
		r.Use(RateLimiter(RateLimiterConfig{Rate: 1, Burst: 1}))
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		// IP1 的第一个请求
		w1 := httptest.NewRecorder()
		req1, _ := http.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "1.1.1.1:1234"
		r.ServeHTTP(w1, req1)
		if w1.Code != 200 {
			t.Errorf("IP1 first request: expected 200, got %d", w1.Code)
		}

		// IP2 的第一个请求也应该通过
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "2.2.2.2:1234"
		r.ServeHTTP(w2, req2)
		if w2.Code != 200 {
			t.Errorf("IP2 first request: expected 200, got %d", w2.Code)
		}
	})
}

func TestGlobalRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(GlobalRateLimiter(3, 3))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// 消耗所有令牌
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
	}

	// 不同 IP 也应该被限流（全局共享）
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "9.9.9.9:1234"
	r.ServeHTTP(w, req)
	if w.Code != 429 {
		t.Errorf("expected 429, got %d", w.Code)
	}
}

func TestCircuitBreaker(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("opens after threshold", func(t *testing.T) {
		r := gin.New()
		r.Use(CircuitBreaker(CircuitBreakerConfig{
			Threshold: 3,
			Window:    10 * time.Second,
			CoolDown:  100 * time.Millisecond,
		}))
		r.GET("/test", func(c *gin.Context) {
			c.Status(500)
		})

		// 触发 3 次失败
		for i := 0; i < 3; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)
		}

		// 第 4 次应该被熔断
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		if w.Code != 503 {
			t.Errorf("expected 503 (circuit open), got %d", w.Code)
		}
	})

	t.Run("recovers after cooldown", func(t *testing.T) {
		r := gin.New()
		r.Use(CircuitBreaker(CircuitBreakerConfig{
			Threshold: 2,
			Window:    10 * time.Second,
			CoolDown:  50 * time.Millisecond,
		}))
		shouldFail := true
		r.GET("/test", func(c *gin.Context) {
			if shouldFail {
				c.Status(500)
			} else {
				c.String(200, "ok")
			}
		})

		// 触发熔断
		for i := 0; i < 2; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)
		}

		// 等待冷却
		time.Sleep(80 * time.Millisecond)

		// 现在修复服务
		shouldFail = false

		// 半开状态放行一个试探请求
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("expected 200 (half-open success), got %d", w.Code)
		}

		// 后续请求也应该正常
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w2, req2)
		if w2.Code != 200 {
			t.Errorf("expected 200 (recovered), got %d", w2.Code)
		}
	})
}

func TestRateLimiter_Concurrent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RateLimiter(RateLimiterConfig{Rate: 100, Burst: 100}))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			r.ServeHTTP(w, req)
			// 不检查具体状态码，只确认不 panic
		}()
	}
	wg.Wait()
}
