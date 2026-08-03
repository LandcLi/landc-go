package tag

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Validator interface {
	Validate(value interface{}) error
}

// ValidationOptions 验证选项
type ValidationOptions struct {
	CaseInsensitive bool // 是否不区分大小写
	BailEarly       bool // 是否快速失败
	ForEach         bool // 是否对数组元素进行遍历验证
}

// WithOptions 使用选项创建验证器
func WithOptions(validator Validator, options ValidationOptions) Validator {
	return &optionsValidator{
		validator: validator,
		options:   options,
	}
}

// optionsValidator 带选项的验证器包装器
type optionsValidator struct {
	validator Validator
	options   ValidationOptions
}

func (v *optionsValidator) Validate(value interface{}) error {
	// 处理 foreach 选项
	if v.options.ForEach {
		if value == nil {
			return nil // 数组为 nil 时跳过验证
		}

		val := reflect.ValueOf(value)
		if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
			for i := 0; i < val.Len(); i++ {
				elem := val.Index(i).Interface()
				err := v.validator.Validate(elem)
				if err != nil {
					if v.options.BailEarly {
						return fmt.Errorf("element %d: %w", i, err)
					}
				}
				// 快速失败模式下直接返回
				if v.options.BailEarly {
					break
				}
			}
			return nil
		}
	}

	// 处理 ci 选项（目前主要影响字符串比较，后续在具体验证器中实现）
	return v.validator.Validate(value)
}

// RequiredValidator 必需性验证器
// 验证值是否为必需（不能为空、0、false或空数组）
// 注意：对于nil值、空字符串、空白字符串、0值、false值、空数组都会返回错误
type RequiredValidator struct{}

//nolint:gocyclo // 类型检查分派为线性 switch，拆分收益低
func (v *RequiredValidator) Validate(value interface{}) error {
	if value == nil {
		return fmt.Errorf("value is required")
	}

	switch val := value.(type) {
	case string:
		if strings.TrimSpace(val) == "" {
			return fmt.Errorf("value is required")
		}
	case []string:
		if len(val) == 0 {
			return fmt.Errorf("value is required")
		}
	case []interface{}:
		if len(val) == 0 {
			return fmt.Errorf("value is required")
		}
	case int:
		if val == 0 {
			return fmt.Errorf("value is required")
		}
	case int8:
		if val == 0 {
			return fmt.Errorf("value is required")
		}
	case int16:
		if val == 0 {
			return fmt.Errorf("value is required")
		}
	case int32:
		if val == 0 {
			return fmt.Errorf("value is required")
		}
	case int64:
		if val == 0 {
			return fmt.Errorf("value is required")
		}
	case uint:
		if val == 0 {
			return fmt.Errorf("value is required")
		}
	case uint8:
		if val == 0 {
			return fmt.Errorf("value is required")
		}
	case uint16:
		if val == 0 {
			return fmt.Errorf("value is required")
		}
	case uint32:
		if val == 0 {
			return fmt.Errorf("value is required")
		}
	case uint64:
		if val == 0 {
			return fmt.Errorf("value is required")
		}
	case float32:
		if val == 0 {
			return fmt.Errorf("value is required")
		}
	case float64:
		if val == 0 {
			return fmt.Errorf("value is required")
		}
	case bool:
		if !val {
			return fmt.Errorf("value is required")
		}
	}

	return nil
}

// RequiredIfValidator 条件必需性验证器
// 当指定字段等于某值时，当前字段必需
// 格式：required-if:field,value
type RequiredIfValidator struct {
	Field string
	Value interface{}
}

func (v *RequiredIfValidator) Validate(value interface{}) error {
	// 注意：此验证器需要访问其他字段的值
	// 实际使用时需要在结构体验证上下文中使用
	// 这里只实现基本结构，具体逻辑需要在结构体验证器中完善
	return nil
}

// RequiredUnlessValidator 条件必需性验证器（反向）
// 除非指定字段等于某值，否则当前字段必需
// 格式：required-unless:field,value
type RequiredUnlessValidator struct {
	Field string
	Value interface{}
}

func (v *RequiredUnlessValidator) Validate(value interface{}) error {
	// 注意：此验证器需要访问其他字段的值
	// 实际使用时需要在结构体验证上下文中使用
	return nil
}

// RequiredWithValidator 关联必需性验证器
// 当任意指定字段不为空时，当前字段必需
// 格式：required-with:field1,field2...
type RequiredWithValidator struct {
	Fields []string
}

func (v *RequiredWithValidator) Validate(value interface{}) error {
	// 注意：此验证器需要访问其他字段的值
	// 实际使用时需要在结构体验证上下文中使用
	return nil
}

