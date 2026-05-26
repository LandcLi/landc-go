package email

import (
	"strings"
	"testing"
)

// TestIsValid 测试邮箱验证
func TestIsValid(t *testing.T) {
	testCases := []struct {
		name  string
		email string
		want  bool
	}{
		{"Valid email", "test@example.com", true},
		{"Valid email with dot", "test.name@example.com", true},
		{"Valid email with number", "test123@example.com", true},
		{"Valid email with hyphen", "test-name@example.com", true},
		{"Valid email with underscore", "test_name@example.com", true},
		{"Valid email with multiple dots", "test.name@example.co.uk", true},
		{"Invalid email (no @)", "testexample.com", false},
		{"Invalid email (no domain)", "test@", false},
		{"Invalid email (no username)", "@example.com", false},
		{"Invalid email (short domain)", "test@ex.co", true},
		{"Invalid email (very short domain)", "test@ex.c", false},
		{"Empty email", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsValid(tc.email)
			if got != tc.want {
				t.Errorf("IsValid(%q) = %v, want %v", tc.email, got, tc.want)
			}
		})
	}
}

// TestIsValidWithDomain 测试带域名的邮箱验证
func TestIsValidWithDomain(t *testing.T) {
	testCases := []struct {
		name   string
		email  string
		domain string
		want   bool
	}{
		{"Valid email with correct domain", "test@gmail.com", "gmail.com", true},
		{"Valid email with incorrect domain", "test@gmail.com", "yahoo.com", false},
		{"Invalid email with correct domain", "testgmail.com", "gmail.com", false},
		{"Empty domain", "test@example.com", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsValidWithDomain(tc.email, tc.domain)
			if got != tc.want {
				t.Errorf("IsValidWithDomain(%q, %q) = %v, want %v", tc.email, tc.domain, got, tc.want)
			}
		})
	}
}

// TestIsValidWithDomains 测试带域名列表的邮箱验证
func TestIsValidWithDomains(t *testing.T) {
	testCases := []struct {
		name    string
		email   string
		domains []string
		want    bool
	}{
		{"Valid email with domain in list", "test@gmail.com", []string{"gmail.com", "yahoo.com"}, true},
		{"Valid email with domain not in list", "test@gmail.com", []string{"hotmail.com", "outlook.com"}, false},
		{"Invalid email with domain in list", "testgmail.com", []string{"gmail.com", "yahoo.com"}, false},
		{"Empty domain list", "test@example.com", []string{}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsValidWithDomains(tc.email, tc.domains)
			if got != tc.want {
				t.Errorf("IsValidWithDomains(%q, %v) = %v, want %v", tc.email, tc.domains, got, tc.want)
			}
		})
	}
}

// TestParse 测试邮箱解析
func TestParse(t *testing.T) {
	testCases := []struct {
		name       string
		email      string
		wantUser   string
		wantDomain string
	}{
		{"Valid email", "test@example.com", "test", "example.com"},
		{"Valid email with dot", "test.name@example.com", "test.name", "example.com"},
		{"Invalid email (no @)", "testexample.com", "", ""},
		{"Invalid email (no domain)", "test@", "", ""},
		{"Invalid email (no username)", "@example.com", "", ""},
		{"Empty email", "", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotUser, gotDomain := Parse(tc.email)
			if gotUser != tc.wantUser || gotDomain != tc.wantDomain {
				t.Errorf("Parse(%q) = (%q, %q), want (%q, %q)", tc.email, gotUser, gotDomain, tc.wantUser, tc.wantDomain)
			}
		})
	}
}

// TestGetUsername 测试获取邮箱用户名
func TestGetUsername(t *testing.T) {
	testCases := []struct {
		name  string
		email string
		want  string
	}{
		{"Valid email", "test@example.com", "test"},
		{"Valid email with dot", "test.name@example.com", "test.name"},
		{"Invalid email", "testexample.com", ""},
		{"Empty email", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetUsername(tc.email)
			if got != tc.want {
				t.Errorf("GetUsername(%q) = %q, want %q", tc.email, got, tc.want)
			}
		})
	}
}

// TestGetDomain 测试获取邮箱域名
func TestGetDomain(t *testing.T) {
	testCases := []struct {
		name  string
		email string
		want  string
	}{
		{"Valid email", "test@example.com", "example.com"},
		{"Valid email with dot", "test.name@example.com", "example.com"},
		{"Invalid email", "testexample.com", ""},
		{"Empty email", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetDomain(tc.email)
			if got != tc.want {
				t.Errorf("GetDomain(%q) = %q, want %q", tc.email, got, tc.want)
			}
		})
	}
}

