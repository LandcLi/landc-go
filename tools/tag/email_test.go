package tag

import (
	"testing"
)

func TestEmailValidator1(t *testing.T) {
	validator := &EmailValidator{}

	err := validator.Validate("invalid-email")
	if err == nil {
		t.Error("应该返回错误")
	} else {
		t.Logf("错误信息: %v", err)
	}

	err = validator.Validate("test@example.com")
	if err != nil {
		t.Errorf("不应该返回错误: %v", err)
	}
}
