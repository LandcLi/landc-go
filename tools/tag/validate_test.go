package tag

import (
	"testing"
)

// TestRequiredValidator 测试必需性验证器
func TestRequiredValidator(t *testing.T) {
	validator := CreateRequiredValidator()

	// 测试nil值
	if err := validator.Validate(nil); err == nil {
		t.Error("nil值应该返回错误，但没有返回错误")
	}

	// 测试空字符串
	if err := validator.Validate(""); err == nil {
		t.Error("空字符串应该返回错误，但没有返回错误")
	}

	// 测试空数组
	if err := validator.Validate([]interface{}{}); err == nil {
		t.Error("空数组应该返回错误，但没有返回错误")
	}

	// 测试零值
	if err := validator.Validate(0); err == nil {
		t.Error("零值应该返回错误，但没有返回错误")
	}

	// 测试有效的值
	if err := validator.Validate("hello"); err != nil {
		t.Error("有效值应该通过验证，但返回了错误:", err)
	}
}

// TestEmailValidator 测试邮箱验证器
func TestEmailValidator(t *testing.T) {
	validator := &EmailValidator{}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试空字符串
	if err := validator.Validate(""); err == nil {
		t.Error("空字符串应该返回错误，但没有返回错误")
	}

	// 测试无效邮箱
	if err := validator.Validate("invalid-email"); err == nil {
		t.Error("无效邮箱应该返回错误，但没有返回错误")
	}

	// 测试有效邮箱
	if err := validator.Validate("test@example.com"); err != nil {
		t.Error("有效邮箱应该通过验证，但返回了错误:", err)
	}
}

// TestIPValidator 测试IP地址验证器
func TestIPValidator(t *testing.T) {
	validator := &IPValidator{}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试空字符串
	if err := validator.Validate(""); err == nil {
		t.Error("空字符串应该返回错误，但没有返回错误")
	}

	// 测试无效IP
	if err := validator.Validate("invalid-ip"); err == nil {
		t.Error("无效IP应该返回错误，但没有返回错误")
	}

	// 测试有效IPv4
	if err := validator.Validate("192.168.1.1"); err != nil {
		t.Error("有效IPv4应该通过验证，但返回了错误:", err)
	}

	// 测试有效IPv6
	if err := validator.Validate("2001:0db8:85a3:0000:0000:8a2e:0370:7334"); err != nil {
		t.Error("有效IPv6应该通过验证，但返回了错误:", err)
	}
}

// TestSizeValidator 测试大小验证器
func TestSizeValidator(t *testing.T) {
	validator := &SizeValidator{Size: 3}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试长度不符合的字符串
	if err := validator.Validate("hello"); err == nil {
		t.Error("长度不符合的字符串应该返回错误，但没有返回错误")
	}

	// 测试长度符合的字符串
	if err := validator.Validate("abc"); err != nil {
		t.Error("长度符合的字符串应该通过验证，但返回了错误:", err)
	}

	// 测试长度不符合的数组
	if err := validator.Validate([]interface{}{1, 2}); err == nil {
		t.Error("长度不符合的数组应该返回错误，但没有返回错误")
	}

	// 测试长度符合的数组
	if err := validator.Validate([]interface{}{1, 2, 3}); err != nil {
		t.Error("长度符合的数组应该通过验证，但返回了错误:", err)
	}
}

// TestLengthValidator 测试长度验证器
func TestLengthValidator(t *testing.T) {
	validator := &LengthValidator{Length: 3}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试长度不符合的字符串
	if err := validator.Validate("hello"); err == nil {
		t.Error("长度不符合的字符串应该返回错误，但没有返回错误")
	}

	// 测试长度符合的字符串
	if err := validator.Validate("abc"); err != nil {
		t.Error("长度符合的字符串应该通过验证，但返回了错误:", err)
	}
}

