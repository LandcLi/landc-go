package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// Translator 翻译器
type Translator struct {
	defaultLang string
	messages    map[string]map[string]string // lang -> key -> message
	mu          sync.RWMutex
}

// New 创建翻译器
func New(defaultLang string) *Translator {
	return &Translator{
		defaultLang: defaultLang,
		messages:    make(map[string]map[string]string),
	}
}

// SetDefault 设置默认语言
func (t *Translator) SetDefault(lang string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.defaultLang = lang
}

// Default 获取默认语言
func (t *Translator) Default() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.defaultLang
}

// Register 注册语言消息
func (t *Translator) Register(lang string, messages map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.messages[lang] == nil {
		t.messages[lang] = make(map[string]string)
	}
	for k, v := range messages {
		t.messages[lang][k] = v
	}
}

// LoadDir 从目录加载所有语言文件（支持 JSON/YAML）
// 文件名作为语言代码，如 zh-CN.json、en-US.yaml
func (t *Translator) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read i18n directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := filepath.Ext(name)
		lang := strings.TrimSuffix(name, ext)

		filePath := filepath.Join(dir, name)
		if err := t.LoadFile(lang, filePath); err != nil {
			return fmt.Errorf("failed to load %s: %w", filePath, err)
		}
	}

	return nil
}

// LoadFile 加载单个语言文件
func (t *Translator) LoadFile(lang, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	messages := make(map[string]string)
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".json":
		// 支持嵌套 JSON，展平为 dot-notation
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("failed to parse JSON %s: %w", filePath, err)
		}
		flattenMap("", raw, messages)
	case ".yaml", ".yml":
		var raw map[string]interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("failed to parse YAML %s: %w", filePath, err)
		}
		flattenMap("", raw, messages)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	t.Register(lang, messages)
	return nil
}

// T 翻译消息（使用默认语言）
func (t *Translator) T(key string, args ...interface{}) string {
	return t.Tr(t.Default(), key, args...)
}

// Tr 翻译消息（指定语言）
func (t *Translator) Tr(lang, key string, args ...interface{}) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// 先尝试精确匹配
	if msgs, ok := t.messages[lang]; ok {
		if msg, ok := msgs[key]; ok {
			return formatMessage(msg, args...)
		}
	}

	// 尝试匹配基础语言（如 zh-CN -> zh）
	baseLang := strings.Split(lang, "-")[0]
	if baseLang != lang {
		if msgs, ok := t.messages[baseLang]; ok {
			if msg, ok := msgs[key]; ok {
				return formatMessage(msg, args...)
			}
		}
	}

	// 回退到默认语言
	if lang != t.defaultLang {
		if msgs, ok := t.messages[t.defaultLang]; ok {
			if msg, ok := msgs[key]; ok {
				return formatMessage(msg, args...)
			}
		}
	}

	// 都找不到返回 key 本身
	return key
}

// HasLang 检查是否支持某语言
func (t *Translator) HasLang(lang string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.messages[lang]
	return ok
}

// Languages 获取所有已注册的语言列表
func (t *Translator) Languages() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	langs := make([]string, 0, len(t.messages))
	for lang := range t.messages {
		langs = append(langs, lang)
	}
	return langs
}

// --- Gin 集成 ---

const contextKey = "landc_i18n_lang"

// Middleware 创建 Gin 中间件，从 Accept-Language/query/header 解析语言
func Middleware(translator *Translator) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := detectLanguage(c, translator)
		c.Set(contextKey, lang)
		c.Next()
	}
}

// FromContext 从 Gin Context 获取当前请求的语言
func FromContext(c *gin.Context) string {
	if lang, exists := c.Get(contextKey); exists {
		return lang.(string)
	}
	return ""
}

// TrContext 根据请求上下文翻译
func TrContext(c *gin.Context, translator *Translator, key string, args ...interface{}) string {
	lang := FromContext(c)
	if lang == "" {
		lang = translator.Default()
	}
	return translator.Tr(lang, key, args...)
}

// detectLanguage 从请求中探测语言
func detectLanguage(c *gin.Context, t *Translator) string {
	// 1. 优先从 query 参数 lang
	if lang := c.Query("lang"); lang != "" && t.HasLang(lang) {
		return lang
	}

	// 2. 从自定义 header X-Language
	if lang := c.GetHeader("X-Language"); lang != "" && t.HasLang(lang) {
		return lang
	}

	// 3. 从 Accept-Language header
	acceptLang := c.GetHeader("Accept-Language")
	if acceptLang != "" {
		langs := parseAcceptLanguage(acceptLang)
		for _, lang := range langs {
			if t.HasLang(lang) {
				return lang
			}
			// 尝试基础语言
			base := strings.Split(lang, "-")[0]
			if t.HasLang(base) {
				return base
			}
		}
	}

	return t.Default()
}

// parseAcceptLanguage 简单解析 Accept-Language header
func parseAcceptLanguage(header string) []string {
	parts := strings.Split(header, ",")
	langs := make([]string, 0, len(parts))
	for _, part := range parts {
		// 去掉 quality factor (;q=0.9)
		lang := strings.TrimSpace(strings.Split(part, ";")[0])
		if lang != "" && lang != "*" {
			langs = append(langs, lang)
		}
	}
	return langs
}

// formatMessage 格式化消息（支持 {0} {1} 占位符和 %s %d 风格）
func formatMessage(msg string, args ...interface{}) string {
	if len(args) == 0 {
		return msg
	}

	// 先尝试 {0} {1} 风格
	result := msg
	hasPlaceholder := false
	for i, arg := range args {
		placeholder := fmt.Sprintf("{%d}", i)
		if strings.Contains(result, placeholder) {
			result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", arg))
			hasPlaceholder = true
		}
	}

	if hasPlaceholder {
		return result
	}

	// 再尝试 fmt.Sprintf 风格
	if strings.Contains(msg, "%") {
		return fmt.Sprintf(msg, args...)
	}

	return msg
}

// flattenMap 将嵌套 map 展平为 dot-notation
func flattenMap(prefix string, m map[string]interface{}, result map[string]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case string:
			result[key] = val
		case map[string]interface{}:
			flattenMap(key, val, result)
		default:
			result[key] = fmt.Sprintf("%v", val)
		}
	}
}
