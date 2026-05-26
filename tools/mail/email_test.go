package mail

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// TestNewEmail 测试创建新邮件
func TestNewEmail(t *testing.T) {
	email := NewEmail()
	if email == nil {
		t.Error("NewEmail() should return a non-nil Email")
	}
	if len(email.To) != 0 {
		t.Errorf("NewEmail() should have empty To slice, got %d", len(email.To))
	}
	if len(email.Cc) != 0 {
		t.Errorf("NewEmail() should have empty Cc slice, got %d", len(email.Cc))
	}
	if len(email.Bcc) != 0 {
		t.Errorf("NewEmail() should have empty Bcc slice, got %d", len(email.Bcc))
	}
	if len(email.Attachments) != 0 {
		t.Errorf("NewEmail() should have empty Attachments slice, got %d", len(email.Attachments))
	}
	if email.Headers == nil {
		t.Error("NewEmail() should have non-nil Headers map")
	}
}

// TestEmailBuilder 测试邮件构建器模式
func TestEmailBuilder(t *testing.T) {
	email := NewEmail().AddTo("recipient1@example.com", "recipient2@example.com").AddCc("cc1@example.com").AddBcc("bcc1@example.com").SetSubject("Test Subject").SetBody("Test Body").SetHTMLBody("<p>Test HTML Body</p>").AddHeader("X-Custom-Header", "custom-value")

	if len(email.To) != 2 {
		t.Errorf("Email should have 2 recipients, got %d", len(email.To))
	}
	if len(email.Cc) != 1 {
		t.Errorf("Email should have 1 CC recipient, got %d", len(email.Cc))
	}
	if len(email.Bcc) != 1 {
		t.Errorf("Email should have 1 BCC recipient, got %d", len(email.Bcc))
	}
	if email.Subject != "Test Subject" {
		t.Errorf("Email subject should be 'Test Subject', got %q", email.Subject)
	}
	if email.Body != "Test Body" {
		t.Errorf("Email body should be 'Test Body', got %q", email.Body)
	}
	if email.HTMLBody != "<p>Test HTML Body</p>" {
		t.Errorf("Email HTML body should be '<p>Test HTML Body</p>', got %q", email.HTMLBody)
	}
	if email.Headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("Email custom header should be 'custom-value', got %q", email.Headers["X-Custom-Header"])
	}
}

// TestAddAttachment 测试添加附件
func TestAddAttachment(t *testing.T) {
	email := NewEmail()
	attachmentContent := []byte("attachment content")
	email.AddAttachment("test.txt", "text/plain", attachmentContent)

	if len(email.Attachments) != 1 {
		t.Errorf("Email should have 1 attachment, got %d", len(email.Attachments))
	}

	attachment := email.Attachments[0]
	if attachment.Filename != "test.txt" {
		t.Errorf("Attachment filename should be 'test.txt', got %q", attachment.Filename)
	}
	if attachment.ContentType != "text/plain" {
		t.Errorf("Attachment content type should be 'text/plain', got %q", attachment.ContentType)
	}
	if !bytes.Equal(attachment.Content, attachmentContent) {
		t.Error("Attachment content should match")
	}
}

// TestToMIME 测试转换为MIME格式
func TestToMIME(t *testing.T) {
	email := NewEmail()
	email.From = "sender@example.com"
	email.AddTo("recipient@example.com")
	email.SetSubject("Test Subject")
	email.SetBody("Test Body")

	mimeData, err := email.ToMIME()
	if err != nil {
		t.Errorf("ToMIME() should not return error, got %v", err)
	}
	if len(mimeData) == 0 {
		t.Error("ToMIME() should return non-empty MIME data")
	}

	// 检查MIME数据是否包含必要的部分
	if !bytes.Contains(mimeData, []byte("From: sender@example.com")) {
		t.Error("MIME data should contain From header")
	}
	if !bytes.Contains(mimeData, []byte("To: recipient@example.com")) {
		t.Error("MIME data should contain To header")
	}
	if !bytes.Contains(mimeData, []byte("Subject: Test Subject")) {
		t.Error("MIME data should contain Subject header")
	}
	if !bytes.Contains(mimeData, []byte("Test Body")) {
		t.Error("MIME data should contain body")
	}
}