// TestBetweenValidator 测试范围验证器
func TestBetweenValidator(t *testing.T) {
	validator := &BetweenValidator{Min: 2, Max: 5}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试长度超出范围的字符串
	if err := validator.Validate("hello world"); err == nil {
		t.Error("长度超出范围的字符串应该返回错误，但没有返回错误")
	}

	// 测试长度在范围内的字符串
	if err := validator.Validate("hello"); err != nil {
		t.Error("长度在范围内的字符串应该通过验证，但返回了错误:", err)
	}

	// 测试值超出范围的数字
	if err := validator.Validate(10); err == nil {
		t.Error("值超出范围的数字应该返回错误，但没有返回错误")
	}

	// 测试值在范围内的数字
	if err := validator.Validate(3); err != nil {
		t.Error("值在范围内的数字应该通过验证，但返回了错误:", err)
	}
}

// TestMinValidator 测试最小值验证器
func TestMinValidator(t *testing.T) {
	validator := CreateMinValidator(3)

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试长度小于最小值的字符串
	if err := validator.Validate("ab"); err == nil {
		t.Error("长度小于最小值的字符串应该返回错误，但没有返回错误")
	}

	// 测试长度大于等于最小值的字符串
	if err := validator.Validate("abc"); err != nil {
		t.Error("长度大于等于最小值的字符串应该通过验证，但返回了错误:", err)
	}

	// 测试值小于最小值的数字
	if err := validator.Validate(2); err == nil {
		t.Error("值小于最小值的数字应该返回错误，但没有返回错误")
	}

	// 测试值大于等于最小值的数字
	if err := validator.Validate(3); err != nil {
		t.Error("值大于等于最小值的数字应该通过验证，但返回了错误:", err)
	}
}

// TestMaxValidator 测试最大值验证器
func TestMaxValidator(t *testing.T) {
	validator := CreateMaxValidator(5)

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试长度大于最大值的字符串
	if err := validator.Validate("hello world"); err == nil {
		t.Error("长度大于最大值的字符串应该返回错误，但没有返回错误")
	}

	// 测试长度小于等于最大值的字符串
	if err := validator.Validate("hello"); err != nil {
		t.Error("长度小于等于最大值的字符串应该通过验证，但返回了错误:", err)
	}

	// 测试值大于最大值的数字
	if err := validator.Validate(10); err == nil {
		t.Error("值大于最大值的数字应该返回错误，但没有返回错误")
	}

	// 测试值小于等于最大值的数字
	if err := validator.Validate(5); err != nil {
		t.Error("值小于等于最大值的数字应该通过验证，但返回了错误:", err)
	}
}

// TestInValidator 测试包含验证器
func TestInValidator(t *testing.T) {
	validator := &InValidator{Values: []string{"apple", "banana", "orange"}}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试空字符串
	if err := validator.Validate(""); err == nil {
		t.Error("空字符串应该返回错误，但没有返回错误")
	}

	// 测试不在列表中的值
	if err := validator.Validate("grape"); err == nil {
		t.Error("不在列表中的值应该返回错误，但没有返回错误")
	}

	// 测试在列表中的值
	if err := validator.Validate("apple"); err != nil {
		t.Error("在列表中的值应该通过验证，但返回了错误:", err)
	}
}

// TestNotInValidator 测试不包含验证器
func TestNotInValidator(t *testing.T) {
	validator := &NotInValidator{Values: []string{"apple", "banana", "orange"}}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试空字符串
	if err := validator.Validate(""); err == nil {
		t.Error("空字符串应该返回错误，但没有返回错误")
	}

	// 测试在列表中的值
	if err := validator.Validate("apple"); err == nil {
		t.Error("在列表中的值应该返回错误，但没有返回错误")
	}

	// 测试不在列表中的值
	if err := validator.Validate("grape"); err != nil {
		t.Error("不在列表中的值应该通过验证，但返回了错误:", err)
	}
}

// TestJSONValidator 测试JSON格式验证器
func TestJSONValidator(t *testing.T) {
	validator := &JSONValidator{}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试空字符串
	if err := validator.Validate(""); err == nil {
		t.Error("空字符串应该返回错误，但没有返回错误")
	}

	// 测试无效JSON
	if err := validator.Validate("invalid json"); err == nil {
		t.Error("无效JSON应该返回错误，但没有返回错误")
	}

	// 测试有效JSON
	if err := validator.Validate(`{"name": "test"}`); err != nil {
		t.Error("有效JSON应该通过验证，但返回了错误:", err)
	}
}

