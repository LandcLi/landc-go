package core

import (
	"time"
)

// Response 规范的HTTP响应结构
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	TraceID   string      `json:"trace_id"`
	Latency   int64       `json:"latency"` // 毫秒
	Timestamp time.Time   `json:"timestamp"`
}

// SetLatency 设置延迟（从 time.Duration 转为毫秒）
func (r *Response) SetLatency(d time.Duration) {
	r.Latency = d.Milliseconds()
}
