package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/LandcLi/landc-go/workflow/pkg/model"
)

// ============================================================
// InputNodeExecutor — 入口节点，传递工作流输入
// ============================================================

type InputNodeExecutor struct{}

func NewInputNodeExecutor() *InputNodeExecutor { return &InputNodeExecutor{} }

func (e *InputNodeExecutor) Execute(_ context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	return &ExecuteResponse{Success: true, Output: req.Input}, nil
}

func (e *InputNodeExecutor) Type() string { return string(model.NodeTypeInput) }

// ============================================================
// OutputNodeExecutor — 出口节点，汇聚上游输出为最终结果
// ============================================================

type OutputNodeExecutor struct{}

func NewOutputNodeExecutor() *OutputNodeExecutor { return &OutputNodeExecutor{} }

func (e *OutputNodeExecutor) Execute(_ context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	output := req.Input
	if output == nil {
		output = json.RawMessage(`{}`)
	}
	return &ExecuteResponse{Success: true, Output: output}, nil
}

func (e *OutputNodeExecutor) Type() string { return string(model.NodeTypeOutput) }

// ============================================================
// ConditionNodeExecutor — 条件判断节点
// 增强: 输入为 JSON 时读取语义化字段判断，不仅仅是字符串 truthiness
// ============================================================

type ConditionNodeExecutor struct{}

func NewConditionNodeExecutor() *ConditionNodeExecutor { return &ConditionNodeExecutor{} }

type conditionConfig struct {
	Expression string `json:"expression"`
	Field      string `json:"field"`       // JSON 输入的哪个字段用于判断
	Operator   string `json:"operator"`    // equals / not_equals / contains / exists
	Expected   string `json:"expected"`    // 期望值
}

func (e *ConditionNodeExecutor) Execute(_ context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	inputRaw := string(req.Input)

	// 解析节点配置
	var cfg conditionConfig
	if req.Config != nil {
		json.Unmarshal(req.Config, &cfg)
	}

	var result bool

	if cfg.Field != "" {
		// 模式 A: 从 JSON 输入中提取指定字段进行判断
		result = evaluateJSONField(inputRaw, &cfg)
	} else if cfg.Expression != "" {
		// 模式 B: 预留表达式引擎
		result = inputRaw != ""
		_ = cfg.Expression
	} else {
		// 模式 C: 判断整个输入
		// 先尝试解析 JSON，如果 input 本身就是 JSON 结构，根据语义判断
		result = evaluateTruth(inputRaw)
	}

	output := "false"
	if result {
		output = "true"
	}
	return &ExecuteResponse{Success: true, Output: json.RawMessage(`"` + output + `"`)}, nil
}

func (e *ConditionNodeExecutor) Type() string { return string(model.NodeTypeCondition) }

// evaluateJSONField 从 JSON 中提取字段做语义匹配
func evaluateJSONField(inputRaw string, cfg *conditionConfig) bool {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(inputRaw), &data); err != nil {
		return false
	}

	fieldVal, ok := data[cfg.Field]
	if !ok {
		return cfg.Operator == "not_exists"
	}

	fieldStr := fmt.Sprintf("%v", fieldVal)

	switch cfg.Operator {
	case "equals", "":
		return fieldStr == cfg.Expected
	case "not_equals":
		return fieldStr != cfg.Expected
	case "contains":
		return strings.Contains(fieldStr, cfg.Expected)
	case "exists":
		return true
	case "not_exists":
		return false
	default:
		return fieldStr != "" && fieldStr != "false" && fieldStr != "0"
	}
}

// evaluateTruth 语义化 truthiness 判断
func evaluateTruth(inputRaw string) bool {
	// 尝试 JSON 解析
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(inputRaw), &data); err == nil {
		// JSON 对象: 检查是否有语义字段表示"成功"
		if v, ok := data["success"]; ok {
			if b, ok := v.(bool); ok {
				return b
			}
			if s, ok := v.(string); ok {
				return s == "true" || s == "ok" || s == "success"
			}
		}
		if v, ok := data["status"]; ok {
			if s, ok := v.(string); ok {
				return s == "ok" || s == "success" || s == "completed"
			}
		}
		if v, ok := data["code"]; ok {
			if f, ok := v.(float64); ok {
				return f >= 200 && f < 300
			}
		}
		// JSON 对象非空即为 true（有一个字段就算）
		return len(data) > 0
	}

	// 纯文本: 非空且不是否定值
	cleaned := strings.Trim(strings.TrimSpace(inputRaw), `"`)
	if cleaned == "" {
		return false
	}
	negatives := []string{"false", "0", "null", "undefined", "no", "fail", "error", "[]", "{}"}
	for _, n := range negatives {
		if cleaned == n {
			return false
		}
	}
	return true
}

