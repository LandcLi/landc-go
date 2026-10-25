package web

import (
	"errors"
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

// RegisterHandlers 将控制器注册到任意 gin.IRouter（gin.Engine 或 gin.RouterGroup）。
//
// 与 Server.RegisterHandler 的区别：不登记到 Server 的路由表，因此不参与
// Server.Routes() / Controllers() 的文档生成。适用于「库模式」——宿主服务
// 已经有自己的 gin 引擎（例如进程内组合多个业务系统），只想把本系统的控制器
// 挂到指定路由上。
//
// 返回值是该控制器最终生效的路由列表，便于宿主自行记录或生成文档。
func RegisterHandlers(router gin.IRouter, instance interface{}, opts ...RegisterOption) ([]RouteInfo, error) {
	if router == nil {
		return nil, errors.New("web: router must not be nil")
	}
	if instance == nil {
		return nil, errors.New("web: controller instance must not be nil")
	}
	return registerHandlers(router, instance, opts...)
}
