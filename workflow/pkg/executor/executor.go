package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/LandcLi/landc-go/tools/httpclient"
	"github.com/LandcLi/landc-go/workflow/pkg/model"
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
}

type HTTPExecutor struct {
	client *http.Client
}

func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{
		client: httpclient.New(30*time.Second, httpclient.WithTrace(nil)),
	}
}

func (e *HTTPExecutor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	var cfg HTTPExecutorConfig
	if err := json.Unmarshal(req.Config, &cfg); err != nil {
		return &ExecuteResponse{Success: false, Error: fmt.Sprintf("invalid config: %v", err)}, err
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

func (e *ScriptExecutor) Execute(_ context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	var cfg ScriptExecutorConfig
	if err := json.Unmarshal(req.Config, &cfg); err != nil {
		return &ExecuteResponse{Success: false, Error: fmt.Sprintf("invalid config: %v", err)}, err
	}
	_ = cfg
	return &ExecuteResponse{Success: true, Output: req.Input}, nil
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

func (e *SubWorkflowExecutor) Execute(_ context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	return &ExecuteResponse{Success: true, Output: req.Input}, nil
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
