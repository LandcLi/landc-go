package web

import "github.com/LandcLi/landc-go/api/core"

// statusRange 描述一段错误码区间映射到的 HTTP 状态码。
type statusRange struct {
	from, to core.ErrorCode
	status   int
}

// statusRanges 预定义错误码区间 → HTTP 状态码。
var statusRanges = []statusRange{
	{40100, 40199, 401},
	{40300, 40399, 403},
	{40400, 40499, 404},
	{40500, 40599, 405},
	{40900, 40999, 409},
	{42200, 42299, 422},
	{50100, 50199, 501},
	{50200, 50299, 502},
	{50300, 50399, 503},
	{50400, 50499, 504},
}

// httpStatusFromCode 将业务错误码映射为 HTTP 状态码。
//
// 规则：
//   - 4xxxx 客户端错误 → 4xx（预定义区间按百位细分，其余 → 400）
//   - 5xxxx 服务端错误 → 5xx（预定义区间按百位细分，其余 → 500）
//   - 6xxxx-99999 自定义错误 → 400（视为客户端可纠正的业务错误）
func httpStatusFromCode(code core.ErrorCode) int {
	for _, r := range statusRanges {
		if code >= r.from && code <= r.to {
			return r.status
		}
	}
	switch {
	case code >= 40000 && code < 50000:
		return 400
	case code >= 50000 && code < 60000:
		return 500
	default:
		return 400
	}
}
