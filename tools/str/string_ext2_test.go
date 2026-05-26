package str

import (
	"testing"
)

// ==================== 中文拼音处理测试 ====================

func TestToPinyin(t *testing.T) {
	if ToPinyin("你好") != "nihao" {
		t.Errorf("ToPinyin should return nihao")
	}
}

func TestToPinyinWithFirstLetter(t *testing.T) {
	if ToPinyinWithFirstLetter("你好世界") != "nhsj" {
		t.Errorf("ToPinyinWithFirstLetter should return nhsj")
	}
}

func TestToPinyinWithSep(t *testing.T) {
	if ToPinyinWithSep("你好世界", "-") != "ni-hao-shi-jie" {
		t.Errorf("ToPinyinWithSep should return ni-hao-shi-jie")
	}
}

// ==================== 加密哈希测试 ====================

func TestMD5(t *testing.T) {
	result := MD5("hello")
	if result != "5d41402abc4b2a76b9719d911017c592" {
		t.Errorf("MD5 should return 5d41402abc4b2a76b9719d911017c592")
	}
}

func TestSHA1(t *testing.T) {
	result := SHA1("hello")
	if result != "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d" {
		t.Errorf("SHA1 should return aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d")
	}
}

// ==================== Base64编解码测试 ====================

func TestBase64Encode(t *testing.T) {
	if Base64Encode("hello") != "aGVsbG8=" {
		t.Errorf("Base64Encode should return aGVsbG8=")
	}
}

func TestBase64Decode(t *testing.T) {
	result, err := Base64Decode("aGVsbG8=")
	if err != nil || result != "hello" {
		t.Errorf("Base64Decode should return hello")
	}
}

// ==================== 随机字符串生成测试 ====================

func TestRandomString(t *testing.T) {
	result := RandomString(10)
	if len(result) != 10 {
		t.Errorf("RandomString should return string with length 10")
	}
}

func TestRandomNumberString(t *testing.T) {
	result := RandomNumberString(10)
	if len(result) != 10 {
		t.Errorf("RandomNumberString should return string with length 10")
	}
}

// ==================== 字符串转换测试 ====================

func TestToInt(t *testing.T) {
	result, err := ToInt("123")
	if err != nil || result != 123 {
		t.Errorf("ToInt should return 123")
	}
}

func TestToIntDefault(t *testing.T) {
	result := ToIntDefault("123", 0)
	if result != 123 {
		t.Errorf("ToIntDefault should return 123")
	}
	result = ToIntDefault("abc", 0)
	if result != 0 {
		t.Errorf("ToIntDefault should return 0")
	}
}

func TestToFloat(t *testing.T) {
	result, err := ToFloat("123.45")
	if err != nil || result != 123.45 {
		t.Errorf("ToFloat should return 123.45")
	}
}

func TestToFloatDefault(t *testing.T) {
	result := ToFloatDefault("123.45", 0)
	if result != 123.45 {
		t.Errorf("ToFloatDefault should return 123.45")
	}
	result = ToFloatDefault("abc", 0)
	if result != 0 {
		t.Errorf("ToFloatDefault should return 0")
	}
}

func TestToBool(t *testing.T) {
	result, err := ToBool("true")
	if err != nil || result != true {
		t.Errorf("ToBool should return true")
	}
}

func TestToBoolDefault(t *testing.T) {
	result := ToBoolDefault("true", false)
	if result != true {
		t.Errorf("ToBoolDefault should return true")
	}
	result = ToBoolDefault("abc", false)
	if result != false {
		t.Errorf("ToBoolDefault should return false")
	}
}

// ==================== 字符串验证测试 ====================

func TestIsPhone(t *testing.T) {
	if !IsPhone("13800138000") {
		t.Errorf("IsPhone should return true")
	}
	if IsPhone("12345678901") {
		t.Errorf("IsPhone should return false")
	}
}

func TestIsIDCard(t *testing.T) {
	if !IsIDCard("110101199001011234") {
		t.Errorf("IsIDCard should return true")
	}
}

func TestIsIP(t *testing.T) {
	if !IsIP("192.168.1.1") {
		t.Errorf("IsIP should return true")
	}
	if IsIP("256.0.0.1") {
		t.Errorf("IsIP should return false")
	}
}

// ==================== 其他实用功能测试 ====================

func TestIsPalindrome(t *testing.T) {
	if !IsPalindrome("level") {
		t.Errorf("IsPalindrome should return true")
	}
	if IsPalindrome("hello") {
		t.Errorf("IsPalindrome should return false")
	}
}

func TestRotate(t *testing.T) {
	if Rotate("hello", 2) != "llohe" {
		t.Errorf("Rotate should return llohe")
	}
}

func TestSwapCase(t *testing.T) {
	if SwapCase("Hello World") != "hELLO wORLD" {
		t.Errorf("SwapCase should return hELLO wORLD")
	}
}

func TestRepeatWithSep(t *testing.T) {
	if RepeatWithSep("hello", 3, ",") != "hello,hello,hello" {
		t.Errorf("RepeatWithSep should return hello,hello,hello")
	}
}

func TestIndent(t *testing.T) {
	if Indent("hello\nworld", "  ") != "  hello\n  world" {
		t.Errorf("Indent should return   hello\n  world")
	}
}

func TestUnindent(t *testing.T) {
	if Unindent("  hello\n  world") != "hello\nworld" {
		t.Errorf("Unindent should return hello\nworld")
	}
}
