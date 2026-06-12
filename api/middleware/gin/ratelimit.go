package gin

import (
	"net/http"
	"sync"

	"github.com/LandcLi/landc-go/tools/ratelimit"
	"github.com/gin-gonic/gin"
)

// RateLimiter 基于令牌桶的限流中间件。
// 所有请求共享同一个令牌桶。
//
// 用法:
//
//	tb := ratelimit.NewTokenBucket(100, 200) // 每秒 100 个，突发 200
//	r.Use(ginmw.RateLimiter(tb))
func RateLimiter(tb *ratelimit.TokenBucket) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !tb.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    42900,
				"message": "too many requests",
			})
			return
		}
		c.Next()
	}
}

// GlobalRateLimiter 全局限流器（全局单例，内部创建令牌桶）。
//
// 用法:
//
//	r.Use(ginmw.GlobalRateLimiter(100, 200))
func GlobalRateLimiter(rate float64, burst int) gin.HandlerFunc {
	var (
		once sync.Once
		tb   *ratelimit.TokenBucket
	)
	once.Do(func() {
		tb = ratelimit.NewTokenBucket(rate, burst)
	})

	return RateLimiter(tb)
}
