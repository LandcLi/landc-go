package generate

import (
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestUUID 测试 UUID 生成
func TestUUID(t *testing.T) {
	id := UUID()
	if id == "" {
		t.Error("UUID should not be empty")
	}
	// 验证 UUID 格式
	if _, err := uuid.Parse(id); err != nil {
		t.Errorf("UUID should be valid: %v", err)
	}
}

// TestUUIDv1 测试 UUID v1 生成
func TestUUIDv1(t *testing.T) {
	id := UUIDv1()
	if id == "" {
		t.Error("UUIDv1 should not be empty")
	}
	// 验证 UUID 格式
	if _, err := uuid.Parse(id); err != nil {
		t.Errorf("UUIDv1 should be valid: %v", err)
	}
}

// TestUUIDv3 测试 UUID v3 生成
func TestUUIDv3(t *testing.T) {
	// 创建一个随机的命名空间UUID
	namespace := uuid.New()
	name := "example.com"
	id := UUIDv3(namespace, name)
	if id == "" {
		t.Error("UUIDv3 should not be empty")
	}
	// 验证 UUID 格式
	if _, err := uuid.Parse(id); err != nil {
		t.Errorf("UUIDv3 should be valid: %v", err)
	}
}

// TestUUIDv5 测试 UUID v5 生成
func TestUUIDv5(t *testing.T) {
	// 创建一个随机的命名空间UUID
	namespace := uuid.New()
	name := "example.com"
	id := UUIDv5(namespace, name)
	if id == "" {
		t.Error("UUIDv5 should not be empty")
	}
	// 验证 UUID 格式
	if _, err := uuid.Parse(id); err != nil {
		t.Errorf("UUIDv5 should be valid: %v", err)
	}
}

// TestRandomInt 测试随机整数生成
func TestRandomInt(t *testing.T) {
	min := 1
	max := 100
	val, err := RandomInt(min, max)
	if err != nil {
		t.Errorf("RandomInt should not return error: %v", err)
	}
	if val < min || val > max {
		t.Errorf("RandomInt should return value between %d and %d, got %d", min, max, val)
	}
}

// TestMustRandomInt 测试 MustRandomInt 函数
func TestMustRandomInt(t *testing.T) {
	min := 1
	max := 100
	val := MustRandomInt(min, max)
	if val < min || val > max {
		t.Errorf("MustRandomInt should return value between %d and %d, got %d", min, max, val)
	}
}

// TestRandomString 测试随机字符串生成
func TestRandomString(t *testing.T) {
	length := 10
	str, err := RandomString(length)
	if err != nil {
		t.Errorf("RandomString should not return error: %v", err)
	}
	if len(str) != length {
		t.Errorf("RandomString should return string of length %d, got %d", length, len(str))
	}
}

// TestMustRandomString 测试 MustRandomString 函数
func TestMustRandomString(t *testing.T) {
	length := 10
	str := MustRandomString(length)
	if len(str) != length {
		t.Errorf("MustRandomString should return string of length %d, got %d", length, len(str))
	}
}

// TestRandomStringWithCharset 测试带自定义字符集的随机字符串生成
func TestRandomStringWithCharset(t *testing.T) {
	length := 10
	charset := "abc"
	str, err := RandomStringWithCharset(length, charset)
	if err != nil {
		t.Errorf("RandomStringWithCharset should not return error: %v", err)
	}
	if len(str) != length {
		t.Errorf("RandomStringWithCharset should return string of length %d, got %d", length, len(str))
	}
	// 验证字符串只包含指定字符集的字符
	for _, c := range str {
		if c != 'a' && c != 'b' && c != 'c' {
			t.Errorf("RandomStringWithCharset should only contain characters from charset, got %c", c)
		}
	}
}

// TestCaptcha 测试数字验证码生成
func TestCaptcha(t *testing.T) {
	length := 6
	captcha, err := Captcha(length)
	if err != nil {
		t.Errorf("Captcha should not return error: %v", err)
	}
	if len(captcha) != length {
		t.Errorf("Captcha should return string of length %d, got %d", length, len(captcha))
	}
	// 验证验证码只包含数字
	match, _ := regexp.MatchString("^\\d+$", captcha)
	if !match {
		t.Errorf("Captcha should only contain digits, got %s", captcha)
	}
}

// TestMustCaptcha 测试 MustCaptcha 函数
func TestMustCaptcha(t *testing.T) {
	length := 6
	captcha := MustCaptcha(length)
	if len(captcha) != length {
		t.Errorf("MustCaptcha should return string of length %d, got %d", length, len(captcha))
	}
	// 验证验证码只包含数字
	match, _ := regexp.MatchString("^\\d+$", captcha)
	if !match {
		t.Errorf("MustCaptcha should only contain digits, got %s", captcha)
	}
}

// TestAlphaCaptcha 测试字母验证码生成
func TestAlphaCaptcha(t *testing.T) {
	length := 6
	captcha, err := AlphaCaptcha(length)
	if err != nil {
		t.Errorf("AlphaCaptcha should not return error: %v", err)
	}
	if len(captcha) != length {
		t.Errorf("AlphaCaptcha should return string of length %d, got %d", length, len(captcha))
	}
	// 验证验证码只包含字母
	match, _ := regexp.MatchString("^[a-zA-Z]+$", captcha)
	if !match {
		t.Errorf("AlphaCaptcha should only contain letters, got %s", captcha)
	}
}

// TestMustAlphaCaptcha 测试 MustAlphaCaptcha 函数
func TestMustAlphaCaptcha(t *testing.T) {
	length := 6
	captcha := MustAlphaCaptcha(length)
	if len(captcha) != length {
		t.Errorf("MustAlphaCaptcha should return string of length %d, got %d", length, len(captcha))
	}
	// 验证验证码只包含字母
	match, _ := regexp.MatchString("^[a-zA-Z]+$", captcha)
	if !match {
		t.Errorf("MustAlphaCaptcha should only contain letters, got %s", captcha)
	}
}

// TestAlphaNumericCaptcha 测试字母数字混合验证码生成
func TestAlphaNumericCaptcha(t *testing.T) {
	length := 6
	captcha, err := AlphaNumericCaptcha(length)
	if err != nil {
		t.Errorf("AlphaNumericCaptcha should not return error: %v", err)
	}
	if len(captcha) != length {
		t.Errorf("AlphaNumericCaptcha should return string of length %d, got %d", length, len(captcha))
	}
	// 验证验证码只包含字母和数字
	match, _ := regexp.MatchString("^[a-zA-Z0-9]+$", captcha)
	if !match {
		t.Errorf("AlphaNumericCaptcha should only contain letters and digits, got %s", captcha)
	}
}

// TestMustAlphaNumericCaptcha 测试 MustAlphaNumericCaptcha 函数
func TestMustAlphaNumericCaptcha(t *testing.T) {
	length := 6
	captcha := MustAlphaNumericCaptcha(length)
	if len(captcha) != length {
		t.Errorf("MustAlphaNumericCaptcha should return string of length %d, got %d", length, len(captcha))
	}
	// 验证验证码只包含字母和数字
	match, _ := regexp.MatchString("^[a-zA-Z0-9]+$", captcha)
	if !match {
		t.Errorf("MustAlphaNumericCaptcha should only contain letters and digits, got %s", captcha)
	}
}

// TestRandomIP 测试随机 IP 地址生成
func TestRandomIP(t *testing.T) {
	ip, err := RandomIP()
	if err != nil {
		t.Errorf("RandomIP should not return error: %v", err)
	}
	if ip == "" {
		t.Error("RandomIP should not return empty string")
	}
	// 验证 IP 地址格式
	match, _ := regexp.MatchString("^\\d+\\.\\d+\\.\\d+\\.\\d+$", ip)
	if !match {
		t.Errorf("RandomIP should return valid IP address, got %s", ip)
	}
}

// TestMustRandomIP 测试 MustRandomIP 函数
func TestMustRandomIP(t *testing.T) {
	ip := MustRandomIP()
	if ip == "" {
		t.Error("MustRandomIP should not return empty string")
	}
	// 验证 IP 地址格式
	match, _ := regexp.MatchString("^\\d+\\.\\d+\\.\\d+\\.\\d+$", ip)
	if !match {
		t.Errorf("MustRandomIP should return valid IP address, got %s", ip)
	}
}

// TestRandomMAC 测试随机 MAC 地址生成
func TestRandomMAC(t *testing.T) {
	mac, err := RandomMAC()
	if err != nil {
		t.Errorf("RandomMAC should not return error: %v", err)
	}
	if mac == "" {
		t.Error("RandomMAC should not return empty string")
	}
	// 验证 MAC 地址格式
	match, _ := regexp.MatchString("^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$", mac)
	if !match {
		t.Errorf("RandomMAC should return valid MAC address, got %s", mac)
	}
}

// TestMustRandomMAC 测试 MustRandomMAC 函数
func TestMustRandomMAC(t *testing.T) {
	mac := MustRandomMAC()
	if mac == "" {
		t.Error("MustRandomMAC should not return empty string")
	}
	// 验证 MAC 地址格式
	match, _ := regexp.MatchString("^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$", mac)
	if !match {
		t.Errorf("MustRandomMAC should return valid MAC address, got %s", mac)
	}
}

// TestRandomBool 测试随机布尔值生成
func TestRandomBool(t *testing.T) {
	val, err := RandomBool()
	if err != nil {
		t.Errorf("RandomBool should not return error: %v", err)
	}
	// 验证返回值是布尔类型（这里只是为了测试函数执行成功）
	_ = val
}

// TestMustRandomBool 测试 MustRandomBool 函数
func TestMustRandomBool(t *testing.T) {
	val := MustRandomBool()
	// 验证返回值是布尔类型（这里只是为了测试函数执行成功）
	_ = val
}

// TestRandomFloat 测试随机浮点数生成
func TestRandomFloat(t *testing.T) {
	min := 1.0
	max := 10.0
	val, err := RandomFloat(min, max)
	if err != nil {
		t.Errorf("RandomFloat should not return error: %v", err)
	}
	if val < min || val >= max {
		t.Errorf("RandomFloat should return value between %f and %f, got %f", min, max, val)
	}
}

// TestMustRandomFloat 测试 MustRandomFloat 函数
func TestMustRandomFloat(t *testing.T) {
	min := 1.0
	max := 10.0
	val := MustRandomFloat(min, max)
	if val < min || val >= max {
		t.Errorf("MustRandomFloat should return value between %f and %f, got %f", min, max, val)
	}
}

// TestRandomTimestamp 测试随机时间戳生成
func TestRandomTimestamp(t *testing.T) {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	timestamp, err := RandomTimestamp(start, end)
	if err != nil {
		t.Errorf("RandomTimestamp should not return error: %v", err)
	}
	if timestamp < start.Unix() || timestamp > end.Unix() {
		t.Errorf("RandomTimestamp should return value between %d and %d, got %d", start.Unix(), end.Unix(), timestamp)
	}
}

// TestMustRandomTimestamp 测试 MustRandomTimestamp 函数
func TestMustRandomTimestamp(t *testing.T) {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	timestamp := MustRandomTimestamp(start, end)
	if timestamp < start.Unix() || timestamp > end.Unix() {
		t.Errorf("MustRandomTimestamp should return value between %d and %d, got %d", start.Unix(), end.Unix(), timestamp)
	}
}

// TestRandomTime 测试随机时间生成
func TestRandomTime(t *testing.T) {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	tm, err := RandomTime(start, end)
	if err != nil {
		t.Errorf("RandomTime should not return error: %v", err)
	}
	if tm.Before(start) || tm.After(end) {
		t.Errorf("RandomTime should return time between %v and %v, got %v", start, end, tm)
	}
}

// TestMustRandomTime 测试 MustRandomTime 函数
func TestMustRandomTime(t *testing.T) {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	tm := MustRandomTime(start, end)
	if tm.Before(start) || tm.After(end) {
		t.Errorf("MustRandomTime should return time between %v and %v, got %v", start, end, tm)
	}
}

// TestGenerateStrongPassword 测试生成强密码
func TestGenerateStrongPassword(t *testing.T) {
	length := 12
	password, err := GenerateStrongPassword(length)
	if err != nil {
		t.Errorf("GenerateStrongPassword should not return error: %v", err)
	}
	if len(password) != length {
		t.Errorf("GenerateStrongPassword should return string of length %d, got %d", length, len(password))
	}
	
	// 验证密码包含至少一个小写字母
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		t.Error("GenerateStrongPassword should include at least one lowercase letter")
	}
	// 验证密码包含至少一个大写字母
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		t.Error("GenerateStrongPassword should include at least one uppercase letter")
	}
	// 验证密码包含至少一个数字
	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		t.Error("GenerateStrongPassword should include at least one digit")
	}
	// 验证密码包含至少一个特殊符号
	if !regexp.MustCompile(`[!@#$%^&*()_+-=\[\]{}|;:,.<>?]`).MatchString(password) {
		t.Error("GenerateStrongPassword should include at least one special character")
	}
}

