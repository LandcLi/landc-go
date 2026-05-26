package core

import (
	"net"
	"net/http"
	"strings"

	"github.com/LandcLi/landc-go/api/trace"
)

// HeaderExtractor 请求头提取接口
type HeaderExtractor interface {
	ExtractHeaders(r *http.Request) map[string]string
}

// DefaultHeaderExtractor 默认的请求头提取器
type DefaultHeaderExtractor struct {
	customExtractors map[string]func(r *http.Request) string
	trustedProxies   []string // 可信代理 CIDR 列表
}

// NewDefaultHeaderExtractor 创建默认的请求头提取器
func NewDefaultHeaderExtractor() *DefaultHeaderExtractor {
	return &DefaultHeaderExtractor{
		customExtractors: make(map[string]func(r *http.Request) string),
		trustedProxies:   []string{"127.0.0.1/32", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
	}
}

// WithCustomExtractor 添加自定义提取器
func (e *DefaultHeaderExtractor) WithCustomExtractor(header string, extractor func(r *http.Request) string) *DefaultHeaderExtractor {
	e.customExtractors[header] = extractor
	return e
}

// WithTrustedProxies 设置可信代理列表（CIDR 格式）
func (e *DefaultHeaderExtractor) WithTrustedProxies(proxies []string) *DefaultHeaderExtractor {
	e.trustedProxies = proxies
	return e
}

// ExtractHeaders 提取请求头
func (e *DefaultHeaderExtractor) ExtractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)

	headers["X-Request-ID"] = e.extractRequestID(r)
	headers["X-Client-IP"] = e.extractClientIP(r)
	headers["X-User-Agent"] = e.extractUserAgent(r)
	headers["X-Device-Fingerprint"] = e.extractDeviceFingerprint(r)

	for header, extractor := range e.customExtractors {
		if value := extractor(r); value != "" {
			headers[header] = value
		}
	}

	return headers
}

func (e *DefaultHeaderExtractor) extractRequestID(r *http.Request) string {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = r.Header.Get("X-Trace-ID")
	}
	if requestID == "" {
		requestID = trace.GenerateTraceID()
	}
	return requestID
}

func (e *DefaultHeaderExtractor) extractClientIP(r *http.Request) string {
	// 只有来自可信代理的请求才信任 X-Forwarded-For
	remoteIP := e.getRemoteIP(r)

	if e.isTrustedProxy(remoteIP) {
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			ips := strings.Split(ip, ",")
			// 从右向左找到第一个非可信代理 IP
			for i := len(ips) - 1; i >= 0; i-- {
				candidate := strings.TrimSpace(ips[i])
				if !e.isTrustedProxy(candidate) {
					return candidate
				}
			}
			// 所有都是可信代理，返回最左边的
			if len(ips) > 0 {
				return strings.TrimSpace(ips[0])
			}
		}

		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return ip
		}
	}

	return remoteIP
}

func (e *DefaultHeaderExtractor) getRemoteIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (e *DefaultHeaderExtractor) isTrustedProxy(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range e.trustedProxies {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return true
		}
	}
	return false
}

func (e *DefaultHeaderExtractor) extractUserAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
}

func (e *DefaultHeaderExtractor) extractDeviceFingerprint(r *http.Request) string {
	fingerprint := r.Header.Get("X-Device-Fingerprint")
	if fingerprint == "" {
		fingerprint = r.Header.Get("X-Device-ID")
	}
	if fingerprint == "" {
		fingerprint = r.Header.Get("X-Device-Token")
	}
	if fingerprint == "" {
		fingerprint = r.Header.Get("X-Session-ID")
	}
	return fingerprint
}

// HeaderInjector 响应头注入接口
type HeaderInjector interface {
	InjectHeaders(w http.ResponseWriter, headers map[string]string)
}

// DefaultHeaderInjector 默认的响应头注入器
type DefaultHeaderInjector struct {
	customHeaders map[string]string
}

// NewDefaultHeaderInjector 创建默认的响应头注入器
func NewDefaultHeaderInjector() *DefaultHeaderInjector {
	return &DefaultHeaderInjector{
		customHeaders: make(map[string]string),
	}
}

// WithCustomHeader 添加自定义响应头
func (i *DefaultHeaderInjector) WithCustomHeader(header, value string) *DefaultHeaderInjector {
	i.customHeaders[header] = value
	return i
}

// InjectHeaders 注入响应头
func (i *DefaultHeaderInjector) InjectHeaders(w http.ResponseWriter, headers map[string]string) {
	for key, value := range headers {
		if value != "" {
			w.Header().Set(key, value)
		}
	}

	for key, value := range i.customHeaders {
		if value != "" {
			w.Header().Set(key, value)
		}
	}
}

// HeaderProcessor 请求头处理器
type HeaderProcessor struct {
	extractor HeaderExtractor
	injector  HeaderInjector
}

// NewHeaderProcessor 创建请求头处理器
func NewHeaderProcessor() *HeaderProcessor {
	return &HeaderProcessor{
		extractor: NewDefaultHeaderExtractor(),
		injector:  NewDefaultHeaderInjector(),
	}
}

// WithExtractor 设置提取器
func (p *HeaderProcessor) WithExtractor(extractor HeaderExtractor) *HeaderProcessor {
	p.extractor = extractor
	return p
}

// WithInjector 设置注入器
func (p *HeaderProcessor) WithInjector(injector HeaderInjector) *HeaderProcessor {
	p.injector = injector
	return p
}

// WithCustomExtractor 添加自定义提取器
func (p *HeaderProcessor) WithCustomExtractor(header string, extractor func(r *http.Request) string) *HeaderProcessor {
	if defaultExtractor, ok := p.extractor.(*DefaultHeaderExtractor); ok {
		defaultExtractor.WithCustomExtractor(header, extractor)
	}
	return p
}

// WithCustomHeader 添加自定义响应头
func (p *HeaderProcessor) WithCustomHeader(header, value string) *HeaderProcessor {
	if defaultInjector, ok := p.injector.(*DefaultHeaderInjector); ok {
		defaultInjector.WithCustomHeader(header, value)
	}
	return p
}

// WithTrustedProxies 设置可信代理
func (p *HeaderProcessor) WithTrustedProxies(proxies []string) *HeaderProcessor {
	if defaultExtractor, ok := p.extractor.(*DefaultHeaderExtractor); ok {
		defaultExtractor.WithTrustedProxies(proxies)
	}
	return p
}

// Process 处理请求头
func (p *HeaderProcessor) Process(r *http.Request, w http.ResponseWriter) map[string]string {
	headers := p.extractor.ExtractHeaders(r)
	p.injector.InjectHeaders(w, headers)
	return headers
}
