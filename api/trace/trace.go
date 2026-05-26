package trace

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// 常量定义
const (
	// 标准HTTP头
	TraceIDHeader = "X-Trace-ID"

	// 长度定义
	TraceIDLength = 32
)

// Trace 链路跟踪上下文
type Trace struct {
	TraceID string
}

// NewTrace 创建新的链路跟踪上下文
func NewTrace() *Trace {
	return &Trace{
		TraceID: GenerateTraceID(),
	}
}

// ExtractFromRequest 从HTTP请求中提取链路跟踪信息
func ExtractFromRequest(r *http.Request) *Trace {
	traceID := r.Header.Get(TraceIDHeader)
	if traceID == "" {
		traceID = GenerateTraceID()
	}

	return &Trace{
		TraceID: traceID,
	}
}

// InjectToRequest 将链路跟踪信息注入到HTTP请求中
func (t *Trace) InjectToRequest(r *http.Request) {
	r.Header.Set(TraceIDHeader, t.TraceID)
}

// InjectToHeaders 将链路跟踪信息注入到HTTP头中
func (t *Trace) InjectToHeaders(headers map[string]string) {
	headers[TraceIDHeader] = t.TraceID
}

// GenerateTraceID 生成唯一的TraceID
func GenerateTraceID() string {
	return generateRandomID(TraceIDLength)
}

// 生成指定长度的随机ID
func generateRandomID(length int) string {
	bytes := make([]byte, length/2)
	_, err := rand.Read(bytes)
	if err != nil {
		// 失败时使用时间戳和随机数兜底
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// ValidateTraceID 验证TraceID格式
func ValidateTraceID(traceID string) bool {
	if len(traceID) != TraceIDLength {
		return false
	}
	_, err := hex.DecodeString(traceID)
	return err == nil
}

// ExtractTraceFromHeaders 从HTTP头中提取链路跟踪信息
func ExtractTraceFromHeaders(headers map[string]string) *Trace {
	traceID := headers[TraceIDHeader]
	if traceID == "" {
		traceID = GenerateTraceID()
	}
	return &Trace{
		TraceID: traceID,
	}
}
