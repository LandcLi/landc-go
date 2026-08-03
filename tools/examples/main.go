// tools 模块使用示例
//
// 运行：go run ./examples
package main

import (
	"fmt"
	"time"

	"github.com/LandcLi/landc-go/tools/cache"
	"github.com/LandcLi/landc-go/tools/email"
	"github.com/LandcLi/landc-go/tools/generate"
	"github.com/LandcLi/landc-go/tools/ratelimit"
)

func main() {
	// 1. UUID / 随机数
	fmt.Println("uuid:", generate.UUID())
	fmt.Println("random int:", generate.MustRandomInt(1, 100))
	fmt.Println("random string:", generate.MustRandomString(12))

	// 2. 本地缓存（LRU + 过期清理）
	c := cache.NewGlobalCacheWithCapacity(100)
	c.Set("user:1", map[string]string{"name": "alice"})
	c.SetWithExpiration("session:1", "token-value", 5*time.Minute)

	if v, ok := c.Get("user:1"); ok {
		fmt.Println("cache hit user:1 =", v)
	}
	if _, ok := c.Get("missing"); !ok {
		fmt.Println("cache miss for missing key")
	}
	c.Delete("user:1")

	// 3. 邮箱校验
	for _, addr := range []string{"foo@example.com", "bad-address"} {
		fmt.Printf("email %q valid=%v\n", addr, email.IsValid(addr))
	}

	// 4. 令牌桶限流
	tb := ratelimit.NewTokenBucket(5, 5) // 每秒 5 个令牌，突发 5
	for i := 0; i < 7; i++ {
		fmt.Printf("allow #%d = %v\n", i+1, tb.Allow())
	}
}
