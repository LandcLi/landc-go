package health

import "context"

// Checker 健康检查器接口。
// 框架内置 DB 和 Redis 检查器；用户可自定义并注册自己的检查器。
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
