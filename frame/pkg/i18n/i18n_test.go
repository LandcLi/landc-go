package i18n

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTranslator_Basic(t *testing.T) {
	tr := New("en")
	tr.Register("en", map[string]string{
		"hello":   "Hello",
		"welcome": "Welcome, {0}!",
	})
	tr.Register("zh-CN", map[string]string{
		"hello":   "你好",
		"welcome": "欢迎, {0}!",
	})

	if got := tr.T("hello"); got != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", got)
	}

	if got := tr.Tr("zh-CN", "hello"); got != "你好" {
		t.Errorf("expected '你好', got '%s'", got)
	}

	if got := tr.Tr("en", "welcome", "Alice"); got != "Welcome, Alice!" {
		t.Errorf("expected 'Welcome, Alice!', got '%s'", got)
	}

	if got := tr.Tr("zh-CN", "welcome", "张三"); got != "欢迎, 张三!" {
		t.Errorf("expected '欢迎, 张三!', got '%s'", got)
	}
}

func TestTranslator_Fallback(t *testing.T) {
	tr := New("en")
	tr.Register("en", map[string]string{
		"hello":   "Hello",
		"only_en": "Only in English",
	})
	tr.Register("zh-CN", map[string]string{
		"hello": "你好",
	})

	// zh-CN 没有 only_en，应回退到 en
	if got := tr.Tr("zh-CN", "only_en"); got != "Only in English" {
		t.Errorf("expected fallback to 'Only in English', got '%s'", got)
	}

	// 不存在的 key 返回 key 本身
	if got := tr.Tr("en", "nonexist"); got != "nonexist" {
		t.Errorf("expected 'nonexist', got '%s'", got)
	}
}

func TestTranslator_BaseLangFallback(t *testing.T) {
	tr := New("en")
	tr.Register("zh", map[string]string{
		"hello": "你好",
	})

	// zh-TW 精确匹配失败，应回退到 zh
	if got := tr.Tr("zh-TW", "hello"); got != "你好" {
		t.Errorf("expected fallback to zh 'hello', got '%s'", got)
	}
}

func TestTranslator_LoadDir(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建 JSON 文件
	enJSON := `{"greeting": {"hello": "Hello", "bye": "Goodbye"}, "error": "Error occurred"}`
	_ = os.WriteFile(filepath.Join(tmpDir, "en.json"), []byte(enJSON), 0o600)

	// 创建 YAML 文件
	zhYAML := "greeting:\n  hello: 你好\n  bye: 再见\nerror: 发生错误\n"
	_ = os.WriteFile(filepath.Join(tmpDir, "zh-CN.yaml"), []byte(zhYAML), 0o600)

	tr := New("en")
	if err := tr.LoadDir(tmpDir); err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}

	if got := tr.Tr("en", "greeting.hello"); got != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", got)
	}

	if got := tr.Tr("zh-CN", "greeting.bye"); got != "再见" {
		t.Errorf("expected '再见', got '%s'", got)
	}

	if got := tr.Tr("en", "error"); got != "Error occurred" {
		t.Errorf("expected 'Error occurred', got '%s'", got)
	}
}

func TestTranslator_FmtStyle(t *testing.T) {
	tr := New("en")
	tr.Register("en", map[string]string{
		"count": "You have %d items",
	})

	if got := tr.Tr("en", "count", 5); got != "You have 5 items" {
		t.Errorf("expected 'You have 5 items', got '%s'", got)
	}
}

func TestMiddleware_AcceptLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tr := New("en")
	tr.Register("en", map[string]string{"hello": "Hello"})
	tr.Register("zh-CN", map[string]string{"hello": "你好"})

	r := gin.New()
	r.Use(Middleware(tr))
	r.GET("/test", func(c *gin.Context) {
		msg := TrContext(c, tr, "hello")
		c.String(200, msg)
	})

	// 测试 Accept-Language: zh-CN
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	r.ServeHTTP(w, req)
	if w.Body.String() != "你好" {
		t.Errorf("expected '你好', got '%s'", w.Body.String())
	}

	// 测试 query 参数 ?lang=en
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test?lang=en", http.NoBody)
	r.ServeHTTP(w2, req2)
	if w2.Body.String() != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", w2.Body.String())
	}
}

func TestMiddleware_XLanguageHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tr := New("en")
	tr.Register("en", map[string]string{"hello": "Hello"})
	tr.Register("zh-CN", map[string]string{"hello": "你好"})

	r := gin.New()
	r.Use(Middleware(tr))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, TrContext(c, tr, "hello"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", http.NoBody)
	req.Header.Set("X-Language", "zh-CN")
	r.ServeHTTP(w, req)
	if w.Body.String() != "你好" {
		t.Errorf("expected '你好', got '%s'", w.Body.String())
	}
}

func TestHasLang(t *testing.T) {
	tr := New("en")
	tr.Register("en", map[string]string{"a": "b"})

	if !tr.HasLang("en") {
		t.Error("expected HasLang('en') = true")
	}
	if tr.HasLang("fr") {
		t.Error("expected HasLang('fr') = false")
	}
}

func TestLanguages(t *testing.T) {
	tr := New("en")
	tr.Register("en", map[string]string{"a": "b"})
	tr.Register("zh-CN", map[string]string{"a": "c"})

	langs := tr.Languages()
	if len(langs) != 2 {
		t.Errorf("expected 2 languages, got %d", len(langs))
	}
}
