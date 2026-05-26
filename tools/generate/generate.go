package generate

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UUID 生成 UUID v4
func UUID() string {
	return uuid.New().String()
}

// UUIDv1 生成 UUID v1
func UUIDv1() string {
	id, _ := uuid.NewUUID()
	return id.String()
}

// UUIDv3 生成 UUID v3
func UUIDv3(namespace uuid.UUID, name string) string {
	id := uuid.NewMD5(namespace, []byte(name))
	return id.String()
}

// UUIDv5 生成 UUID v5
func UUIDv5(namespace uuid.UUID, name string) string {
	id := uuid.NewSHA1(namespace, []byte(name))
	return id.String()
}

// RandomInt 生成指定范围内的随机整数 [min, max]
func RandomInt(min, max int) (int, error) {
	if min > max {
		return 0, fmt.Errorf("min cannot be greater than max")
	}

	bigMax := big.NewInt(int64(max - min + 1))
	n, err := rand.Int(rand.Reader, bigMax)
	if err != nil {
		return 0, err
	}

	return min + int(n.Int64()), nil
}

// MustRandomInt 生成指定范围内的随机整数，如果出错则 panic
func MustRandomInt(min, max int) int {
	val, err := RandomInt(min, max)
	if err != nil {
		panic(err)
	}
	return val
}

// RandomString 生成指定长度的随机字符串
func RandomString(length int, charsets ...string) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be greater than 0")
	}

	// 默认字符集
	defaultCharset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charset := defaultCharset

	// 如果提供了字符集，则使用提供的
	if len(charsets) > 0 {
		charset = strings.Join(charsets, "")
	}

	if len(charset) == 0 {
		return "", fmt.Errorf("charset cannot be empty")
	}

	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}

	return string(result), nil
}

// MustRandomString 生成指定长度的随机字符串，如果出错则 panic
func MustRandomString(length int, charsets ...string) string {
	val, err := RandomString(length, charsets...)
	if err != nil {
		panic(err)
	}
	return val
}

// RandomStringWithCharset 生成指定长度和字符集的随机字符串
func RandomStringWithCharset(length int, charset string) (string, error) {
	return RandomString(length, charset)
}

// Captcha 生成指定长度的数字验证码
func Captcha(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be greater than 0")
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := RandomInt(0, 9)
		if err != nil {
			return "", err
		}
		result[i] = byte('0' + n)
	}

	return string(result), nil
}

// MustCaptcha 生成指定长度的数字验证码，如果出错则 panic
func MustCaptcha(length int) string {
	val, err := Captcha(length)
	if err != nil {
		panic(err)
	}
	return val
}

// AlphaCaptcha 生成指定长度的字母验证码
func AlphaCaptcha(length int) (string, error) {
	return RandomString(length, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
}

// MustAlphaCaptcha 生成指定长度的字母验证码，如果出错则 panic
func MustAlphaCaptcha(length int) string {
	val, err := AlphaCaptcha(length)
	if err != nil {
		panic(err)
	}
	return val
}

// AlphaNumericCaptcha 生成指定长度的字母数字混合验证码
func AlphaNumericCaptcha(length int) (string, error) {
	return RandomString(length, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
}

// MustAlphaNumericCaptcha 生成指定长度的字母数字混合验证码，如果出错则 panic
func MustAlphaNumericCaptcha(length int) string {
	val, err := AlphaNumericCaptcha(length)
	if err != nil {
		panic(err)
	}
	return val
}

// RandomIP 生成随机 IPv4 地址
func RandomIP() (string, error) {
	ip := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		n, err := RandomInt(0, 255)
		if err != nil {
			return "", err
		}
		ip[i] = byte(n)
	}
	return ip.String(), nil
}

// MustRandomIP 生成随机 IPv4 地址，如果出错则 panic
func MustRandomIP() string {
	val, err := RandomIP()
	if err != nil {
		panic(err)
	}
	return val
}

// RandomMAC 生成随机 MAC 地址
func RandomMAC() (string, error) {
	mac := make([]byte, 6)
	for i := 0; i < 6; i++ {
		n, err := RandomInt(0, 255)
		if err != nil {
			return "", err
		}
		mac[i] = byte(n)
	}
	// 设置本地管理位
	mac[0] &= 0xFE
	// 设置全局唯一位
	mac[0] |= 0x02

	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]), nil
}

