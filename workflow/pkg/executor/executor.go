package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LandcLi/landc-go/tools/httpclient"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
	"github.com/dop251/goja"
)

// ============================================================
// ExecuteRequest / ExecuteResponse
// ============================================================

type ExecuteRequest struct {
	NodeID      string          `json:"node_id"`
	NodeName    string          `json:"node_name"`
	NodeType    string          `json:"node_type"`
	Config      json.RawMessage `json:"config"`
	Input       json.RawMessage `json:"input"`
	InputMap    json.RawMessage `json:"input_mapping"`
	OutputMap   json.RawMessage `json:"output_mapping"`
	RetryCount  int             `json:"retry_count"`
	MaxRetries  int             `json:"max_retries"`
	AttemptID   string          `json:"attempt_id"`
	ExecutionID string          `json:"execution_id"`
	TriggerID   string          `json:"trigger_id"`
	WorkflowID  string          `json:"workflow_id"`
	NodeRefs    []NodeRef       `json:"node_refs"`
	EdgeRefs    []EdgeRef       `json:"edge_refs"`
}

type NodeRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type EdgeRef struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
}

type ExecuteResponse struct {
	Success  bool              `json:"success"`
	Output   json.RawMessage   `json:"output,omitempty"`
	Error    string            `json:"error,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ============================================================
// NodeExecutor 接口（REQ-003: 增加 Schema()）
// ============================================================

type NodeExecutor interface {
	Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)
	Type() string
	Schema() json.RawMessage
}

// ============================================================
// 重试策略
// ============================================================

type RetryStrategy interface {
	NextDelay(attempt int, baseDelay time.Duration, maxDelay time.Duration) time.Duration
	MaxAttempts() int
}

type LinearRetry struct {
	MaxRetries int
	Delay      time.Duration
	MaxDelay   time.Duration
}

func (r *LinearRetry) NextDelay(_ int, _ time.Duration, _ time.Duration) time.Duration {
	return r.Delay
}
func (r *LinearRetry) MaxAttempts() int { return r.MaxRetries + 1 }

type ExponentialRetry struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

func (r *ExponentialRetry) NextDelay(attempt int, baseDelay time.Duration, _ time.Duration) time.Duration {
	if r.BaseDelay > 0 {
		baseDelay = r.BaseDelay
	}
	delay := float64(baseDelay) * math.Pow(2, float64(attempt-1))
	jitter := rand.Float64() * 0.5 * delay
	total := time.Duration(delay + jitter)
	maxDelay := r.MaxDelay
	if maxDelay == 0 {
		maxDelay = 5 * time.Minute
	}
	if total > maxDelay {
		total = maxDelay
	}
	return total
}
func (r *ExponentialRetry) MaxAttempts() int { return r.MaxRetries + 1 }

func NewRetryStrategy(mode string, maxRetries int, delaySec, maxDelaySec int64) RetryStrategy {
	if delaySec <= 0 {
		delaySec = 1
	}
	switch mode {
	case "EXPONENTIAL":
		return &ExponentialRetry{
			MaxRetries: maxRetries,
			BaseDelay:  time.Duration(delaySec) * time.Second,
			MaxDelay:   time.Duration(maxDelaySec) * time.Second,
		}
	default:
		return &LinearRetry{
			MaxRetries: maxRetries,
			Delay:      time.Duration(delaySec) * time.Second,
			MaxDelay:   time.Duration(maxDelaySec) * time.Second,
		}
	}
}

// ============================================================
// RetryableExecutor
// ============================================================

type RetryableExecutor struct {
	inner     NodeExecutor
	strategy  RetryStrategy
	reentrant bool
}

func NewRetryableExecutor(inner NodeExecutor, strategy RetryStrategy, reentrant bool) *RetryableExecutor {
	return &RetryableExecutor{inner: inner, strategy: strategy, reentrant: reentrant}
}

func (e *RetryableExecutor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	maxAttempts := e.strategy.MaxAttempts()
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	var lastErr error
	attempt := req.RetryCount + 1
	for ; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return &ExecuteResponse{Success: false, Error: ctx.Err().Error()}, ctx.Err()
		default:
		}
		if attempt > 1 {
			delay := e.strategy.NextDelay(attempt, 0, 0)
			select {
			case <-ctx.Done():
				return &ExecuteResponse{Success: false, Error: ctx.Err().Error()}, ctx.Err()
			case <-time.After(delay):
			}
		}
		resp, err := e.inner.Execute(ctx, req)
		if err == nil && resp.Success {
			return resp, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("%s", resp.Error)
		}
	}
	return &ExecuteResponse{
		Success: false,
		Error:   fmt.Sprintf("all %d attempts failed: %v", maxAttempts, lastErr),
	}, lastErr
}

func (e *RetryableExecutor) Type() string { return e.inner.Type() }
func (e *RetryableExecutor) Schema() json.RawMessage { return e.inner.Schema() }

// ============================================================
// HTTPExecutor
// ============================================================

type HTTPExecutorConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout int               `json:"timeout"`
	// AllowPrivateNetwork 允许访问内网/保留地址（默认拒绝，SSRF 防护）
	// 仅在信任工作流定义来源时开启
	AllowPrivateNetwork bool `json:"allow_private_network,omitempty"`
}

type HTTPExecutor struct {
	client *http.Client
}

func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{
		client: httpclient.New(30*time.Second, httpclient.WithTrace(nil)),
	}
}

// isBlockedAddress 判断目标地址是否为内网/保留地址（SSRF 防护）
// 无法解析的主机名也保守拒绝
func isBlockedAddress(host string) bool {
	// 去除可能携带的端口
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]") // IPv6 字面量

	ips, err := net.LookupIP(host)
	if err != nil {
		return true // DNS 解析失败：保守拒绝
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() ||
			ip.IsMulticast() || ip.IsInterfaceLocalMulticast() {
			return true
		}
		// CGNAT 段 100.64.0.0/10、保留段 192.0.0.0/24 等
		if ip4 := ip.To4(); ip4 != nil {
			first := ip4[0]
			second := ip4[1]
			if first == 100 && second >= 64 && second <= 127 {
				return true
			}
			if first == 192 && second == 0 {
				return true
			}
			if first == 0 {
				return true
			}
		}
	}
	return false
}

func (e *HTTPExecutor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	var cfg HTTPExecutorConfig
	if err := json.Unmarshal(req.Config, &cfg); err != nil {
		return &ExecuteResponse{Success: false, Error: fmt.Sprintf("invalid config: %v", err)}, err
	}

	// SSRF 防护：默认拒绝访问内网/保留地址，除非显式开启 AllowPrivateNetwork
	if !cfg.AllowPrivateNetwork {
		parsedURL, err := url.Parse(cfg.URL)
		if err != nil {
			return &ExecuteResponse{Success: false, Error: fmt.Sprintf("invalid url: %v", err)}, err
		}
		if isBlockedAddress(parsedURL.Host) {
			return &ExecuteResponse{Success: false, Error: "SSRF protection: target address is blocked (internal/private network)"}, nil
		}
	}

	method := cfg.Method
	if method == "" {
		method = http.MethodPost
	}
	var bodyReader io.Reader
	if req.Input != nil {
		bodyReader = bytes.NewReader(req.Input)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, cfg.URL, bodyReader)
	if err != nil {
		return &ExecuteResponse{Success: false, Error: fmt.Sprintf("create request failed: %v", err)}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.Headers {
		httpReq.Header.Set(k, v)
	}
	httpResp, err := e.client.Do(httpReq)
	if err != nil {
		return &ExecuteResponse{Success: false, Error: fmt.Sprintf("http request failed: %v", err)}, err
	}
	defer httpResp.Body.Close()
	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode >= 400 {
		return &ExecuteResponse{
			Success: false,
			Error:   fmt.Sprintf("http status %d: %s", httpResp.StatusCode, string(respBody)),
		}, nil
	}
	return &ExecuteResponse{Success: true, Output: respBody}, nil
}

func (e *HTTPExecutor) Type() string { return string(model.NodeTypeHttp) }
func (e *HTTPExecutor) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"status_code":{"type":"number"},"body":{"type":"string"},"headers":{"type":"object"}}}`)
}

