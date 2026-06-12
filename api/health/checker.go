// Package health 提供健康检查的通用接口和数据结构。
//
// 本包不依赖任何 Web 框架，可在 Gin / GoFrame / 纯 net/http 项目中使用。
//
// 用法:
//
//	// 注册自定义检查器
//	health.Register(&myDBChecker{})
//
//	// 获取所有检查器，逐个执行探测
//	for _, c := range health.All() {
//	    err := c.Check(ctx)
//	    // ...
//	}
package health

import (
	"context"
	"sync"
)

// Checker 健康检查器接口。
// 任何实现了 Checker 的组件都可被框架的健康检查端点探测。
type Checker interface {
	// Name 返回检查器名称，用于日志和响应标识。
	Name() string

	// Check 执行健康检查，返回 nil 表示正常，返回 error 表示异常。
	Check(ctx context.Context) error
}

// CheckResult 单次检查结果。
type CheckResult struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "up" | "down"
	Error  string `json:"error,omitempty"`
}

// CheckResponse 聚合健康检查响应。
type CheckResponse struct {
	Status string        `json:"status"` // "ok" | "degraded"
	Checks []CheckResult `json:"checks"`
}

// ==================== 全局注册表 ====================

var (
	checkers   []Checker
	checkersMu sync.RWMutex
)

// Register 注册一个或多个健康检查器。
// 可在 init() 中调用。
func Register(c ...Checker) {
	checkersMu.Lock()
	defer checkersMu.Unlock()
	checkers = append(checkers, c...)
}

// All 返回所有已注册检查器的副本。
func All() []Checker {
	checkersMu.RLock()
	defer checkersMu.RUnlock()
	result := make([]Checker, len(checkers))
	copy(result, checkers)
	return result
}
