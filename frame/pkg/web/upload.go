package web

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/LandcLi/landc-go/tools/generate"
	"github.com/gin-gonic/gin"
)

// UploadConfig 上传配置
type UploadConfig struct {
	// MaxFileSize 最大文件大小（字节，默认 32MB）
	MaxFileSize int64
	// AllowedExtensions 允许的文件扩展名（为空则不限制）
	AllowedExtensions []string
	// UploadDir 上传目录（默认 ./uploads）
	UploadDir string
	// FileNameFunc 自定义文件名生成函数（默认 UUID+原扩展名）
	FileNameFunc func(originalName string) string
}

// UploadResult 上传结果
type UploadResult struct {
	OriginalName string `json:"original_name"`
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	FileSize     int64  `json:"file_size"`
	MimeType     string `json:"mime_type"`
}

// defaultUploadConfig 默认上传配置
func defaultUploadConfig() *UploadConfig {
	return &UploadConfig{
		MaxFileSize: 32 << 20, // 32MB
		UploadDir:   "./uploads",
		FileNameFunc: func(originalName string) string {
			ext := filepath.Ext(originalName)
			return generate.UUID() + ext
		},
	}
}

// UploadFile 上传单个文件
func UploadFile(c *gin.Context, fieldName string, config *UploadConfig) (*UploadResult, error) {
	if config == nil {
		config = defaultUploadConfig()
	}
	applyUploadDefaults(config)

	file, header, err := c.Request.FormFile(fieldName)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from field '%s': %w", fieldName, err)
	}
	defer file.Close()

	if err := validateFile(header, config); err != nil {
		return nil, err
	}

	return saveFile(file, header, config)
}

// UploadFiles 上传多个文件
func UploadFiles(c *gin.Context, fieldName string, config *UploadConfig) ([]*UploadResult, error) {
	if config == nil {
		config = defaultUploadConfig()
	}
	applyUploadDefaults(config)

	form, err := c.MultipartForm()
	if err != nil {
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	files := form.File[fieldName]
	if len(files) == 0 {
		return nil, fmt.Errorf("no files found for field '%s'", fieldName)
	}

	results := make([]*UploadResult, 0, len(files))
	for _, header := range files {
		if err := validateFile(header, config); err != nil {
			return results, err
		}

		file, err := header.Open()
		if err != nil {
			return results, fmt.Errorf("failed to open file %s: %w", header.Filename, err)
		}

		result, err := saveFile(file, header, config)
		file.Close()
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}

// ServeFile 下载文件（Content-Disposition: attachment）
func ServeFile(c *gin.Context, filePath string, downloadName string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"code":    40400,
			"message": "File not found",
		})
		return
	}

	if downloadName == "" {
		downloadName = filepath.Base(filePath)
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
	c.File(filePath)
}

// ServeFileInline 在线预览文件（Content-Disposition: inline）
func ServeFileInline(c *gin.Context, filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"code":    40400,
			"message": "File not found",
		})
		return
	}

	c.Header("Content-Disposition", "inline")
	c.File(filePath)
}

func applyUploadDefaults(config *UploadConfig) {
	if config.MaxFileSize <= 0 {
		config.MaxFileSize = 32 << 20
	}
	if config.UploadDir == "" {
		config.UploadDir = "./uploads"
	}
	if config.FileNameFunc == nil {
		config.FileNameFunc = func(originalName string) string {
			ext := filepath.Ext(originalName)
			return generate.UUID() + ext
		}
	}
}

func validateFile(header *multipart.FileHeader, config *UploadConfig) error {
	if header.Size > config.MaxFileSize {
		return fmt.Errorf("file %s exceeds maximum size (%d bytes)", header.Filename, config.MaxFileSize)
	}

	if len(config.AllowedExtensions) > 0 {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		allowed := false
		for _, e := range config.AllowedExtensions {
			if strings.ToLower(e) == ext || "."+strings.ToLower(e) == ext {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file extension %s is not allowed", ext)
		}
	}

	return nil
}

func saveFile(file multipart.File, header *multipart.FileHeader, config *UploadConfig) (*UploadResult, error) {
	// 确保上传目录存在
	if err := os.MkdirAll(config.UploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	fileName := config.FileNameFunc(header.Filename)
	destPath := filepath.Join(config.UploadDir, fileName)

	dst, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	return &UploadResult{
		OriginalName: header.Filename,
		FileName:     fileName,
		FilePath:     destPath,
		FileSize:     written,
		MimeType:     header.Header.Get("Content-Type"),
	}, nil
}