// MustRandomMAC 生成随机 MAC 地址，如果出错则 panic
func MustRandomMAC() string {
	val, err := RandomMAC()
	if err != nil {
		panic(err)
	}
	return val
}

// RandomBool 生成随机布尔值
func RandomBool() (bool, error) {
	val, err := RandomInt(0, 1)
	if err != nil {
		return false, err
	}
	return val == 1, nil
}

// MustRandomBool 生成随机布尔值，如果出错则 panic
func MustRandomBool() bool {
	val, err := RandomBool()
	if err != nil {
		panic(err)
	}
	return val
}

// RandomFloat 生成指定范围内的随机浮点数 [min, max)
func RandomFloat(min, max float64) (float64, error) {
	if min > max {
		return 0, fmt.Errorf("min cannot be greater than max")
	}

	// 生成 [0, 1) 范围内的随机浮点数
	bigMax := big.NewInt(1000000000)
	n, err := rand.Int(rand.Reader, bigMax)
	if err != nil {
		return 0, err
	}

	fraction := float64(n.Int64()) / 1000000000.0
	return min + fraction*(max-min), nil
}

// MustRandomFloat 生成指定范围内的随机浮点数，如果出错则 panic
func MustRandomFloat(min, max float64) float64 {
	val, err := RandomFloat(min, max)
	if err != nil {
		panic(err)
	}
	return val
}

// RandomTimestamp 生成指定范围内的随机时间戳
func RandomTimestamp(start, end time.Time) (int64, error) {
	if start.After(end) {
		return 0, fmt.Errorf("start time cannot be after end time")
	}

	startUnix := start.Unix()
	endUnix := end.Unix()

	val, err := RandomInt(int(startUnix), int(endUnix))
	if err != nil {
		return 0, err
	}
	return int64(val), nil
}

// MustRandomTimestamp 生成指定范围内的随机时间戳，如果出错则 panic
func MustRandomTimestamp(start, end time.Time) int64 {
	val, err := RandomTimestamp(start, end)
	if err != nil {
		panic(err)
	}
	return val
}

// RandomTime 生成指定范围内的随机时间
func RandomTime(start, end time.Time) (time.Time, error) {
	timestamp, err := RandomTimestamp(start, end)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(timestamp, 0), nil
}

// MustRandomTime 生成指定范围内的随机时间，如果出错则 panic
func MustRandomTime(start, end time.Time) time.Time {
	val, err := RandomTime(start, end)
	if err != nil {
		panic(err)
	}
	return val
}

// GenerateStrongPassword 生成强密码
// 确保包含至少一个小写字母、一个大写字母、一个数字和一个特殊符号
func GenerateStrongPassword(length int) (string, error) {
	if length < 8 {
		return "", fmt.Errorf("password length should be at least 8")
	}

	// 定义字符集
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits := "0123456789"
	symbols := "!@#$%^&*()_+-=[]{}|;:,.<>?"

	// 先各选一个字符
	password := make([]byte, 0, length)

	// 从每种字符集中随机选择一个
	if c, err := RandomString(1, lowercase); err == nil {
		password = append(password, c...)
	}
	if c, err := RandomString(1, uppercase); err == nil {
		password = append(password, c...)
	}
	if c, err := RandomString(1, digits); err == nil {
		password = append(password, c...)
	}
	if c, err := RandomString(1, symbols); err == nil {
		password = append(password, c...)
	}

	// 生成剩余的字符
	remaining := length - len(password)
	if remaining > 0 {
		if c, err := RandomString(remaining, lowercase, uppercase, digits, symbols); err == nil {
			password = append(password, c...)
		}
	}

	// 打乱顺序
	for i := range password {
		j, err := RandomInt(0, len(password)-1)
		if err != nil {
			continue
		}
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}

// MustGenerateStrongPassword 生成强密码，如果出错则 panic
func MustGenerateStrongPassword(length int) string {
	val, err := GenerateStrongPassword(length)
	if err != nil {
		panic(err)
	}
	return val
}
