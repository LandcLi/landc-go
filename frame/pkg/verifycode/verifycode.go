// Package verifycode 提供基于 framecache 的验证码能力（frame 层包装）。
//
// 与 tools/verifycode 的关系：本包是 frame 层包装——缓存从请求 ctx 解析
// （framecache.GetCacheFrom），因此：
//   - 独立部署使用全局缓存
//   - 库模式嵌入（web.RegisterLibrary + WithScope）自动使用命名缓存
//
// 默认选项：6 位数字、TTL 5 分钟、发送间隔 60s、每日上限 10 次。
// 需要自定义选项时，请直接使用 tools/verifycode 的 Manager。
package verifycode

import (
	"context"
	"errors"

	framecache "github.com/LandcLi/landc-go/frame/pkg/cache"
	toolsvc "github.com/LandcLi/landc-go/tools/verifycode"
)

// Generate 生成验证码并存储，返回验证码值（供业务侧发送短信/邮件）。
// 受发送间隔与每日上限约束；缓存不可用时返回错误。
func Generate(ctx context.Context, key string) (string, error) {
	c := framecache.GetCacheFrom(ctx)
	if c == nil {
		return "", errors.New("verifycode: cache not available")
	}
	return toolsvc.NewManager(framecache.AsToolsCache(c)).Generate(key)
}

// Verify 校验验证码（一次性，成功后删除，防重放）。
func Verify(ctx context.Context, key, input string) bool {
	c := framecache.GetCacheFrom(ctx)
	if c == nil {
		return false
	}
	return toolsvc.NewManager(framecache.AsToolsCache(c)).Verify(key, input)
}
