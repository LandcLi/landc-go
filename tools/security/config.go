// Package security 提供安全响应头的配置结构体和常量。
//
// 本包不含任何 Web 框架绑定，可被任何 Go 项目使用。
//
// 典型用法（配合 Gin）:
//
//	import ginmw "github.com/LandcLi/landc-go/api/middleware/gin"
//	r.Use(ginmw.SecurityHeaders(security.DefaultHeadersConfig))
package security

// HeadersConfig 安全响应头配置（与任何 Web 框架无关）。
type HeadersConfig struct {
	// HSTSMaxAge Strict-Transport-Security 的 max-age（秒）。
	// 0 表示不设置 HSTS。
	HSTSMaxAge int

	// FrameDeny 设置 X-Frame-Options: DENY。
	FrameDeny bool

	// ContentTypeNoSniff 设置 X-Content-Type-Options: nosniff。
	ContentTypeNoSniff bool

	// CSP Content-Security-Policy 值。空字符串表示不设置。
	CSP string

	// ReferrerPolicy Referrer-Policy 值。空字符串表示不设置。
	ReferrerPolicy string
}

// DefaultHeadersConfig 推荐的默认安全响应头配置。
var DefaultHeadersConfig = HeadersConfig{
	HSTSMaxAge:         31536000, // 1 year
	FrameDeny:          true,
	ContentTypeNoSniff: true,
	CSP:                "",
	ReferrerPolicy:     "strict-origin-when-cross-origin",
}

// HSTSHeader 返回 Strict-Transport-Security 头的值。
func (c HeadersConfig) HSTSHeader() string {
	if c.HSTSMaxAge <= 0 {
		return ""
	}
	return "max-age=" + itoa(c.HSTSMaxAge)
}

// XFrameOptionsHeader 返回 X-Frame-Options 的值。
func (c HeadersConfig) XFrameOptionsHeader() string {
	if !c.FrameDeny {
		return ""
	}
	return "DENY"
}

// XContentTypeOptionsHeader 返回 X-Content-Type-Options 的值。
func (c HeadersConfig) XContentTypeOptionsHeader() string {
	if !c.ContentTypeNoSniff {
		return ""
	}
	return "nosniff"
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