// TestGenerate 测试生成随机邮箱
func TestGenerate(t *testing.T) {
	domain := "example.com"
	email := Generate(domain)
	if !strings.Contains(email, "@"+domain) {
		t.Errorf("Generate(%q) should contain @%s", domain, domain)
	}
	if !IsValid(email) {
		t.Errorf("Generate(%q) should return a valid email, got %q", domain, email)
	}
}

// TestGenerateWithCustomUsername 测试生成指定用户名的邮箱
func TestGenerateWithCustomUsername(t *testing.T) {
	username := "testuser"
	domain := "example.com"
	expected := username + "@" + domain
	got := GenerateWithCustomUsername(username, domain)
	if got != expected {
		t.Errorf("GenerateWithCustomUsername(%q, %q) = %q, want %q", username, domain, got, expected)
	}
}

// TestGenerateWithRandomDomain 测试生成带随机域名的邮箱
func TestGenerateWithRandomDomain(t *testing.T) {
	email := GenerateWithRandomDomain()
	if !IsValid(email) {
		t.Errorf("GenerateWithRandomDomain() should return a valid email, got %q", email)
	}
}

// TestGenerateBatch 测试批量生成随机邮箱
func TestGenerateBatch(t *testing.T) {
	count := 5
	domain := "example.com"
	emails := GenerateBatch(count, domain)
	if len(emails) != count {
		t.Errorf("GenerateBatch(%d, %q) should return %d emails, got %d", count, domain, count, len(emails))
	}
	for _, email := range emails {
		if !strings.Contains(email, "@"+domain) {
			t.Errorf("GenerateBatch email should contain @%s, got %q", domain, email)
		}
		if !IsValid(email) {
			t.Errorf("GenerateBatch should return valid emails, got %q", email)
		}
	}
}

// TestMask 测试掩码邮箱
func TestMask(t *testing.T) {
	testCases := []struct {
		name  string
		email string
		want  string
	}{
		{"Email with long username", "testuser@example.com", "te****er@example.com"},
		{"Email with medium username", "test@example.com", "te**@example.com"},
		{"Email with short username", "te@example.com", "te**@example.com"},
		{"Invalid email", "testexample.com", "testexample.com"},
		{"Empty email", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Mask(tc.email)
			if got != tc.want {
				t.Errorf("Mask(%q) = %q, want %q", tc.email, got, tc.want)
			}
		})
	}
}

// TestMaskWithCustom 测试自定义掩码邮箱
func TestMaskWithCustom(t *testing.T) {
	testCases := []struct {
		name       string
		email      string
		keepPrefix int
		keepSuffix int
		want       string
	}{
		{"Custom mask with 3 prefix and 2 suffix", "testuser@example.com", 3, 2, "tes***er@example.com"},
		{"Custom mask with 2 prefix and 1 suffix", "test@example.com", 2, 1, "te*t@example.com"},
		{"Custom mask with insufficient length", "test@example.com", 2, 3, "test@example.com"},
		{"Invalid email", "testexample.com", 2, 2, "testexample.com"},
		{"Empty email", "", 2, 2, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MaskWithCustom(tc.email, tc.keepPrefix, tc.keepSuffix)
			if got != tc.want {
				t.Errorf("MaskWithCustom(%q, %d, %d) = %q, want %q", tc.email, tc.keepPrefix, tc.keepSuffix, got, tc.want)
			}
		})
	}
}

// TestNormalize 测试标准化邮箱
func TestNormalize(t *testing.T) {
	testCases := []struct {
		name  string
		email string
		want  string
	}{
		{"Normalize email with spaces", "  Test@Example.Com  ", "test@example.com"},
		{"Normalize email with uppercase", "TEST@EXAMPLE.COM", "test@example.com"},
		{"Normalize email with mixed case", "Test.Name@Example.Com", "test.name@example.com"},
		{"Normalize already normalized email", "test@example.com", "test@example.com"},
		{"Normalize empty email", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Normalize(tc.email)
			if got != tc.want {
				t.Errorf("Normalize(%q) = %q, want %q", tc.email, got, tc.want)
			}
		})
	}
}