// ============================================================
// SwitchNodeExecutor — 多路分支节点（核心增强）
// 场景: 根据文件格式路由到不同解析节点
//
// Config 示例:
//
//	{
//	  "input_field": "filetype",        // 从 JSON 输入中提取哪个字段
//	  "extract_extension": true,        // 从文件名提取扩展名（当 input_field 是文件名时）
//	  "cases": [
//	    {"value": "pdf",  "output": "pdf-branch"},
//	    {"value": "docx", "output": "docx-branch"},
//	    {"value": "csv",  "output": "csv-branch"}
//	  ],
//	  "default_output": "unknown"
//	}
// ============================================================

type SwitchNodeExecutor struct{}

func NewSwitchNodeExecutor() *SwitchNodeExecutor { return &SwitchNodeExecutor{} }

type switchConfig struct {
	Cases []struct {
		Value  string `json:"value"`
		Output string `json:"output"`
	} `json:"cases"`
	DefaultOutput    string `json:"default_output"`
	InputField       string `json:"input_field"`        // 要匹配的 JSON 字段名
	ExtractExtension bool   `json:"extract_extension"`  // 从文件名提取扩展名
	CaseInsensitive  bool   `json:"case_insensitive"`   // 大小写不敏感
}

func (e *SwitchNodeExecutor) Execute(_ context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	var cfg switchConfig
	if req.Config != nil {
		json.Unmarshal(req.Config, &cfg)
	}

	// 1. 提取要匹配的值
	matchValue := extractMatchValue(req.Input, &cfg)

	// 2. 大小写统一
	if cfg.CaseInsensitive {
		matchValue = strings.ToLower(matchValue)
	}

	// 3. 遍历 cases 匹配
	matched := cfg.DefaultOutput
	if matched == "" {
		matched = "default"
	}
	for _, c := range cfg.Cases {
		cmp := c.Value
		if cfg.CaseInsensitive {
			cmp = strings.ToLower(cmp)
		}
		if matchValue == cmp {
			matched = c.Output
			break
		}
	}

	return &ExecuteResponse{Success: true, Output: json.RawMessage(`"` + matched + `"`)}, nil
}

func (e *SwitchNodeExecutor) Type() string { return string(model.NodeTypeSwitch) }

// extractMatchValue 从输入中提取要匹配的值
func extractMatchValue(input json.RawMessage, cfg *switchConfig) string {
	raw := strings.Trim(string(input), `"`)

	if cfg.InputField != "" {
		// 从 JSON 对象中提取指定字段
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &data); err == nil {
			if val, ok := data[cfg.InputField]; ok {
				strVal := fmt.Sprintf("%v", val)
				if cfg.ExtractExtension {
					return extractExtension(strVal)
				}
				return strVal
			}
		}
		// JSON 解析失败但配置了 field, 尝试在原始字符串中找
		return raw
	}

	if cfg.ExtractExtension {
		return extractExtension(raw)
	}

	return raw
}

// extractExtension 从文件名中提取扩展名（不含点号）
func extractExtension(filename string) string {
	ext := filepath.Ext(filename)
	return strings.TrimPrefix(strings.ToLower(ext), ".")
}

// ============================================================
// HumanInputExecutor — 人类输入节点
// 暂停工作流，等待外部通过 Engine.ProvideHumanInput() 注入输入
// ============================================================

type HumanInputExecutor struct {
	waitingInputs func(execID, nodeID string) <-chan string
}

func NewHumanInputExecutor(waitingInputs func(execID, nodeID string) <-chan string) *HumanInputExecutor {
	return &HumanInputExecutor{waitingInputs: waitingInputs}
}

func (e *HumanInputExecutor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	if e.waitingInputs == nil {
		return &ExecuteResponse{Success: false, Error: "human input not supported: no input channel"}, nil
	}

	inputCh := e.waitingInputs(req.ExecutionID, req.NodeID)
	if inputCh == nil {
		return &ExecuteResponse{Success: false, Error: "human input not available"}, nil
	}

	select {
	case <-ctx.Done():
		return &ExecuteResponse{Success: false, Error: "cancelled"}, ctx.Err()
	case input := <-inputCh:
		return &ExecuteResponse{Success: true, Output: json.RawMessage(input)}, nil
	}
}

func (e *HumanInputExecutor) Type() string { return string(model.NodeTypeHumanInput) }