// ============================================================
// ScriptExecutor
// ============================================================

type ScriptExecutorConfig struct {
	Lang    string `json:"lang"`
	Script  string `json:"script"`
	Timeout int    `json:"timeout"`
}

type ScriptExecutor struct{}

func NewScriptExecutor() *ScriptExecutor { return &ScriptExecutor{} }

// Execute 执行脚本节点
// 支持语言：js/javascript（基于 goja 纯 Go 引擎，无 CGO 依赖）
// 脚本内可通过 input（解析后的 JSON 对象）与 inputRaw（原始 JSON 字符串）访问上游输入；
// 脚本的最后一个表达式/返回值为节点输出（序列化为 JSON）。
func (e *ScriptExecutor) Execute(_ context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	var cfg ScriptExecutorConfig
	if err := json.Unmarshal(req.Config, &cfg); err != nil {
		return &ExecuteResponse{Success: false, Error: fmt.Sprintf("invalid config: %v", err)}, err
	}

	lang := strings.ToLower(strings.TrimSpace(cfg.Lang))
	if lang == "" {
		lang = "js"
	}
	if lang != "js" && lang != "javascript" {
		return &ExecuteResponse{
			Success: false,
			Error:   fmt.Sprintf("unsupported script language %q: only js/javascript is supported", cfg.Lang),
		}, nil
	}
	if strings.TrimSpace(cfg.Script) == "" {
		return &ExecuteResponse{Success: false, Error: "script is empty"}, nil
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 // 默认 30s，防死循环
	}

	vm := goja.New()

	// 注入上游输入
	if req.Input != nil {
		var input any
		if err := json.Unmarshal(req.Input, &input); err == nil {
			_ = vm.Set("input", input)
		}
		_ = vm.Set("inputRaw", string(req.Input))
	}

	// 超时中断，防止脚本死循环
	timer := time.AfterFunc(time.Duration(timeout)*time.Second, func() {
		vm.Interrupt("script execution timeout")
	})
	defer timer.Stop()

	result, err := vm.RunString(cfg.Script)
	if err != nil {
		return &ExecuteResponse{Success: false, Error: fmt.Sprintf("script error: %v", err)}, nil
	}

	exported := result.Export()
	out, marshalErr := json.Marshal(exported)
	if marshalErr != nil {
		// 非 JSON 可序列化值（如 function/undefined）回退为字符串
		out = []byte(fmt.Sprintf("%v", exported))
	}
	return &ExecuteResponse{Success: true, Output: out}, nil
}

