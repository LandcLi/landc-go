package v1

import "github.com/LandcLi/landc-go/frame/pkg/meta"

// SayHelloRequest 打招呼请求（Meta Tag 声明路由）
type SayHelloRequest struct {
	meta.Meta `path:"/api/hello" method:"POST" description:"打招呼，首次呼叫会写入数据库"`
	Name      string `source:"body" name:"name" binding:"required"`
}

type SayHelloResponse struct {
	Message string `json:"message"`
	ID      int64  `json:"id"`
	Hit     bool   `json:"hit"` // true=缓存命中，false=已入库
}
