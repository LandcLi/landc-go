package web

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
)

// registerHandlers 注册控制器的全部导出方法为 HTTP 路由，返回最终生效路由列表。
//
// 路由优先级：注册选项（WithPrefix / WithMethodPath / WithMethodHTTPMethod）
// > 编译期 meta.Meta 标签。不提供选项时与旧行为一致。
func registerHandlers(router gin.IRouter, instance interface{}, opts ...RegisterOption) ([]RouteInfo, error) {
	options := &RegisterOptions{}
	for _, opt := range opts {
		opt(options)
	}

	instanceValue := reflect.ValueOf(instance)
	instanceType := reflect.TypeOf(instance)

	groupPath := getGroupPath(instance)
	if options.Prefix != "" {
		groupPath = joinPath(options.Prefix, groupPath)
	}

	if groupPath != "" {
		router = router.Group(groupPath)
	}

	// 控制器级中间件：包一层空 group，使其先于方法级中间件与业务 handler 执行
	if len(options.controllerMiddlewares) > 0 {
		router = router.Group("", options.controllerMiddlewares...)
	}

	routes := make([]RouteInfo, 0, instanceType.NumMethod())

	for i := 0; i < instanceType.NumMethod(); i++ {
		method := instanceType.Method(i)
		if !isExported(method.Name) {
			continue
		}

		meta, err := parseMethodMeta(method)
		if err != nil {
			return nil, fmt.Errorf("failed to parse method %s: %w", method.Name, err)
		}
		if meta.HTTPMethod == "" {
			continue
		}

		// 运行时覆盖优先于编译期标签
		path := meta.Path
		if p, ok := options.methodPaths[method.Name]; ok && p != "" {
			path = p
		}
		httpMethod := meta.HTTPMethod
		if m, ok := options.methodHTTPMethods[method.Name]; ok && m != "" {
			httpMethod = m
		}

		handler := createHandler(instanceValue, method)
		mws := options.methodMiddlewares[method.Name]
		handlers := make([]gin.HandlerFunc, 0, len(mws)+1)
		handlers = append(handlers, mws...)
		handlers = append(handlers, handler)

		switch strings.ToUpper(httpMethod) {
		case "GET":
			router.GET(path, handlers...)
		case "POST":
			router.POST(path, handlers...)
		case "PUT":
			router.PUT(path, handlers...)
		case "DELETE":
			router.DELETE(path, handlers...)
		case "PATCH":
			router.PATCH(path, handlers...)
		case "OPTIONS":
			router.OPTIONS(path, handlers...)
		case "HEAD":
			router.HEAD(path, handlers...)
		default:
			return nil, fmt.Errorf("unsupported HTTP method: %s", httpMethod)
		}

		routes = append(routes, RouteInfo{
			Method:      strings.ToUpper(httpMethod),
			Path:        joinPath(groupPath, path),
			Description: meta.Description,
			HandlerName: method.Name,
		})
	}

	return routes, nil
}
