// Package resource 提供命名资源作用域（Scope）。
//
// 库模式嵌入场景下，嵌入服务可通过 Scope 指定使用宿主为之准备的独立命名资源
// （配置 / 数据库 / 缓存），未指定的字段回退到全局资源。作用域经 context 传递，
// 由注册入口（web.RegisterLibrary）自动注入到请求上下文，controller / service
// 通过 db.GetDBFrom(ctx) / cache.GetCacheFrom(ctx) / config.GetConfigFrom(ctx) 解析。
package resource

import "context"

// scopeKey 是 context 中作用域的键类型（避免与其他包冲突）。
type scopeKey struct{}

// Scope 描述一个命名资源作用域。
//
// 字段为空表示回退全局资源；非空表示使用对应命名资源。
// 命名资源必须先注册（config.InitNamedConfig / db.InitNamedDB / cache.InitNamedCache），
// 注册入口（web.RegisterLibrary）会做存在性校验。
type Scope struct {
	Name   string // 作用域名，如 "lum"
	Config string // 命名配置名；空 = 回退全局
	DB     string // 命名数据库名；空 = 回退全局
	Cache  string // 命名缓存名；空 = 回退全局
}

// Empty 报告作用域是否未指定任何命名资源（全部回退全局）。
func (s Scope) Empty() bool {
	return s.Config == "" && s.DB == "" && s.Cache == ""
}

// WithScope 将作用域绑定到 ctx。
func WithScope(ctx context.Context, s Scope) context.Context {
	return context.WithValue(ctx, scopeKey{}, s)
}

// FromContext 从 ctx 读取作用域；未绑定返回 ok=false。
func FromContext(ctx context.Context) (Scope, bool) {
	s, ok := ctx.Value(scopeKey{}).(Scope)
	return s, ok
}
