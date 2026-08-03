package email

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
)

// ==================== 邮箱验证 ====================

// IsValid 判断邮箱是否有效
func IsValid(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(email)
}

// IsValidWithDomain 判断邮箱是否有效且属于指定域名
func IsValidWithDomain(email, domain string) bool {
	if !IsValid(email) {
		return false
	}
	_, emailDomain := Parse(email)
	return strings.EqualFold(emailDomain, domain)
}

// IsValidWithDomains 判断邮箱是否有效且属于指定域名列表
func IsValidWithDomains(email string, domains []string) bool {
	if !IsValid(email) {
		return false
	}
	_, emailDomain := Parse(email)
	for _, domain := range domains {
		if strings.EqualFold(emailDomain, domain) {
			return true
		}
	}
	return false
}

// ==================== 邮箱解析 ====================

// Parse 解析邮箱地址，返回用户名和域名
func Parse(email string) (username, domain string) {
	parts := strings.Split(email, "@")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1]
	}
	return "", ""
}

// GetUsername 获取邮箱用户名
func GetUsername(email string) string {
	username, _ := Parse(email)
	return username
}

// GetDomain 获取邮箱域名
func GetDomain(email string) string {
	_, domain := Parse(email)
	return domain
}

// ==================== 邮箱生成 ====================

// Generate 生成随机邮箱地址
func Generate(domain string) string {
	username := generateUsername()
	return fmt.Sprintf("%s@%s", username, domain)
}

// GenerateWithCustomUsername 生成指定用户名的邮箱地址
func GenerateWithCustomUsername(username, domain string) string {
	return fmt.Sprintf("%s@%s", username, domain)
}

// GenerateWithRandomDomain 生成带随机域名的邮箱地址
func GenerateWithRandomDomain() string {
	username := generateUsername()
	domain := getRandomDomain()
	return fmt.Sprintf("%s@%s", username, domain)
}

// GenerateBatch 批量生成随机邮箱地址
func GenerateBatch(count int, domain string) []string {
	emails := make([]string, count)
	for i := 0; i < count; i++ {
		emails[i] = Generate(domain)
	}
	return emails
}

// ==================== 邮箱操作 ====================

// Mask 掩码邮箱地址
func Mask(email string) string {
	username, domain := Parse(email)
	if username == "" || domain == "" {
		return email
	}

	usernameLen := len(username)
	var maskedUsername string
	if usernameLen <= 2 {
		maskedUsername = username[0:] + strings.Repeat("*", usernameLen)
	} else if usernameLen <= 4 {
		maskedUsername = username[:2] + strings.Repeat("*", usernameLen-2)
	} else {
		maskedUsername = username[:2] + strings.Repeat("*", usernameLen-4) + username[usernameLen-2:]
	}

	return fmt.Sprintf("%s@%s", maskedUsername, domain)
}

// MaskWithCustom 自定义掩码邮箱地址
func MaskWithCustom(email string, keepPrefix, keepSuffix int) string {
	username, domain := Parse(email)
	if username == "" || domain == "" {
		return email
	}

	usernameLen := len(username)
	if keepPrefix+keepSuffix >= usernameLen {
		return email
	}

	maskedUsername := username[:keepPrefix] + strings.Repeat("*", usernameLen-keepPrefix-keepSuffix) + username[usernameLen-keepSuffix:]
	return fmt.Sprintf("%s@%s", maskedUsername, domain)
}

// Normalize 标准化邮箱地址
func Normalize(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	return email
}

// ExtractFromText 从文本中提取邮箱地址
func ExtractFromText(text string) []string {
	pattern := `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`
	re := regexp.MustCompile(pattern)
	return re.FindAllString(text, -1)
}

// ==================== 辅助函数 ====================

// generateUsername 生成随机用户名
//
//nolint:gosec // 生成测试邮箱的伪随机，非安全场景
func generateUsername() string {
	length := rand.Intn(8) + 5 // 5-12位
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	username := make([]byte, length)
	for i := range username {
		username[i] = chars[rand.Intn(len(chars))]
	}
	return string(username)
}

// getRandomDomain 获取随机域名
//
//nolint:gosec // 生成测试邮箱的伪随机，非安全场景
func getRandomDomain() string {
	domains := []string{
		"gmail.com",
		"yahoo.com",
		"hotmail.com",
		"outlook.com",
		"icloud.com",
		"qq.com",
		"163.com",
		"126.com",
		"sina.com",
		"aliyun.com",
	}
	return domains[rand.Intn(len(domains))]
}

// ==================== 邮箱格式检查 ====================

// IsGmail 判断是否为Gmail邮箱
func IsGmail(email string) bool {
	return IsValidWithDomain(email, "gmail.com")
}

// IsQQ 判断是否为QQ邮箱
func IsQQ(email string) bool {
	return IsValidWithDomain(email, "qq.com")
}

// Is163 判断是否为163邮箱
func Is163(email string) bool {
	return IsValidWithDomain(email, "163.com")
}

// IsOutlook 判断是否为Outlook邮箱
func IsOutlook(email string) bool {
	return IsValidWithDomain(email, "outlook.com") || IsValidWithDomain(email, "hotmail.com")
}

// ==================== 邮箱处理 ====================

// ReplaceDomain 替换邮箱域名
func ReplaceDomain(email, newDomain string) string {
	username, _ := Parse(email)
	if username == "" {
		return email
	}
	return fmt.Sprintf("%s@%s", username, newDomain)
}

// AppendToUsername 向邮箱用户名追加内容
func AppendToUsername(email, suffix string) string {
	username, domain := Parse(email)
	if username == "" || domain == "" {
		return email
	}
	return fmt.Sprintf("%s%s@%s", username, suffix, domain)
}

// PrependToUsername 向邮箱用户名前缀添加内容
func PrependToUsername(email, prefix string) string {
	username, domain := Parse(email)
	if username == "" || domain == "" {
		return email
	}
	return fmt.Sprintf("%s%s@%s", prefix, username, domain)
}

// ==================== 邮箱验证增强 ====================

// IsDisposable 判断是否为临时邮箱
func IsDisposable(email string) bool {
	disposableDomains := []string{
		"yopmail.com",
		"tempmail.com",
		"fakeinbox.com",
		"dispostable.com",
		"trashmail.com",
		" Guerrillamail.com",
		"mailinator.com",
		"throwawaymail.com",
		"tempmailaddress.com",
		"tempmailer.com",
	}
	domain := GetDomain(email)
	for _, disposableDomain := range disposableDomains {
		if strings.EqualFold(domain, disposableDomain) {
			return true
		}
	}
	return false
}

// IsCorporate 判断是否为企业邮箱（基于域名判断）
func IsCorporate(email string) bool {
	// 常见个人邮箱域名
	personalDomains := []string{
		"gmail.com",
		"yahoo.com",
		"hotmail.com",
		"outlook.com",
		"icloud.com",
		"qq.com",
		"163.com",
		"126.com",
		"sina.com",
		"aliyun.com",
	}
	domain := GetDomain(email)
	for _, personalDomain := range personalDomains {
		if strings.EqualFold(domain, personalDomain) {
			return false
		}
	}
	return IsValid(email)
}