// TestToMIMEWithHTML 测试转换为带HTML的MIME格式
func TestToMIMEWithHTML(t *testing.T) {
	email := NewEmail()
	email.From = "sender@example.com"
	email.AddTo("recipient@example.com")
	email.SetSubject("Test Subject")
	email.SetBody("Test Body")
	email.SetHTMLBody("<p>Test HTML Body</p>")

	mimeData, err := email.ToMIME()
	if err != nil {
		t.Errorf("ToMIME() should not return error, got %v", err)
	}
	if len(mimeData) == 0 {
		t.Error("ToMIME() should return non-empty MIME data")
	}

	// 检查MIME数据是否包含HTML部分
	if !bytes.Contains(mimeData, []byte("text/html")) {
		t.Error("MIME data should contain HTML content type")
	}
	if !bytes.Contains(mimeData, []byte("<p>Test HTML Body</p>")) {
		t.Error("MIME data should contain HTML body")
	}
}

// TestToMIMEWithAttachment 测试转换为带附件的MIME格式
func TestToMIMEWithAttachment(t *testing.T) {
	email := NewEmail()
	email.From = "sender@example.com"
	email.AddTo("recipient@example.com")
	email.SetSubject("Test Subject")
	email.SetBody("Test Body")
	email.AddAttachment("test.txt", "text/plain", []byte("attachment content"))

	mimeData, err := email.ToMIME()
	if err != nil {
		t.Errorf("ToMIME() should not return error, got %v", err)
	}
	if len(mimeData) == 0 {
		t.Error("ToMIME() should return non-empty MIME data")
	}

	// 检查MIME数据是否包含附件部分
	if !bytes.Contains(mimeData, []byte("attachment")) {
		t.Error("MIME data should contain attachment")
	}
	if !bytes.Contains(mimeData, []byte("test.txt")) {
		t.Error("MIME data should contain attachment filename")
	}
}

// TestParseMIME 测试从MIME格式解析邮件
func TestParseMIME(t *testing.T) {
	// 创建一个简单的MIME邮件
	mimeData := []byte(`From: sender@example.com
To: recipient@example.com
Cc: cc@example.com
Subject: Test Subject
Date: Fri, 01 Jan 2026 12:00:00 +0000
Content-Type: text/plain; charset=utf-8

Test Body`)

	email, err := ParseMIME(mimeData)
	if err != nil {
		t.Errorf("ParseMIME() should not return error, got %v", err)
	}
	if email == nil {
		t.Error("ParseMIME() should return a non-nil Email")
	}

	// 检查邮件地址是否包含在解析结果中（允许包含尖括号）
	if !strings.Contains(email.From, "sender@example.com") {
		t.Errorf("Parsed email From should contain 'sender@example.com', got %q", email.From)
	}
	if len(email.To) != 1 || !strings.Contains(email.To[0], "recipient@example.com") {
		t.Errorf("Parsed email To should contain 'recipient@example.com', got %v", email.To)
	}
	if len(email.Cc) != 1 || !strings.Contains(email.Cc[0], "cc@example.com") {
		t.Errorf("Parsed email Cc should contain 'cc@example.com', got %v", email.Cc)
	}
	if email.Subject != "Test Subject" {
		t.Errorf("Parsed email Subject should be 'Test Subject', got %q", email.Subject)
	}
	if email.Body != "Test Body" {
		t.Errorf("Parsed email Body should be 'Test Body', got %q", email.Body)
	}
}

// TestSetSentAt 测试设置发送时间
func TestSetSentAt(t *testing.T) {
	email := NewEmail()
	testTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	email.SetSentAt(testTime)

	if !email.SentAt.Equal(testTime) {
		t.Errorf("Email SentAt should be %v, got %v", testTime, email.SentAt)
	}
}
