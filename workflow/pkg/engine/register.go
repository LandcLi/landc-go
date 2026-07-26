package engine

import (
	"github.com/LandcLi/landc-go/frame/pkg/di"
)

const engineName = "workflow.engine"

// RegisterEngine 通过 DI 注册引擎实例
func RegisterEngine(e *Engine) {
	di.Provide[*Engine](engineName, e)
}

// GetEngine 从 DI 获取引擎实例
func GetEngine() *Engine {
	return di.Require[*Engine](engineName)
}
