package str

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mozillazg/go-pinyin"
)

// IsEmpty 判断字符串是否为空
func IsEmpty(s string) bool {
	return s == ""
}

// IsNotEmpty 判断字符串是否不为空
func IsNotEmpty(s string) bool {
	return !IsEmpty(s)
}

// IsBlank 判断字符串是否为空白
func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotBlank 判断字符串是否不为空白
func IsNotBlank(s string) bool {
	return !IsBlank(s)
}

// Length 获取字符串长度（按Unicode字符计算）
func Length(s string) int {
	return utf8.RuneCountInString(s)
}

// SubString 截取字符串
func SubString(s string, start, end int) string {
	runes := []rune(s)
	runeLen := len(runes)

	if start < 0 {
		start = 0
	}
	if end > runeLen {
		end = runeLen
	}
	if start > end {
		return ""
	}

	return string(runes[start:end])
}

// SubStringStart 从指定位置开始截取字符串到末尾
func SubStringStart(s string, start int) string {
	return SubString(s, start, Length(s))
}

// Concat 拼接字符串
func Concat(strs ...string) string {
	return strings.Join(strs, "")
}

// Replace 替换字符串
func Replace(s, old, newValue string, n int) string {
	return strings.Replace(s, old, newValue, n)
}

// ReplaceAll 替换所有匹配的字符串
func ReplaceAll(s, old, newValue string) string {
	return strings.ReplaceAll(s, old, newValue)
}

// Split 分割字符串
func Split(s, sep string) []string {
	return strings.Split(s, sep)
}

// SplitTrim 分割字符串并去除首尾空格
func SplitTrim(s, sep string) []string {
	parts := Split(s, sep)
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

// ToUpper 转换为大写
func ToUpper(s string) string {
	return strings.ToUpper(s)
}

// ToLower 转换为小写
func ToLower(s string) string {
	return strings.ToLower(s)
}

// ToTitle 将每个单词的首字母大写（Unicode 安全，替代已废弃的 strings.Title）
func ToTitle(s string) string {
	atWordStart := true
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			atWordStart = true
			return r
		}
		if atWordStart {
			atWordStart = false
			return unicode.ToTitle(r)
		}
		return r
	}, s)
}

// Trim 去除首尾空格
func Trim(s string) string {
	return strings.TrimSpace(s)
}

// TrimLeft 去除左侧空格
func TrimLeft(s string) string {
	return strings.TrimLeft(s, " \t\n\r\f")
}

// TrimRight 去除右侧空格
func TrimRight(s string) string {
	return strings.TrimRight(s, " \t\n\r\f")
}

// Contains 判断字符串是否包含子串
func Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// StartsWith 判断字符串是否以指定前缀开始
func StartsWith(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// EndsWith 判断字符串是否以指定后缀结束
func EndsWith(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

// IndexOf 获取子串第一次出现的位置
func IndexOf(s, substr string) int {
	return strings.Index(s, substr)
}

// LastIndexOf 获取子串最后一次出现的位置
func LastIndexOf(s, substr string) int {
	return strings.LastIndex(s, substr)
}

// Repeat 重复字符串
func Repeat(s string, count int) string {
	return strings.Repeat(s, count)
}

// Reverse 反转字符串
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// PadLeft 左侧填充
func PadLeft(s string, length int, padStr string) string {
	if padStr == "" {
		return s
	}
	runeCount := Length(s)
	if runeCount >= length {
		return s
	}
	paddingLen := length - runeCount
	padding := strings.Repeat(padStr, (paddingLen+len(padStr)-1)/len(padStr))[:paddingLen]
	return padding + s
}

// PadRight 右侧填充
func PadRight(s string, length int, padStr string) string {
	if padStr == "" {
		return s
	}
	runeCount := Length(s)
	if runeCount >= length {
		return s
	}
	paddingLen := length - runeCount
	padding := strings.Repeat(padStr, (paddingLen+len(padStr)-1)/len(padStr))[:paddingLen]
	return s + padding
}

// PadCenter 居中填充
func PadCenter(s string, length int, padStr string) string {
	if padStr == "" {
		return s
	}
	runeCount := Length(s)
	if runeCount >= length {
		return s
	}
	paddingLen := length - runeCount
	leftPaddingLen := paddingLen / 2
	rightPaddingLen := paddingLen - leftPaddingLen
	leftPadding := strings.Repeat(padStr, (leftPaddingLen+len(padStr)-1)/len(padStr))[:leftPaddingLen]
	rightPadding := strings.Repeat(padStr, (rightPaddingLen+len(padStr)-1)/len(padStr))[:rightPaddingLen]
	return leftPadding + s + rightPadding
}

// Matches 正则匹配
func Matches(pattern, s string) bool {
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}

// ReplaceAllFunc 使用函数替换
func ReplaceAllFunc(s string, repl func(string) string, pattern string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllStringFunc(s, repl)
}

// ReplaceFirst 替换第一个匹配项
func ReplaceFirst(s, old, newValue string) string {
	return strings.Replace(s, old, newValue, 1)
}

// ToCamelCase 转换为驼峰命名
func ToCamelCase(s string) string {
	var result []rune
	capitalizeNext := false

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if capitalizeNext {
				result = append(result, unicode.ToUpper(r))
				capitalizeNext = false
			} else {
				result = append(result, unicode.ToLower(r))
			}
		} else {
			capitalizeNext = true
		}
	}

	if len(result) > 0 {
		result[0] = unicode.ToLower(result[0])
	}

	return string(result)
}