// RequiredWithAllValidator 关联必需性验证器（全部）
// 当所有指定字段都不为空时，当前字段必需
// 格式：required-with-all:field1,field2...
type RequiredWithAllValidator struct {
	Fields []string
}

func (v *RequiredWithAllValidator) Validate(value interface{}) error {
	// 注意：此验证器需要访问其他字段的值
	// 实际使用时需要在结构体验证上下文中使用
	return nil
}

// RequiredWithoutValidator 反向关联必需性验证器
// 当任意指定字段为空时，当前字段必需
// 格式：required-without:field1,field2...
type RequiredWithoutValidator struct {
	Fields []string
}

func (v *RequiredWithoutValidator) Validate(value interface{}) error {
	// 注意：此验证器需要访问其他字段的值
	// 实际使用时需要在结构体验证上下文中使用
	return nil
}

// RequiredWithoutAllValidator 反向关联必需性验证器（全部）
// 当所有指定字段都为空时，当前字段必需
// 格式：required-without-all:field1,field2...
type RequiredWithoutAllValidator struct {
	Fields []string
}

func (v *RequiredWithoutAllValidator) Validate(value interface{}) error {
	// 注意：此验证器需要访问其他字段的值
	// 实际使用时需要在结构体验证上下文中使用
	return nil
}

type MinValidator struct {
	Min int
}

//nolint:gocyclo // 数值类型分派为线性 switch，拆分收益低
func (v *MinValidator) Validate(value interface{}) error {
	if value == nil {
		return nil
	}

	switch val := value.(type) {
	case int:
		if val < v.Min {
			return fmt.Errorf("value must be at least %d", v.Min)
		}
	case int8:
		if int(val) < v.Min {
			return fmt.Errorf("value must be at least %d", v.Min)
		}
	case int16:
		if int(val) < v.Min {
			return fmt.Errorf("value must be at least %d", v.Min)
		}
	case int32:
		if int(val) < v.Min {
			return fmt.Errorf("value must be at least %d", v.Min)
		}
	case int64:
		if int(val) < v.Min {
			return fmt.Errorf("value must be at least %d", v.Min)
		}
	case uint:
		if val < uint(v.Min) {
			return fmt.Errorf("value must be at least %d", v.Min)
		}
	case uint8:
		if v.Min < 0 || val < uint8(v.Min) { //nolint:gosec // 左侧短路保护已排除负值
			return fmt.Errorf("value must be at least %d", v.Min)
		}
	case uint16:
		if v.Min < 0 || val < uint16(v.Min) { //nolint:gosec // 左侧短路保护已排除负值
			return fmt.Errorf("value must be at least %d", v.Min)
		}
	case uint32:
		if v.Min < 0 || val < uint32(v.Min) { //nolint:gosec // 左侧短路保护已排除负值
			return fmt.Errorf("value must be at least %d", v.Min)
		}
	case uint64:
		if v.Min < 0 || val < uint64(v.Min) { //nolint:gosec // 左侧短路保护已排除负值
			return fmt.Errorf("value must be at least %d", v.Min)
		}
	case float32:
		if float64(val) < float64(v.Min) {
			return fmt.Errorf("value must be at least %d", v.Min)
		}
	case float64:
		if val < float64(v.Min) {
			return fmt.Errorf("value must be at least %d", v.Min)
		}
	case string:
		if len(val) < v.Min {
			return fmt.Errorf("value length must be at least %d", v.Min)
		}
	}

	return nil
}

type MaxValidator struct {
	Max int
}

