package web

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUploadFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()

	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
		result, err := UploadFile(c, "file", &UploadConfig{
			UploadDir:   tmpDir,
			MaxFileSize: 1 << 20, // 1MB
		})
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, result)
	})

	// 创建 multipart 请求
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("hello world"))
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 验证文件已保存
	files, _ := os.ReadDir(tmpDir)
	if len(files) != 1 {
		t.Fatalf("expected 1 file in upload dir, got %d", len(files))
	}
}

func TestUploadFile_ExtensionRestriction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()

	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
		_, err := UploadFile(c, "file", &UploadConfig{
			UploadDir:         tmpDir,
			AllowedExtensions: []string{".png", ".jpg"},
		})
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	})

	// 上传不允许的扩展名
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "hack.exe")
	part.Write([]byte("payload"))
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for disallowed extension, got %d", w.Code)
	}
}

func TestUploadFile_SizeLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()

	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
		_, err := UploadFile(c, "file", &UploadConfig{
			UploadDir:   tmpDir,
			MaxFileSize: 10, // 10 bytes
		})
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "big.txt")
	part.Write([]byte("this content exceeds 10 bytes limit"))
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for oversized file, got %d", w.Code)
	}
}

func TestUploadFiles_Multiple(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()

	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
		results, err := UploadFiles(c, "files", &UploadConfig{
			UploadDir: tmpDir,
		})
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"count": len(results)})
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for i := 0; i < 3; i++ {
		part, _ := writer.CreateFormFile("files", "file"+string(rune('0'+i))+".txt")
		part.Write([]byte("content"))
	}
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	files, _ := os.ReadDir(tmpDir)
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
}

func TestServeFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "download.txt")
	_ = os.WriteFile(testFile, []byte("download content"), 0o600)

	r := gin.New()
	r.GET("/download", func(c *gin.Context) {
		ServeFile(c, testFile, "myfile.txt")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/download", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	disposition := w.Header().Get("Content-Disposition")
	if disposition == "" {
		t.Error("expected Content-Disposition header")
	}

	body, _ := io.ReadAll(w.Body)
	if string(body) != "download content" {
		t.Errorf("expected 'download content', got '%s'", string(body))
	}
}

func TestServeFile_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/download", func(c *gin.Context) {
		ServeFile(c, "/nonexistent/file.txt", "")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/download", http.NoBody)
	r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDefaultValueTag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type ListReq struct {
		Page     int    `source:"query" name:"page" d:"1"`
		PageSize int    `source:"query" name:"page_size" d:"20"`
		Sort     string `source:"query" name:"sort" d:"created_at"`
	}

	// 测试 parseParamMeta 读取 d tag
	typ := reflect.TypeOf(ListReq{})

	field0 := typ.Field(0)
	meta0, _ := parseParamMeta(field0)
	if meta0.Default != "1" {
		t.Errorf("expected default '1', got '%s'", meta0.Default)
	}

	field1 := typ.Field(1)
	meta1, _ := parseParamMeta(field1)
	if meta1.Default != "20" {
		t.Errorf("expected default '20', got '%s'", meta1.Default)
	}

	field2 := typ.Field(2)
	meta2, _ := parseParamMeta(field2)
	if meta2.Default != "created_at" {
		t.Errorf("expected default 'created_at', got '%s'", meta2.Default)
	}
}