// ToSnakeCase 转换为蛇形命名
func ToSnakeCase(s string) string {
	runes := []rune(s)
	var result []rune

	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLetter(runes[i-1]) || unicode.IsDigit(runes[i-1])) {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result = append(result, unicode.ToLower(r))
		} else if len(result) > 0 && result[len(result)-1] != '_' {
			result = append(result, '_')
		}
	}

	if len(result) > 0 && result[len(result)-1] == '_' {
		result = result[:len(result)-1]
	}

	return string(result)
}

// ToKebabCase 转换为短横线命名
func ToKebabCase(s string) string {
	runes := []rune(s)
	var result []rune

	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLetter(runes[i-1]) || unicode.IsDigit(runes[i-1])) {
				result = append(result, '-')
			}
			result = append(result, unicode.ToLower(r))
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result = append(result, unicode.ToLower(r))
		} else if len(result) > 0 && result[len(result)-1] != '-' {
			result = append(result, '-')
		}
	}

	if len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}

	return string(result)
}

// ReverseWords 反转单词顺序
func ReverseWords(s string) string {
	words := strings.Fields(s)
	for i, j := 0, len(words)-1; i < j; i, j = i+1, j-1 {
		words[i], words[j] = words[j], words[i]
	}
	return strings.Join(words, " ")
}

// SplitN 分割指定次数
func SplitN(s, sep string, n int) []string {
	return strings.SplitN(s, sep, n)
}

// SplitAfter 分割后保留分隔符
func SplitAfter(s, sep string) []string {
	return strings.SplitAfter(s, sep)
}

// SplitAfterN 分割后保留分隔符，指定次数
func SplitAfterN(s, sep string, n int) []string {
	return strings.SplitAfterN(s, sep, n)
}

// SubStringBetween 提取两个字符串之间的内容
func SubStringBetween(s, start, end string) string {
	startIdx := strings.Index(s, start)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(start)
	endIdx := strings.Index(s[startIdx:], end)
	if endIdx == -1 {
		return ""
	}
	return s[startIdx : startIdx+endIdx]
}