//nolint:gocyclo // 数值类型分派为线性 switch，拆分收益低
func (v *MaxValidator) Validate(value interface{}) error {
	if value == nil {
		return nil
	}

	switch val := value.(type) {
	case int:
		if val > v.Max {
			return fmt.Errorf("value must be at most %d", v.Max)
		}
	case int8:
		if int(val) > v.Max {
			return fmt.Errorf("value must be at most %d", v.Max)
		}
	case int16:
		if int(val) > v.Max {
			return fmt.Errorf("value must be at most %d", v.Max)
		}
	case int32:
		if int(val) > v.Max {
			return fmt.Errorf("value must be at most %d", v.Max)
		}
	case int64:
		if int(val) > v.Max {
			return fmt.Errorf("value must be at most %d", v.Max)
		}
	case uint:
		if val > uint(v.Max) {
			return fmt.Errorf("value must be at most %d", v.Max)
		}
	case uint8:
		if v.Max < 0 || val > uint8(v.Max) { //nolint:gosec // 左侧短路保护已排除负值
			return fmt.Errorf("value must be at most %d", v.Max)
		}
	case uint16:
		if v.Max < 0 || val > uint16(v.Max) { //nolint:gosec // 左侧短路保护已排除负值
			return fmt.Errorf("value must be at most %d", v.Max)
		}
	case uint32:
		if v.Max < 0 || val > uint32(v.Max) { //nolint:gosec // 左侧短路保护已排除负值
			return fmt.Errorf("value must be at most %d", v.Max)
		}
	case uint64:
		if v.Max < 0 || val > uint64(v.Max) { //nolint:gosec // 左侧短路保护已排除负值
			return fmt.Errorf("value must be at most %d", v.Max)
		}
	case float32:
		if float64(val) > float64(v.Max) {
			return fmt.Errorf("value must be at most %d", v.Max)
		}
	case float64:
		if val > float64(v.Max) {
			return fmt.Errorf("value must be at most %d", v.Max)
		}
	case string:
		if len(val) > v.Max {
			return fmt.Errorf("value length must be at most %d", v.Max)
		}
	}

	return nil
}

type LengthValidator struct {
	Length int
}

func (v *LengthValidator) Validate(value interface{}) error {
	if value == nil {
		return nil
	}

	switch val := value.(type) {
	case string:
		if len(val) != v.Length {
			return fmt.Errorf("value length must be exactly %d", v.Length)
		}
	case []string:
		if len(val) != v.Length {
			return fmt.Errorf("array length must be exactly %d", v.Length)
		}
	}

	return nil
}

type PatternValidator struct {
	Pattern string
	regex   *regexp.Regexp
}

//nolint:gocritic // type switch 单 case，改写收益低
func (v *PatternValidator) Validate(value interface{}) error {
	if value == nil {
		return nil
	}

	if v.regex == nil {
		v.regex = regexp.MustCompile(v.Pattern)
	}

	switch val := value.(type) {
	case string:
		if !v.regex.MatchString(val) {
			return fmt.Errorf("value does not match pattern %s", v.Pattern)
		}
	}

	return nil
}

// DateValidator 日期验证器
// 验证值是否为有效的日期格式
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type DateValidator struct{}

