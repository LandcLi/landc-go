package mail

import (
	"testing"
	"time"
)

// TestConfig 测试配置类
func TestConfig(t *testing.T) {
	// 测试创建自定义配置
	customConfig := NewConfig(
		"smtp.example.com", 587,
		"pop.example.com", 995,
		"imap.example.com", 993,
		"user@example.com", "password",
		true, "plain",
		60*time.Second, 5, 10*time.Second,
	)
	if customConfig == nil {
		t.Error("NewConfig() should return a non-nil Config")
	}

	// 测试通过配置创建客户端
	client := NewClientWithConfig(customConfig)
	if client == nil {
		t.Error("NewClientWithConfig() should return a non-nil Client")
	}
}
