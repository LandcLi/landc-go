package web

import "github.com/gin-gonic/gin"

// RegisterOptions 携带 RegisterHandler 的声明式注册选项。
//
// 路由最终值 = 注册选项（若指定）> 编译期 meta.Meta 标签（默认）。
// 不提供任何选项时，行为与仅靠编译期标签完全一致。
type RegisterOptions struct {
	Prefix                string
	methodPaths           map[string]string
	methodHTTPMethods     map[string]string
	methodMiddlewares     map[string][]gin.HandlerFunc
	controllerMiddlewares []gin.HandlerFunc
}

// RegisterOption 是函数式注册选项。
type RegisterOption func(*RegisterOptions)

// WithPrefix 为控制器所有路由添加统一前缀，位于编译期 group path 之前。
func WithPrefix(prefix string) RegisterOption {
	return func(o *RegisterOptions) { o.Prefix = prefix }
}

// WithMethodPath 覆盖单个方法的 URL 路径（优先于编译期 meta path）。
func WithMethodPath(method, path string) RegisterOption {
	return func(o *RegisterOptions) {
		if o.methodPaths == nil {
			o.methodPaths = make(map[string]string)
		}
		o.methodPaths[method] = path
	}
}

// WithMethodHTTPMethod 覆盖单个方法的 HTTP 方法（优先于编译期 meta method）。
func WithMethodHTTPMethod(method, httpMethod string) RegisterOption {
	return func(o *RegisterOptions) {
		if o.methodHTTPMethods == nil {
			o.methodHTTPMethods = make(map[string]string)
		}
		o.methodHTTPMethods[method] = httpMethod
	}
}

// WithMethodMiddleware 为单个方法挂载中间件。
// 可多次调用；多个中间件按声明顺序执行（先声明的先执行，最后执行业务 handler）。
func WithMethodMiddleware(method string, mw ...gin.HandlerFunc) RegisterOption {
	return func(o *RegisterOptions) {
		if o.methodMiddlewares == nil {
			o.methodMiddlewares = make(map[string][]gin.HandlerFunc)
		}
		o.methodMiddlewares[method] = append(o.methodMiddlewares[method], mw...)
	}
}

// WithControllerMiddleware 为控制器下所有方法挂载中间件。
// 在方法级中间件之前执行。
func WithControllerMiddleware(mw ...gin.HandlerFunc) RegisterOption {
	return func(o *RegisterOptions) {
		o.controllerMiddlewares = append(o.controllerMiddlewares, mw...)
	}
}