func (e *ScriptExecutor) Type() string { return string(model.NodeTypeScript) }
func (e *ScriptExecutor) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"result":{"type":"string","description":"脚本执行结果"},"logs":{"type":"string"}}}`)
}

// ============================================================
// SubWorkflowExecutor
// ============================================================

type SubWorkflowExecutor struct{}

func NewSubWorkflowExecutor() *SubWorkflowExecutor { return &SubWorkflowExecutor{} }

// Execute 子工作流执行
// 说明：子工作流调用需要引擎级调度上下文（store + engine 执行环境），
// 本 executor 不提供该能力，避免产生循环依赖。如需子工作流，
// 请将子流程在工作流定义中内联展开，或通过自定义执行器接入。
func (e *SubWorkflowExecutor) Execute(_ context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	return &ExecuteResponse{
		Success: false,
		Error:   "sub_workflow executor is not implemented in this version; expand the sub-workflow inline or use a custom executor",
	}, nil
}

func (e *SubWorkflowExecutor) Type() string { return string(model.NodeTypeSubWorkflow) }
func (e *SubWorkflowExecutor) Schema() json.RawMessage { return nil }

// ============================================================
// DelayExecutor
// ============================================================

type DelayExecutorConfig struct {
	Duration int `json:"duration"`
}

type DelayExecutor struct{}

func NewDelayExecutor() *DelayExecutor { return &DelayExecutor{} }

func (e *DelayExecutor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	var cfg DelayExecutorConfig
	if err := json.Unmarshal(req.Config, &cfg); err != nil {
		return &ExecuteResponse{Success: false, Error: fmt.Sprintf("invalid config: %v", err)}, err
	}
	select {
	case <-ctx.Done():
		return &ExecuteResponse{Success: false, Error: "cancelled"}, ctx.Err()
	case <-time.After(time.Duration(cfg.Duration) * time.Second):
	}
	return &ExecuteResponse{Success: true, Output: req.Input}, nil
}

func (e *DelayExecutor) Type() string { return string(model.NodeTypeDelay) }
func (e *DelayExecutor) Schema() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"output":{"type":"string","description":"原样传入"}}}`) }
