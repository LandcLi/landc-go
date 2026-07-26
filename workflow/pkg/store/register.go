package store

import (
	"github.com/LandcLi/landc-go/frame/pkg/di"
)

// 通过 DI 容器注册的 store 实例名称
const storeName = "workflow.store"

// RegisterStore 通过 DI 注册 store 实例
func RegisterStore(s Store) {
	di.Provide[Store](storeName, s)
}

// GetStore 从 DI 获取 store 实例
func GetStore() Store {
	return di.Require[Store](storeName)
}