// TestIntegerValidator 测试整数验证器
func TestIntegerValidator(t *testing.T) {
	validator := &IntegerValidator{}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试空字符串
	if err := validator.Validate(""); err == nil {
		t.Error("空字符串应该返回错误，但没有返回错误")
	}

	// 测试无效整数字符串
	if err := validator.Validate("abc"); err == nil {
		t.Error("无效整数字符串应该返回错误，但没有返回错误")
	}

	// 测试有效整数字符串
	if err := validator.Validate("123"); err != nil {
		t.Error("有效整数字符串应该通过验证，但返回了错误:", err)
	}

	// 测试整数
	if err := validator.Validate(123); err != nil {
		t.Error("整数应该通过验证，但返回了错误:", err)
	}

	// 测试非整数的浮点数
	if err := validator.Validate(123.45); err == nil {
		t.Error("非整数的浮点数应该返回错误，但没有返回错误")
	}
}

// TestFloatValidator 测试浮点数验证器
func TestFloatValidator(t *testing.T) {
	validator := &FloatValidator{}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试空字符串
	if err := validator.Validate(""); err == nil {
		t.Error("空字符串应该返回错误，但没有返回错误")
	}

	// 测试无效浮点数字符串
	if err := validator.Validate("abc"); err == nil {
		t.Error("无效浮点数字符串应该返回错误，但没有返回错误")
	}

	// 测试有效浮点数字符串
	if err := validator.Validate("123.45"); err != nil {
		t.Error("有效浮点数字符串应该通过验证，但返回了错误:", err)
	}

	// 测试浮点数
	if err := validator.Validate(123.45); err != nil {
		t.Error("浮点数应该通过验证，但返回了错误:", err)
	}

	// 测试整数
	if err := validator.Validate(123); err != nil {
		t.Error("整数应该通过验证，但返回了错误:", err)
	}
}

// TestBooleanValidator 测试布尔值验证器
func TestBooleanValidator(t *testing.T) {
	validator := &BooleanValidator{}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试空字符串
	if err := validator.Validate(""); err == nil {
		t.Error("空字符串应该返回错误，但没有返回错误")
	}

	// 测试无效布尔字符串
	if err := validator.Validate("abc"); err == nil {
		t.Error("无效布尔字符串应该返回错误，但没有返回错误")
	}

	// 测试有效布尔字符串
	if err := validator.Validate("true"); err != nil {
		t.Error("有效布尔字符串应该通过验证，但返回了错误:", err)
	}

	if err := validator.Validate("false"); err != nil {
		t.Error("有效布尔字符串应该通过验证，但返回了错误:", err)
	}

	// 测试布尔值
	if err := validator.Validate(true); err != nil {
		t.Error("布尔值应该通过验证，但返回了错误:", err)
	}

	if err := validator.Validate(false); err != nil {
		t.Error("布尔值应该通过验证，但返回了错误:", err)
	}
}

// TestRegexValidator 测试正则表达式验证器
func TestRegexValidator(t *testing.T) {
	validator := &RegexValidator{Pattern: "^[a-z]+$"}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试空字符串
	if err := validator.Validate(""); err == nil {
		t.Error("空字符串应该返回错误，但没有返回错误")
	}

	// 测试不匹配的字符串
	if err := validator.Validate("ABC123"); err == nil {
		t.Error("不匹配的字符串应该返回错误，但没有返回错误")
	}

	// 测试匹配的字符串
	if err := validator.Validate("abc"); err != nil {
		t.Error("匹配的字符串应该通过验证，但返回了错误:", err)
	}
}

// TestNotRegexValidator 测试反向正则表达式验证器
func TestNotRegexValidator(t *testing.T) {
	validator := &NotRegexValidator{Pattern: "^[0-9]+$"}

	// 测试nil值
	if err := validator.Validate(nil); err != nil {
		t.Error("nil值应该跳过验证，但返回了错误:", err)
	}

	// 测试空字符串
	if err := validator.Validate(""); err == nil {
		t.Error("空字符串应该返回错误，但没有返回错误")
	}

	// 测试匹配的字符串
	if err := validator.Validate("123"); err == nil {
		t.Error("匹配的字符串应该返回错误，但没有返回错误")
	}

	// 测试不匹配的字符串
	if err := validator.Validate("abc"); err != nil {
		t.Error("不匹配的字符串应该通过验证，但返回了错误:", err)
	}
}