func (v *DateValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid date") // 空字符串返回错误
		}
		if !IsValidDate(val) {
			return fmt.Errorf("value must be a valid date")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// DateTimeValidator 日期时间验证器
// 验证值是否为有效的日期时间格式
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type DateTimeValidator struct{}

func (v *DateTimeValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid datetime") // 空字符串返回错误
		}
		if !IsValidDateTime(val) {
			return fmt.Errorf("value must be a valid datetime")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// DateFormatValidator 指定格式日期验证器
// 验证值是否为指定格式的日期
// 格式：date-format:format
type DateFormatValidator struct {
	Format string
}

func (v *DateFormatValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid date") // 空字符串返回错误
		}
		_, err := time.Parse(v.Format, val)
		if err != nil {
			return fmt.Errorf("value must be a valid date in format %s", v.Format)
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// BeforeValidator 时间比较验证器
// 验证值是否在指定字段之前
// 格式：before:field
type BeforeValidator struct {
	Field string
}

func (v *BeforeValidator) Validate(value interface{}) error {
	// 注意：此验证器需要访问其他字段的值
	// 实际使用时需要在结构体验证上下文中使用
	return nil
}

// BeforeEqualValidator 时间比较验证器（包含相等）
// 验证值是否在指定字段之前或相等
// 格式：before-equal:field
type BeforeEqualValidator struct {
	Field string
}

func (v *BeforeEqualValidator) Validate(value interface{}) error {
	// 注意：此验证器需要访问其他字段的值
	// 实际使用时需要在结构体验证上下文中使用
	return nil
}

// AfterValidator 时间比较验证器
// 验证值是否在指定字段之后
// 格式：after:field
type AfterValidator struct {
	Field string
}

func (v *AfterValidator) Validate(value interface{}) error {
	// 注意：此验证器需要访问其他字段的值
	// 实际使用时需要在结构体验证上下文中使用
	return nil
}

// AfterEqualValidator 时间比较验证器（包含相等）
// 验证值是否在指定字段之后或相等
// 格式：after-equal:field
type AfterEqualValidator struct {
	Field string
}

func (v *AfterEqualValidator) Validate(value interface{}) error {
	// 注意：此验证器需要访问其他字段的值
	// 实际使用时需要在结构体验证上下文中使用
	return nil
}

type FormatValidator struct {
	Format string
}

//nolint:gocritic,gocyclo // type switch 内含嵌套 switch；格式校验分派为线性 switch
func (v *FormatValidator) Validate(value interface{}) error {
	if value == nil {
		return nil
	}

	switch val := value.(type) {
	case string:
		switch v.Format {
		case "email":
			if !IsValidEmail(val) {
				return fmt.Errorf("value must be a valid email address")
			}
		case "url":
			if !IsValidURL(val) {
				return fmt.Errorf("value must be a valid URL")
			}
		case "phone":
			if !IsValidPhone(val) {
				return fmt.Errorf("value must be a valid phone number")
			}
		case "date":
			if !IsValidDate(val) {
				return fmt.Errorf("value must be a valid date")
			}
		case "datetime":
			if !IsValidDateTime(val) {
				return fmt.Errorf("value must be a valid datetime")
			}
		case "uuid":
			if !IsValidUUID(val) {
				return fmt.Errorf("value must be a valid UUID")
			}
		case "ipv4":
			if !IsValidIPv4(val) {
				return fmt.Errorf("value must be a valid IPv4 address")
			}
		case "ipv6":
			if !IsValidIPv6(val) {
				return fmt.Errorf("value must be a valid IPv6 address")
			}
		case "ip":
			if !IsValidIP(val) {
				return fmt.Errorf("value must be a valid IP address")
			}
		}
	}

	return nil
}

// ArrayValidator 数组验证器
// 验证值是否为有效的数组
// 注意：对于nil值会返回错误，对于空数组会返回错误
type ArrayValidator struct{}

func (v *ArrayValidator) Validate(value interface{}) error {
	if value == nil {
		return fmt.Errorf("value must be an array") // nil值返回错误
	}

	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return fmt.Errorf("value must be an array")
	}

	if val.Len() == 0 {
		return fmt.Errorf("array cannot be empty") // 空数组返回错误
	}

	return nil
}

// EnumsValidator 枚举值验证器
// 验证值是否在指定的枚举值列表中
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type EnumsValidator struct {
	Values []string
}

func (v *EnumsValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be one of: %s", strings.Join(v.Values, ", ")) // 空字符串返回错误
		}
		for _, validValue := range v.Values {
			if val == validValue {
				return nil
			}
		}
		return fmt.Errorf("value must be one of: %s", strings.Join(v.Values, ", "))
	} else {
		return fmt.Errorf("value must be a string")
	}
}

// EmailValidator 邮箱地址验证器
// 验证值是否为有效的邮箱地址
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type EmailValidator struct{}

