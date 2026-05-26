package tests

import (
	"testing"

	"github.com/LandcLi/landc-go/log/facade"
)

func TestCreateLoggerWithOptions(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		opts     []facade.Option
	}{
		{
			name:     "console with options",
			provider: "console",
			opts: []facade.Option{
				facade.WithLevel(facade.WarnLevel),
				facade.WithFormat("json"),
			},
		},
		{
			name:     "std with options",
			provider: "std",
			opts: []facade.Option{
				facade.WithLevel(facade.ErrorLevel),
				facade.WithFormat("text"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := facade.GetLoggerWithProvider("test", tt.provider, tt.opts...)
			if log == nil {
				t.Fatalf("创建日志实例失败")
			}

			log.Info("测试日志")
		})
	}
}

func TestGetLoggerWithProviderWithOptions(t *testing.T) {
	log := facade.GetLoggerWithProvider("test-app", "console",
		facade.WithLevel(facade.DebugLevel),
		facade.WithFormat("json"),
		facade.WithOutputPath("stdout"),
	)

	if log == nil {
		t.Fatal("创建日志实例失败")
	}

	log.Debug("调试日志")
	log.Info("信息日志")
}
