package cache

import (
	"context"
	"time"

	"github.com/LandcLi/landc-go/tools/ratelimit"
)

// AsToolsCache 把 frame 的 Cache（方法带 ctx）适配为 tools/ratelimit.Cache（无 ctx），
// 供 frame 层对 tools 组件的包装复用（限流、验证码等）。
func AsToolsCache(c Cache) ratelimit.Cache {
	return toolsCacheAdapter{c: c}
}

// toolsCacheAdapter 适配器：frame Cache 的方法都带 ctx，tools/ratelimit.Cache 不带。
type toolsCacheAdapter struct {
	c Cache
}

func (a toolsCacheAdapter) Get(key string) (string, error) {
	return a.c.Get(context.Background(), key)
}

func (a toolsCacheAdapter) Set(key, value string, ttl time.Duration) error {
	return a.c.Set(context.Background(), key, value, ttl)
}

func (a toolsCacheAdapter) Delete(key string) error {
	return a.c.Delete(context.Background(), key)
}

func (a toolsCacheAdapter) Exists(key string) (bool, error) {
	return a.c.Exists(context.Background(), key)
}

func (a toolsCacheAdapter) Incr(key string, ttl time.Duration) (int64, error) {
	return a.c.Incr(context.Background(), key, ttl)
}