func (v *EmailValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid email address") // 空字符串返回错误
		}
		if !IsValidEmail(val) {
			return fmt.Errorf("value must be a valid email address")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// PhoneValidator 手机号验证器
// 验证值是否为有效的手机号
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type PhoneValidator struct{}

func (v *PhoneValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid phone number") // 空字符串返回错误
		}
		if !IsValidPhone(val) {
			return fmt.Errorf("value must be a valid phone number")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// PhoneLooseValidator 宽松手机号验证器
// 验证值是否为有效的手机号（宽松格式）
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type PhoneLooseValidator struct{}

func (v *PhoneLooseValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid phone number") // 空字符串返回错误
		}
		// 宽松验证：只要包含数字即可
		numPattern := `\d+`
		numRegex := regexp.MustCompile(numPattern)
		if !numRegex.MatchString(val) {
			return fmt.Errorf("value must be a valid phone number")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// TelephoneValidator 座机电话验证器
// 验证值是否为有效的座机电话号码
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type TelephoneValidator struct{}

func (v *TelephoneValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid telephone number") // 空字符串返回错误
		}
		// 座机电话格式：区号-电话号码
		pattern := `^\d{3,4}-?\d{7,8}$`
		regex := regexp.MustCompile(pattern)
		if !regex.MatchString(val) {
			return fmt.Errorf("value must be a valid telephone number")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// PassportValidator 通用账号验证器
// 验证值是否为有效的通用账号（字母开头，6-18位）
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type PassportValidator struct{}

func (v *PassportValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid passport") // 空字符串返回错误
		}
		// 通用账号格式：字母开头，6-18位
		pattern := `^[a-zA-Z][a-zA-Z0-9]{5,17}$`
		regex := regexp.MustCompile(pattern)
		if !regex.MatchString(val) {
			return fmt.Errorf("value must be a valid passport")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// PasswordValidator 密码验证器
// 验证值是否为有效的密码（6-18位可见字符）
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type PasswordValidator struct{}

func (v *PasswordValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid password") // 空字符串返回错误
		}
		if len(val) < 6 || len(val) > 18 {
			return fmt.Errorf("password length must be between 6 and 18")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// Password2Validator 中级密码验证器
// 验证值是否为有效的中级密码（含大小写字母和数字）
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type Password2Validator struct{}

func (v *Password2Validator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid password") // 空字符串返回错误
		}
		if len(val) < 6 || len(val) > 18 {
			return fmt.Errorf("password length must be between 6 and 18")
		}
		// 包含大小写字母和数字
		hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(val)
		hasLower := regexp.MustCompile(`[a-z]`).MatchString(val)
		hasDigit := regexp.MustCompile(`\d`).MatchString(val)
		if !hasUpper || !hasLower || !hasDigit {
			return fmt.Errorf("password must contain uppercase letters, lowercase letters and numbers")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// Password3Validator 强密码验证器
// 验证值是否为有效的强密码（含大小写字母、数字和特殊字符）
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type Password3Validator struct{}

func (v *Password3Validator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid password") // 空字符串返回错误
		}
		if len(val) < 6 || len(val) > 18 {
			return fmt.Errorf("password length must be between 6 and 18")
		}
		// 包含大小写字母、数字和特殊字符
		hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(val)
		hasLower := regexp.MustCompile(`[a-z]`).MatchString(val)
		hasDigit := regexp.MustCompile(`\d`).MatchString(val)
		hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>/?]`).MatchString(val)
		if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
			return fmt.Errorf("password must contain uppercase letters, lowercase letters, numbers and special characters")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// PostcodeValidator 邮政编码验证器
// 验证值是否为有效的邮政编码
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type PostcodeValidator struct{}

func (v *PostcodeValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid postcode") // 空字符串返回错误
		}
		// 邮政编码格式：6位数字
		pattern := `^\d{6}$`
		regex := regexp.MustCompile(pattern)
		if !regex.MatchString(val) {
			return fmt.Errorf("value must be a valid postcode")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// ResidentIDValidator 身份证号码验证器
// 验证值是否为有效的身份证号码
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type ResidentIDValidator struct{}

func (v *ResidentIDValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid resident ID") // 空字符串返回错误
		}
		// 身份证号码格式：18位，最后一位可能是X
		pattern := `^[1-9]\d{5}(19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$`
		regex := regexp.MustCompile(pattern)
		if !regex.MatchString(val) {
			return fmt.Errorf("value must be a valid resident ID")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// BankCardValidator 银行卡号验证器
// 验证值是否为有效的银行卡号
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type BankCardValidator struct{}

func (v *BankCardValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid bank card number") // 空字符串返回错误
		}
		// 银行卡号格式：16-19位数字
		pattern := `^\d{16,19}$`
		regex := regexp.MustCompile(pattern)
		if !regex.MatchString(val) {
			return fmt.Errorf("value must be a valid bank card number")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// QQValidator QQ号码验证器
// 验证值是否为有效的QQ号码
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type QQValidator struct{}

func (v *QQValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid QQ number") // 空字符串返回错误
		}
		// QQ号码格式：5-13位数字
		pattern := `^[1-9]\d{4,12}$`
		regex := regexp.MustCompile(pattern)
		if !regex.MatchString(val) {
			return fmt.Errorf("value must be a valid QQ number")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

type EnumValidator struct {
	Values []string
}

//nolint:gocritic // type switch 单 case，改写需调整闭合结构，收益低
func (v *EnumValidator) Validate(value interface{}) error {
	if value == nil {
		return nil
	}

	switch val := value.(type) {
	case string:
		for _, validValue := range v.Values {
			if val == validValue {
				return nil
			}
		}
		return fmt.Errorf("value must be one of: %s", strings.Join(v.Values, ", "))
	}

	return nil
}

func IsValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(email)
}

func IsValidURL(url string) bool {
	pattern := `^https?://[^\s/$.?#].[^\s]*$`
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(url)
}

func IsValidPhone(phone string) bool {
	pattern := `^[\d\s\-\+\(\)]+$`
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(phone) && len(strings.ReplaceAll(phone, `[^0-9]`, "")) >= 10
}

func IsValidDate(date string) bool {
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

func IsValidDateTime(datetime string) bool {
	_, err := time.Parse(time.RFC3339, datetime)
	if err == nil {
		return true
	}
	_, err = time.Parse("2006-01-02 15:04:05", datetime)
	return err == nil
}

func IsValidUUID(uuid string) bool {
	pattern := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(uuid)
}

func IsValidIPv4(ip string) bool {
	pattern := `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(ip)
}

func IsValidIPv6(ip string) bool {
	pattern := `^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(ip)
}

func IsValidIP(ip string) bool {
	return IsValidIPv4(ip) || IsValidIPv6(ip)
}

// IPValidator IP地址验证器
// 验证值是否为有效的IPv4或IPv6地址
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type IPValidator struct{}

func (v *IPValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid IP address") // 空字符串返回错误
		}
		if !IsValidIP(val) {
			return fmt.Errorf("value must be a valid IP address")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// IPv4Validator IPv4地址验证器
// 验证值是否为有效的IPv4地址
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type IPv4Validator struct{}

func (v *IPv4Validator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid IPv4 address") // 空字符串返回错误
		}
		if !IsValidIPv4(val) {
			return fmt.Errorf("value must be a valid IPv4 address")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// IPv6Validator IPv6地址验证器
// 验证值是否为有效的IPv6地址
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type IPv6Validator struct{}

func (v *IPv6Validator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid IPv6 address") // 空字符串返回错误
		}
		if !IsValidIPv6(val) {
			return fmt.Errorf("value must be a valid IPv6 address")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// MACValidator MAC地址验证器
// 验证值是否为有效的MAC地址
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type MACValidator struct{}

func (v *MACValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid MAC address") // 空字符串返回错误
		}
		// MAC地址格式：XX-XX-XX-XX-XX-XX 或 XX:XX:XX:XX:XX:XX
		pattern := `^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`
		regex := regexp.MustCompile(pattern)
		if !regex.MatchString(val) {
			return fmt.Errorf("value must be a valid MAC address")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// URLValidator URL地址验证器
// 验证值是否为有效的URL地址
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type URLValidator struct{}

func (v *URLValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid URL") // 空字符串返回错误
		}
		if !IsValidURL(val) {
			return fmt.Errorf("value must be a valid URL")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// DomainValidator 域名验证器
// 验证值是否为有效的域名
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type DomainValidator struct{}

func (v *DomainValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid domain") // 空字符串返回错误
		}
		// 域名格式：字母数字开头，包含字母、数字、-、.，结尾为字母数字
		pattern := `^[a-zA-Z0-9][a-zA-Z0-9\-\.]*[a-zA-Z0-9]$`
		regex := regexp.MustCompile(pattern)
		if !regex.MatchString(val) {
			return fmt.Errorf("value must be a valid domain")
		}
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

func ParseMin(tag *Tag) (int, error) {
	if tag == nil || tag.Value == "" {
		return 0, fmt.Errorf("min value not found in tag")
	}
	return strconv.Atoi(tag.Value)
}

func ParseMax(tag *Tag) (int, error) {
	if tag == nil || tag.Value == "" {
		return 0, fmt.Errorf("max value not found in tag")
	}
	return strconv.Atoi(tag.Value)
}

func ParseLength(tag *Tag) (int, error) {
	if tag == nil || tag.Value == "" {
		return 0, fmt.Errorf("length value not found in tag")
	}
	return strconv.Atoi(tag.Value)
}

func ParsePattern(tag *Tag) (string, error) {
	if tag == nil || tag.Value == "" {
		return "", fmt.Errorf("pattern value not found in tag")
	}
	return tag.Value, nil
}

func ParseFormat(tag *Tag) (string, error) {
	if tag == nil || tag.Value == "" {
		return "", fmt.Errorf("format value not found in tag")
	}
	return tag.Value, nil
}

func ParseEnum(tag *Tag) ([]string, error) {
	if tag == nil || tag.Value == "" {
		return nil, fmt.Errorf("enum values not found in tag")
	}
	return strings.Split(tag.Value, ","), nil
}

// SizeValidator 大小验证器
// 验证数组长度或字符串长度
// 注意：对于nil值会跳过验证，对于空字符串会按长度0处理
type SizeValidator struct {
	Size int
}

func (v *SizeValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	switch val := value.(type) {
	case string:
		if len(val) != v.Size {
			return fmt.Errorf("value length must be exactly %d", v.Size)
		}
	case []interface{}:
		if len(val) != v.Size {
			return fmt.Errorf("array length must be exactly %d", v.Size)
		}
	default:
		// 尝试使用反射获取长度
		reflectVal := reflect.ValueOf(value)
		switch reflectVal.Kind() {
		case reflect.Slice, reflect.Array:
			if reflectVal.Len() != v.Size {
				return fmt.Errorf("array length must be exactly %d", v.Size)
			}
		default:
			return fmt.Errorf("size validation only applies to strings and arrays")
		}
	}

	return nil
}

// BetweenValidator 范围验证器
// 验证值是否在指定范围内
// 注意：对于nil值会跳过验证，对于空字符串会按长度0处理
type BetweenValidator struct {
	Min int
	Max int
}

func (v *BetweenValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	switch val := value.(type) {
	case string:
		length := len(val)
		if length < v.Min || length > v.Max {
			return fmt.Errorf("value length must be between %d and %d", v.Min, v.Max)
		}
	case int:
		if val < v.Min || val > v.Max {
			return fmt.Errorf("value must be between %d and %d", v.Min, v.Max)
		}
	case float64:
		if val < float64(v.Min) || val > float64(v.Max) {
			return fmt.Errorf("value must be between %d and %d", v.Min, v.Max)
		}
	default:
		// 尝试使用反射获取长度
		reflectVal := reflect.ValueOf(value)
		switch reflectVal.Kind() {
		case reflect.Slice, reflect.Array:
			length := reflectVal.Len()
			if length < v.Min || length > v.Max {
				return fmt.Errorf("array length must be between %d and %d", v.Min, v.Max)
			}
		default:
			return fmt.Errorf("between validation applies to strings, numbers, and arrays")
		}
	}

	return nil
}

// InValidator 包含验证器
// 验证值是否在指定列表中
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type InValidator struct {
	Values []string
}

func (v *InValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be one of: %s", strings.Join(v.Values, ", ")) // 空字符串返回错误
		}
		for _, validValue := range v.Values {
			if val == validValue {
				return nil
			}
		}
		return fmt.Errorf("value must be one of: %s", strings.Join(v.Values, ", "))
	} else {
		return fmt.Errorf("in validation only applies to strings")
	}
}

// NotInValidator 不包含验证器
// 验证值是否不在指定列表中
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type NotInValidator struct {
	Values []string
}

func (v *NotInValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must not be one of: %s", strings.Join(v.Values, ", ")) // 空字符串返回错误
		}
		for _, invalidValue := range v.Values {
			if val == invalidValue {
				return fmt.Errorf("value must not be one of: %s", strings.Join(v.Values, ", "))
			}
		}
	} else {
		return fmt.Errorf("not-in validation only applies to strings")
	}

	return nil
}

func ParseDefault(tag *Tag) (interface{}, error) {
	if tag == nil || tag.Value == "" {
		return nil, fmt.Errorf("default value not found in tag")
	}
	return tag.Value, nil
}

// JSONValidator JSON格式验证器
// 验证值是否为有效的JSON格式
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type JSONValidator struct{}

func (v *JSONValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must be a valid JSON") // 空字符串返回错误
		}
		var jsonData interface{}
		err := json.Unmarshal([]byte(val), &jsonData)
		if err != nil {
			return fmt.Errorf("value must be a valid JSON")
		}
	} else {
		return fmt.Errorf("json validation only applies to strings")
	}

	return nil
}

// IntegerValidator 整数验证器
// 验证值是否为有效的整数
// 注意：对于nil值会跳过验证
type IntegerValidator struct{}

func (v *IntegerValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	switch val := value.(type) {
	case int:
		// 已经是整数，验证通过
	case int8:
		// 已经是整数，验证通过
	case int16:
		// 已经是整数，验证通过
	case int32:
		// 已经是整数，验证通过
	case int64:
		// 已经是整数，验证通过
	case uint:
		// 已经是整数，验证通过
	case uint8:
		// 已经是整数，验证通过
	case uint16:
		// 已经是整数，验证通过
	case uint32:
		// 已经是整数，验证通过
	case uint64:
		// 已经是整数，验证通过
	case string:
		if val == "" {
			return fmt.Errorf("value must be a valid integer") // 空字符串返回错误
		}
		_, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("value must be a valid integer")
		}
	case float64:
		// 检查是否为整数（小数部分为0）
		if val != float64(int(val)) {
			return fmt.Errorf("value must be a valid integer")
		}
	default:
		return fmt.Errorf("integer validation applies to numbers and strings")
	}

	return nil
}

// FloatValidator 浮点数验证器
// 验证值是否为有效的浮点数
// 注意：对于nil值会跳过验证
type FloatValidator struct{}

func (v *FloatValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	switch val := value.(type) {
	case float32:
		// 已经是浮点数，验证通过
	case float64:
		// 已经是浮点数，验证通过
	case string:
		if val == "" {
			return fmt.Errorf("value must be a valid float") // 空字符串返回错误
		}
		_, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return fmt.Errorf("value must be a valid float")
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// 整数可以转换为浮点数，验证通过
	default:
		return fmt.Errorf("float validation applies to numbers and strings")
	}

	return nil
}

// BooleanValidator 布尔值验证器
// 验证值是否为有效的布尔值
// 注意：对于nil值会跳过验证
type BooleanValidator struct{}

func (v *BooleanValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	switch val := value.(type) {
	case bool:
		// 已经是布尔值，验证通过
	case string:
		if val == "" {
			return fmt.Errorf("value must be a valid boolean") // 空字符串返回错误
		}
		// 检查是否为有效的布尔字符串
		lowerVal := strings.ToLower(val)
		if lowerVal != "true" && lowerVal != "false" {
			return fmt.Errorf("value must be a valid boolean")
		}
	default:
		return fmt.Errorf("boolean validation applies to booleans and strings")
	}

	return nil
}

// StringValidator 字符串验证器
// 验证值是否为有效的字符串
// 注意：对于nil值会跳过验证，对于空字符串会通过验证
type StringValidator struct{}

func (v *StringValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if _, ok := value.(string); ok {
		// 已经是字符串，验证通过（包括空字符串）
	} else {
		return fmt.Errorf("value must be a string")
	}

	return nil
}

// NumberValidator 数字验证器
// 验证值是否为有效的数字
// 注意：对于nil值会跳过验证
type NumberValidator struct{}

func (v *NumberValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	switch val := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// 已经是整数，验证通过
	case float32, float64:
		// 已经是浮点数，验证通过
	case string:
		if val == "" {
			return fmt.Errorf("value must be a valid number") // 空字符串返回错误
		}
		// 尝试解析为浮点数
		_, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return fmt.Errorf("value must be a valid number")
		}
	default:
		return fmt.Errorf("number validation applies to numbers and strings")
	}

	return nil
}

func CreateRequiredValidator() Validator {
	return &RequiredValidator{}
}

// SameValidator 相同验证器
// 验证值是否与指定字段相同
// 格式：same:field
// 注意：此验证器需要访问其他字段的值，实际使用时需要在结构体验证上下文中使用
type SameValidator struct {
	Field string
}

func (v *SameValidator) Validate(value interface{}) error {
	// 注意：此验证器需要访问其他字段的值
	// 实际使用时需要在结构体验证上下文中使用
	return nil
}

// DifferentValidator 不同验证器
// 验证值是否与指定字段不同
// 格式：different:field
// 注意：此验证器需要访问其他字段的值，实际使用时需要在结构体验证上下文中使用
type DifferentValidator struct {
	Field string
}

func (v *DifferentValidator) Validate(value interface{}) error {
	// 注意：此验证器需要访问其他字段的值
	// 实际使用时需要在结构体验证上下文中使用
	return nil
}

func CreateMinValidator(minValue int) Validator {
	return &MinValidator{Min: minValue}
}

// RegexValidator 正则表达式验证器
// 验证值是否匹配指定的正则表达式
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type RegexValidator struct {
	Pattern string
}

func (v *RegexValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must match pattern %s", v.Pattern) // 空字符串返回错误
		}
		regex := regexp.MustCompile(v.Pattern)
		if !regex.MatchString(val) {
			return fmt.Errorf("value must match pattern %s", v.Pattern)
		}
	} else {
		return fmt.Errorf("regex validation only applies to strings")
	}

	return nil
}

// NotRegexValidator 反向正则表达式验证器
// 验证值是否不匹配指定的正则表达式
// 注意：对于nil值会跳过验证，对于空字符串会返回错误
type NotRegexValidator struct {
	Pattern string
}

func (v *NotRegexValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // nil值跳过验证
	}

	if val, ok := value.(string); ok {
		if val == "" {
			return fmt.Errorf("value must not match pattern %s", v.Pattern) // 空字符串返回错误
		}
		regex := regexp.MustCompile(v.Pattern)
		if regex.MatchString(val) {
			return fmt.Errorf("value must not match pattern %s", v.Pattern)
		}
	} else {
		return fmt.Errorf("not-regex validation only applies to strings")
	}

	return nil
}

func CreateMaxValidator(maxValue int) Validator {
	return &MaxValidator{Max: maxValue}
}

func CreateLengthValidator(length int) Validator {
	return &LengthValidator{Length: length}
}

func CreatePatternValidator(pattern string) Validator {
	return &PatternValidator{Pattern: pattern}
}

func CreateFormatValidator(format string) Validator {
	return &FormatValidator{Format: format}
}

func CreateEnumValidator(values ...string) Validator {
	return &EnumValidator{Values: values}
}
