package health

import "sync"

var (
	checkers   []Checker
	checkersMu sync.RWMutex
)

// RegisterChecker 注册自定义健康检查器。
// 用户可在 init() 或程序启动时调用。
func RegisterChecker(checker Checker) {
	checkersMu.Lock()
	defer checkersMu.Unlock()
	checkers = append(checkers, checker)
}

// GlobalCheckers 返回全局已注册的所有检查器（含内置的和用户自定义的）。
// 返回的是切片的副本，调用方可以安全使用。
func GlobalCheckers() []Checker {
	checkersMu.RLock()
	defer checkersMu.RUnlock()
	result := make([]Checker, len(checkers))
	copy(result, checkers)
	return result
}

// RegisterDefaultCheckers 注册框架内置的 DB 和 Redis 检查器。
// 在 server 初始化时自动调用。
func RegisterDefaultCheckers(dbEnabled, redisEnabled bool) {
	checkersMu.Lock()
	defer checkersMu.Unlock()

	if dbEnabled {
		checkers = append(checkers, &dbChecker{})
	}
	if redisEnabled {
		checkers = append(checkers, &redisChecker{})
	}
}
