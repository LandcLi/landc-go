package web

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/LandcLi/landc-go/frame/pkg/cache"
	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/LandcLi/landc-go/frame/pkg/db"
	"github.com/LandcLi/landc-go/frame/pkg/resource"
)

// libraryOptions 为库模式注册选项。
type libraryOptions struct {
	scope   resource.Scope
	regOpts []RegisterOption
}

// LibraryOption 为库模式注册选项。
type LibraryOption func(*libraryOptions)

// WithScope 为嵌入服务指定命名资源作用域。
// 未指定命名资源（全部字段为空）时，嵌入服务回退使用宿主全局资源。
func WithScope(s resource.Scope) LibraryOption {
	return func(o *libraryOptions) { o.scope = s }
}

// WithRegisterOptions 透传注册选项（前缀 / 方法覆盖 / 方法级与控制器级中间件）。
func WithRegisterOptions(opts ...RegisterOption) LibraryOption {
	return func(o *libraryOptions) { o.regOpts = append(o.regOpts, opts...) }
}

// RegisterLibrary 注册嵌入服务控制器。
//
// 与 RegisterHandler 的区别：若指定资源作用域（WithScope），会先校验引用的命名资源
// 均已注册，并为该控制器的所有路由注入作用域中间件——把 Scope 写入请求 context，
// 使 controller / service / dao 可通过 db.GetDBFrom(ctx) / cache.GetCacheFrom(ctx) /
// config.GetConfigFrom(ctx) 解析到命名资源（未指定的字段回退全局）。
//
// 作用域中间件只作用于本控制器的路由，不影响宿主在 RegisterLibrary 之外注册的路由。
// 中间件执行顺序：宿主全局中间件（如 Auth，先挂载者先执行）→ 作用域中间件 → 业务 handler。
func (s *Server) RegisterLibrary(instance interface{}, opts ...LibraryOption) error {
	o := &libraryOptions{}
	for _, opt := range opts {
		opt(o)
	}

	if err := validateScope(o.scope); err != nil {
		return err
	}

	if !o.scope.Empty() {
		o.regOpts = append(o.regOpts, WithControllerMiddleware(scopeMiddleware(o.scope)))
	}
	return s.RegisterHandler(instance, o.regOpts...)
}

// validateScope 校验作用域引用的命名资源均已注册（不静默回退全局，防数据写错库）。
func validateScope(sc resource.Scope) error {
	if sc.Config != "" && !config.HasNamedConfig(sc.Config) {
		return fmt.Errorf("resource scope %q: named config %q not registered (config.InitNamedConfig)", sc.Name, sc.Config)
	}
	if sc.DB != "" && !db.HasNamedDB(sc.DB) {
		return fmt.Errorf("resource scope %q: named db %q not registered (db.InitNamedDB)", sc.Name, sc.DB)
	}
	if sc.Cache != "" && !cache.HasNamedCache(sc.Cache) {
		return fmt.Errorf("resource scope %q: named cache %q not registered (cache.InitNamedCache)", sc.Name, sc.Cache)
	}
	return nil
}

// scopeMiddleware 把资源作用域写入请求 context。
func scopeMiddleware(sc resource.Scope) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := resource.WithScope(c.Request.Context(), sc)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
