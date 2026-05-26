package mail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Email 邮件结构
type Email struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	Body        string
	HTMLBody    string
	Attachments []Attachment
	Headers     map[string]string
	SentAt      time.Time
}

// Attachment 附件结构
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// NewEmail 创建新邮件
func NewEmail() *Email {
	return &Email{
		To:          make([]string, 0),
		Cc:          make([]string, 0),
		Bcc:         make([]string, 0),
		Attachments: make([]Attachment, 0),
		Headers:     make(map[string]string),
		SentAt:      time.Now(),
	}
}

// AddTo 添加收件人
func (e *Email) AddTo(to ...string) *Email {
	e.To = append(e.To, to...)
	return e
}

// AddCc 添加抄送
func (e *Email) AddCc(cc ...string) *Email {
	e.Cc = append(e.Cc, cc...)
	return e
}

// AddBcc 添加密送
func (e *Email) AddBcc(bcc ...string) *Email {
	e.Bcc = append(e.Bcc, bcc...)
	return e
}

// SetSubject 设置主题
func (e *Email) SetSubject(subject string) *Email {
	e.Subject = subject
	return e
}

// SetBody 设置文本正文
func (e *Email) SetBody(body string) *Email {
	e.Body = body
	return e
}

// SetHTMLBody 设置HTML正文
func (e *Email) SetHTMLBody(htmlBody string) *Email {
	e.HTMLBody = htmlBody
	return e
}

// AddAttachment 添加附件
func (e *Email) AddAttachment(filename string, contentType string, content []byte) *Email {
	e.Attachments = append(e.Attachments, Attachment{
		Filename:    filename,
		ContentType: contentType,
		Content:     content,
	})
	return e
}

// AddAttachmentFromFile 从文件添加附件
func (e *Email) AddAttachmentFromFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	filename := filepath.Base(filePath)
	contentType := getContentType(filename)

	e.Attachments = append(e.Attachments, Attachment{
		Filename:    filename,
		ContentType: contentType,
		Content:     content,
	})

	return nil
}

// AddHeader 添加自定义头部
func (e *Email) AddHeader(key, value string) *Email {
	e.Headers[key] = value
	return e
}

// SetFrom 设置发件人
func (e *Email) SetFrom(from string) *Email {
	e.From = from
	return e
}

// SetSentAt 设置发送时间
func (e *Email) SetSentAt(sentAt time.Time) *Email {
	e.SentAt = sentAt
	return e
}

// ToMIME 将邮件转换为MIME格式
func (e *Email) ToMIME() ([]byte, error) {
	var buf bytes.Buffer
	var writer *multipart.Writer

	// 构建邮件头部
	buf.WriteString("From: " + e.From + "\r\n")
	if len(e.To) > 0 {
		buf.WriteString("To: " + strings.Join(e.To, ", ") + "\r\n")
	}
	if len(e.Cc) > 0 {
		buf.WriteString("Cc: " + strings.Join(e.Cc, ", ") + "\r\n")
	}
	buf.WriteString("Subject: " + e.Subject + "\r\n")
	buf.WriteString("Date: " + e.SentAt.Format(time.RFC1123Z) + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	// 添加自定义头部
	for key, value := range e.Headers {
		buf.WriteString(key + ": " + value + "\r\n")
	}

	// 确定邮件类型
	hasAttachments := len(e.Attachments) > 0
	hasHTML := e.HTMLBody != ""
	hasText := e.Body != ""

	if hasAttachments {
		// 带附件的邮件
		writer = multipart.NewWriter(&buf)
		buf.WriteString("Content-Type: multipart/mixed; boundary=" + writer.Boundary() + "\r\n\r\n")

		// 构建邮件正文部分
		if hasHTML || hasText {
			bodyPart, err := writer.CreatePart(textproto.MIMEHeader{
				"Content-Type": {"multipart/alternative; boundary=alt-boundary"},
			})
			if err != nil {
				return nil, err
			}

			// 文本正文
			if hasText {
				textPart, err := multipart.NewWriter(bodyPart).CreatePart(textproto.MIMEHeader{
					"Content-Type": {"text/plain; charset=utf-8"},
				})
				if err != nil {
					return nil, err
				}
				textPart.Write([]byte(e.Body))
			}

			// HTML正文
			if hasHTML {
				htmlPart, err := multipart.NewWriter(bodyPart).CreatePart(textproto.MIMEHeader{
					"Content-Type": {"text/html; charset=utf-8"},
				})
				if err != nil {
					return nil, err
				}
				htmlPart.Write([]byte(e.HTMLBody))
			}
		}

		// 添加附件
		for _, attachment := range e.Attachments {
			part, err := writer.CreatePart(textproto.MIMEHeader{
				"Content-Type":        {attachment.ContentType},
				"Content-Disposition": {fmt.Sprintf("attachment; filename=\"%s\"", attachment.Filename)},
				"Content-Transfer-Encoding": {"base64"},
			})
			if err != nil {
				return nil, err
			}

			encoder := base64.NewEncoder(base64.StdEncoding, part)
			encoder.Write(attachment.Content)
			encoder.Close()
		}

		writer.Close()
	} else if hasHTML {
		// 纯HTML邮件
		buf.WriteString("Content-Type: multipart/alternative; boundary=alt-boundary\r\n\r\n")

		// 文本正文（备选）
		if hasText {
			buf.WriteString("--alt-boundary\r\n")
			buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
			buf.WriteString(e.Body + "\r\n\r\n")
		}

		// HTML正文
		buf.WriteString("--alt-boundary\r\n")
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
		buf.WriteString(e.HTMLBody + "\r\n\r\n")
		buf.WriteString("--alt-boundary--\r\n")
	} else {
		// 纯文本邮件
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		buf.WriteString(e.Body + "\r\n")
	}

	return buf.Bytes(), nil
}

// ParseMIME 从MIME格式解析邮件
func ParseMIME(data []byte) (*Email, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	email := NewEmail()

	// 解析头部
	if from, err := msg.Header.AddressList("From"); err == nil && len(from) > 0 {
		email.From = from[0].String()
	}

	if to, err := msg.Header.AddressList("To"); err == nil {
		for _, addr := range to {
			email.To = append(email.To, addr.String())
		}
	}

	if cc, err := msg.Header.AddressList("Cc"); err == nil {
		for _, addr := range cc {
			email.Cc = append(email.Cc, addr.String())
		}
	}

	if subject := msg.Header.Get("Subject"); subject != "" {
		email.Subject = subject
	}

	if date := msg.Header.Get("Date"); date != "" {
		if sentAt, err := time.Parse(time.RFC1123Z, date); err == nil {
			email.SentAt = sentAt
		}
	}

	// 解析正文
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return nil, err
	}

	// 简单处理：假设是纯文本
	email.Body = string(body)

	return email, nil
}

// getContentType 根据文件名获取内容类型
func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt":
		return "text/plain"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	case ".zip":
		return "application/zip"
	case ".rar":
		return "application/x-rar-compressed"
	case ".7z":
		return "application/x-7z-compressed"
	default:
		return "application/octet-stream"
	}
}