// TestExtractFromText 测试从文本中提取邮箱
func TestExtractFromText(t *testing.T) {
	testCases := []struct {
		name string
		text string
		want []string
	}{
		{"Extract single email", "Contact me at test@example.com", []string{"test@example.com"}},
		{"Extract multiple emails", "Emails: test1@example.com, test2@example.com", []string{"test1@example.com", "test2@example.com"}},
		{"Extract no emails", "No emails here", []string{}},
		{"Extract empty text", "", []string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractFromText(tc.text)
			if len(got) != len(tc.want) {
				t.Errorf("ExtractFromText(%q) should return %d emails, got %d", tc.text, len(tc.want), len(got))
				return
			}
			for i, email := range got {
				if email != tc.want[i] {
					t.Errorf("ExtractFromText(%q)[%d] = %q, want %q", tc.text, i, email, tc.want[i])
				}
			}
		})
	}
}

// TestIsGmail 测试是否为Gmail邮箱
func TestIsGmail(t *testing.T) {
	testCases := []struct {
		name  string
		email string
		want  bool
	}{
		{"Gmail email", "test@gmail.com", true},
		{"Non-Gmail email", "test@example.com", false},
		{"Invalid email", "testgmail.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsGmail(tc.email)
			if got != tc.want {
				t.Errorf("IsGmail(%q) = %v, want %v", tc.email, got, tc.want)
			}
		})
	}
}

// TestIsQQ 测试是否为QQ邮箱
func TestIsQQ(t *testing.T) {
	testCases := []struct {
		name  string
		email string
		want  bool
	}{
		{"QQ email", "test@qq.com", true},
		{"Non-QQ email", "test@example.com", false},
		{"Invalid email", "testqq.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsQQ(tc.email)
			if got != tc.want {
				t.Errorf("IsQQ(%q) = %v, want %v", tc.email, got, tc.want)
			}
		})
	}
}

// TestReplaceDomain 测试替换邮箱域名
func TestReplaceDomain(t *testing.T) {
	testCases := []struct {
		name      string
		email     string
		newDomain string
		want      string
	}{
		{"Replace domain", "test@example.com", "newdomain.com", "test@newdomain.com"},
		{"Invalid email", "testexample.com", "newdomain.com", "testexample.com"},
		{"Empty email", "", "newdomain.com", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ReplaceDomain(tc.email, tc.newDomain)
			if got != tc.want {
				t.Errorf("ReplaceDomain(%q, %q) = %q, want %q", tc.email, tc.newDomain, got, tc.want)
			}
		})
	}
}

// TestAppendToUsername 测试向邮箱用户名追加内容
func TestAppendToUsername(t *testing.T) {
	testCases := []struct {
		name   string
		email  string
		suffix string
		want   string
	}{
		{"Append suffix", "test@example.com", "123", "test123@example.com"},
		{"Invalid email", "testexample.com", "123", "testexample.com"},
		{"Empty email", "", "123", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := AppendToUsername(tc.email, tc.suffix)
			if got != tc.want {
				t.Errorf("AppendToUsername(%q, %q) = %q, want %q", tc.email, tc.suffix, got, tc.want)
			}
		})
	}
}

// TestPrependToUsername 测试向邮箱用户名前缀添加内容
func TestPrependToUsername(t *testing.T) {
	testCases := []struct {
		name   string
		email  string
		prefix string
		want   string
	}{
		{"Prepend prefix", "test@example.com", "user_", "user_test@example.com"},
		{"Invalid email", "testexample.com", "user_", "testexample.com"},
		{"Empty email", "", "user_", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := PrependToUsername(tc.email, tc.prefix)
			if got != tc.want {
				t.Errorf("PrependToUsername(%q, %q) = %q, want %q", tc.email, tc.prefix, got, tc.want)
			}
		})
	}
}

// TestIsDisposable 测试是否为临时邮箱
func TestIsDisposable(t *testing.T) {
	testCases := []struct {
		name  string
		email string
		want  bool
	}{
		{"Disposable email", "test@yopmail.com", true},
		{"Non-disposable email", "test@example.com", false},
		{"Invalid email", "testexample.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsDisposable(tc.email)
			if got != tc.want {
				t.Errorf("IsDisposable(%q) = %v, want %v", tc.email, got, tc.want)
			}
		})
	}
}

// TestIsCorporate 测试是否为企业邮箱
func TestIsCorporate(t *testing.T) {
	testCases := []struct {
		name  string
		email string
		want  bool
	}{
		{"Corporate email", "test@company.com", true},
		{"Personal email", "test@gmail.com", false},
		{"Invalid email", "testexample.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsCorporate(tc.email)
			if got != tc.want {
				t.Errorf("IsCorporate(%q) = %v, want %v", tc.email, got, tc.want)
			}
		})
	}
}