// SubStringBefore 提取指定字符串之前的内容
func SubStringBefore(s, sep string) string {
	idx := strings.Index(s, sep)
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// SubStringAfter 提取指定字符串之后的内容
func SubStringAfter(s, sep string) string {
	idx := strings.Index(s, sep)
	if idx == -1 {
		return ""
	}
	return s[idx+len(sep):]
}

// Count 统计子串出现次数
func Count(s, substr string) int {
	return strings.Count(s, substr)
}

// CountChars 统计字符出现次数
func CountChars(s string, char rune) int {
	count := 0
	for _, r := range s {
		if r == char {
			count++
		}
	}
	return count
}

// CountWords 统计单词数量
func CountWords(s string) int {
	return len(strings.Fields(s))
}

// IsNumeric 判断是否为数字
func IsNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

// IsAlphabetic 判断是否为字母
func IsAlphabetic(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return s != ""
}

// IsAlphanumeric 判断是否为字母数字
func IsAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

// IsChinese 判断是否包含中文
func IsChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// IsEmail 判断是否为邮箱
func IsEmail(s string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	return Matches(pattern, s)
}

// IsURL 判断是否为URL
func IsURL(s string) bool {
	pattern := `^(https?:\/\/)?([\da-z.-]+)\.([a-z.]{2,6})([/\w .-]*)*\/?$`
	return Matches(pattern, s)
}

// Truncate 截断字符串
func Truncate(s string, maxLen int, suffix string) string {
	runeCount := Length(s)
	if runeCount <= maxLen {
		return s
	}
	suffixLen := Length(suffix)
	if maxLen <= suffixLen {
		return SubString(s, 0, maxLen)
	}
	return SubString(s, 0, maxLen-suffixLen) + suffix
}

// Abbreviate 缩写字符串
func Abbreviate(s string, maxLen int) string {
	return Truncate(s, maxLen, "...")
}

// Capitalize 首字母大写
func Capitalize(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// Uncapitalize 首字母小写
func Uncapitalize(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// RemovePrefix 移除前缀
func RemovePrefix(s, prefix string) string {
	return strings.TrimPrefix(s, prefix)
}

// RemoveSuffix 移除后缀
func RemoveSuffix(s, suffix string) string {
	return strings.TrimSuffix(s, suffix)
}

// RemoveAll 移除所有指定子串
func RemoveAll(s, substr string) string {
	return strings.ReplaceAll(s, substr, "")
}

// Join 使用分隔符连接字符串切片
func Join(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

// EqualsIgnoreCase 忽略大小写比较
func EqualsIgnoreCase(s1, s2 string) bool {
	return strings.EqualFold(s1, s2)
}

// TrimAll 移除所有空白字符
func TrimAll(s string) string {
	var result []rune
	for _, r := range s {
		if !unicode.IsSpace(r) {
			result = append(result, r)
		}
	}
	return string(result)
}

// NormalizeSpace 规范化空白字符（多个连续空白字符替换为单个空格）
func NormalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Wrap 换行字符串
func Wrap(s string, wrapLen int) []string {
	if wrapLen <= 0 {
		return []string{s}
	}

	var result []string
	runes := []rune(s)
	runeLen := len(runes)

	for i := 0; i < runeLen; i += wrapLen {
		end := i + wrapLen
		if end > runeLen {
			end = runeLen
		}
		result = append(result, string(runes[i:end]))
	}

	return result
}

// ==================== 中文拼音处理 ====================

// ToPinyin 转换为拼音
func ToPinyin(s string) string {
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	args.Heteronym = false
	pinyinSlice := pinyin.Pinyin(s, args)
	var result []string
	for _, p := range pinyinSlice {
		if len(p) > 0 {
			result = append(result, p[0])
		}
	}
	return strings.Join(result, "")
}

// ToPinyinWithFirstLetter 转换为拼音首字母
func ToPinyinWithFirstLetter(s string) string {
	args := pinyin.NewArgs()
	args.Style = pinyin.FirstLetter
	args.Heteronym = false
	pinyinSlice := pinyin.Pinyin(s, args)
	var result []string
	for _, p := range pinyinSlice {
		if len(p) > 0 {
			result = append(result, p[0])
		}
	}
	return strings.Join(result, "")
}

// ToPinyinWithSep 转换为带分隔符的拼音
func ToPinyinWithSep(s, sep string) string {
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	args.Heteronym = false
	pinyinSlice := pinyin.Pinyin(s, args)
	var result []string
	for _, p := range pinyinSlice {
		if len(p) > 0 {
			result = append(result, p[0])
		}
	}
	return strings.Join(result, sep)
}

// ==================== 加密哈希 ====================

// MD5 计算MD5哈希
//
//nolint:gosec // 工具库提供 MD5 哈希（非密码场景，如一致性/摘要），使用者需知悉其弱性
func MD5(s string) string {
	hash := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", hash)
}

// SHA1 计算SHA1哈希
//
//nolint:gosec // 工具库提供 SHA1 哈希（非密码场景，如一致性/摘要），使用者需知悉其弱性
func SHA1(s string) string {
	hash := sha1.Sum([]byte(s))
	return fmt.Sprintf("%x", hash)
}

// ==================== Base64编解码 ====================

// Base64Encode Base64编码
func Base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// Base64Decode Base64解码
func Base64Decode(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ==================== 随机字符串生成 ====================

const (
	letterBytes         = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberBytes         = "0123456789"
	allBytes            = letterBytes + numberBytes
	specialBytes        = "!@#$%^&*()_+"
	allWithSpecialBytes = allBytes + specialBytes
)

// RandomString 生成随机字符串（注意：使用 math/rand，非密码学安全，如需安全随机请使用 generate 包）
func RandomString(length int) string {
	return randomString(length, allBytes)
}

// RandomStringWithNumbers 生成包含数字的随机字符串
func RandomStringWithNumbers(length int) string {
	return randomString(length, allBytes)
}

// RandomStringWithSpecial 生成包含特殊字符的随机字符串
func RandomStringWithSpecial(length int) string {
	return randomString(length, allWithSpecialBytes)
}

// RandomNumberString 生成随机数字字符串
func RandomNumberString(length int) string {
	return randomString(length, numberBytes)
}

//nolint:gosec // 伪随机字符串为工具定位（临时 token/测试数据），非安全场景
func randomString(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}

// ==================== 字符串转换 ====================

// ToInt 转换为整数
func ToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// ToIntDefault 转换为整数，失败返回默认值
func ToIntDefault(s string, defaultValue int) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return val
}

// ToFloat 转换为浮点数
func ToFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// ToFloatDefault 转换为浮点数，失败返回默认值
func ToFloatDefault(s string, defaultValue float64) float64 {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultValue
	}
	return val
}

// ToBool 转换为布尔值
func ToBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}

// ToBoolDefault 转换为布尔值，失败返回默认值
func ToBoolDefault(s string, defaultValue bool) bool {
	val, err := strconv.ParseBool(s)
	if err != nil {
		return defaultValue
	}
	return val
}

// ==================== 字符串验证 ====================

// IsPhone 判断是否为手机号
func IsPhone(s string) bool {
	pattern := `^1[3-9]\d{9}$`
	return Matches(pattern, s)
}

// IsIDCard 判断是否为身份证号
func IsIDCard(s string) bool {
	pattern := `^[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[0-9Xx]$`
	return Matches(pattern, s)
}

// IsChineseIDCard 判断是否为中国身份证号
func IsChineseIDCard(s string) bool {
	return IsIDCard(s)
}

// IsIP 判断是否为IP地址
func IsIP(s string) bool {
	pattern := `^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
	return Matches(pattern, s)
}

// IsIPv6 判断是否为IPv6地址
func IsIPv6(s string) bool {
	pattern := `^(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`
	return Matches(pattern, s)
}

// ==================== 其他实用功能 ====================

// IsPalindrome 判断是否为回文
func IsPalindrome(s string) bool {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		if runes[i] != runes[j] {
			return false
		}
	}
	return true
}

// Rotate 旋转字符串
func Rotate(s string, n int) string {
	runes := []rune(s)
	length := len(runes)
	if length == 0 {
		return s
	}
	n %= length
	if n < 0 {
		n += length
	}
	return string(append(runes[n:], runes[:n]...))
}

// Shuffle 打乱字符串
func Shuffle(s string) string {
	runes := []rune(s)
	rand.Shuffle(len(runes), func(i, j int) {
		runes[i], runes[j] = runes[j], runes[i]
	})
	return string(runes)
}

// Intersect 求两个字符串的交集
func Intersect(s1, s2 string) string {
	charMap := make(map[rune]bool)
	for _, r := range s2 {
		charMap[r] = true
	}
	var result []rune
	for _, r := range s1 {
		if charMap[r] {
			result = append(result, r)
			delete(charMap, r) // 避免重复
		}
	}
	return string(result)
}

// Union 求两个字符串的并集
func Union(s1, s2 string) string {
	charMap := make(map[rune]bool)
	for _, r := range s1 {
		charMap[r] = true
	}
	for _, r := range s2 {
		charMap[r] = true
	}
	var result []rune
	for r := range charMap {
		result = append(result, r)
	}
	return string(result)
}

// Difference 求两个字符串的差集
func Difference(s1, s2 string) string {
	charMap := make(map[rune]bool)
	for _, r := range s2 {
		charMap[r] = true
	}
	var result []rune
	for _, r := range s1 {
		if !charMap[r] {
			result = append(result, r)
		}
	}
	return string(result)
}

// FirstNonBlank 获取第一个非空白字符
func FirstNonBlank(s string) string {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return string(r)
		}
	}
	return ""
}

// LastNonBlank 获取最后一个非空白字符
func LastNonBlank(s string) string {
	runes := []rune(s)
	for i := len(runes) - 1; i >= 0; i-- {
		if !unicode.IsSpace(runes[i]) {
			return string(runes[i])
		}
	}
	return ""
}

// SwapCase 交换大小写
func SwapCase(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			runes[i] = unicode.ToLower(r)
		} else if unicode.IsLower(r) {
			runes[i] = unicode.ToUpper(r)
		}
	}
	return string(runes)
}

// RepeatWithSep 重复字符串并添加分隔符
func RepeatWithSep(s string, count int, sep string) string {
	if count <= 0 {
		return ""
	}
	var result []string
	for i := 0; i < count; i++ {
		result = append(result, s)
	}
	return strings.Join(result, sep)
}

// Indent 缩进字符串
func Indent(s, indent string) string {
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, indent+line)
	}
	return strings.Join(result, "\n")
}

// Unindent 取消缩进
func Unindent(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, strings.TrimLeft(line, " \t"))
	}
	return strings.Join(result, "\n")
}
