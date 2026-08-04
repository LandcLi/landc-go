package web

import (
	"strings"
	"sync"
)

// RouteInfo 描述一条最终生效的路由（含前缀与运行时覆盖），供路由查询使用。
type RouteInfo struct {
	Method      string // HTTP 方法：GET / POST / PUT / DELETE / ...
	Path        string // 完整路径：/api/v2/user/login
	Description string // 编译期 meta description（若有）
	HandlerName string // 控制器方法名：Login
}

// routeRegistry 收集已注册的最终路由，供 Routes() 查询（只读、无副作用）。
type routeRegistry struct {
	mu     sync.RWMutex
	routes []RouteInfo
}

func (r *routeRegistry) add(routes []RouteInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes = append(r.routes, routes...)
}

func (r *routeRegistry) all() []RouteInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]RouteInfo, len(r.routes))
	copy(out, r.routes)
	return out
}

// joinPath 拼接 group 与相对路径，保证斜杠语义正确。
func joinPath(group, path string) string {
	group = strings.TrimRight(group, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return group + path
}