// TestMustGenerateStrongPassword 测试 MustGenerateStrongPassword 函数
func TestMustGenerateStrongPassword(t *testing.T) {
	length := 12
	password := MustGenerateStrongPassword(length)
	if len(password) != length {
		t.Errorf("MustGenerateStrongPassword should return string of length %d, got %d", length, len(password))
	}
	
	// 验证密码包含至少一个小写字母
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		t.Error("MustGenerateStrongPassword should include at least one lowercase letter")
	}
	// 验证密码包含至少一个大写字母
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		t.Error("MustGenerateStrongPassword should include at least one uppercase letter")
	}
	// 验证密码包含至少一个数字
	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		t.Error("MustGenerateStrongPassword should include at least one digit")
	}
	// 验证密码包含至少一个特殊符号
	if !regexp.MustCompile(`[!@#$%^&*()_+-=\[\]{}|;:,.<>?]`).MatchString(password) {
		t.Error("MustGenerateStrongPassword should include at least one special character")
	}
}

// TestGenerateStrongPassword_ShortLength 测试生成过短的密码
func TestGenerateStrongPassword_ShortLength(t *testing.T) {
	length := 7
	_, err := GenerateStrongPassword(length)
	if err == nil {
		t.Error("GenerateStrongPassword should return error for length less than 8")
	}
}
