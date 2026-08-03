package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// Config 邮件客户端配置
type Config struct {
	SMTPHost      string        `yaml:"smtp_host" json:"smtp_host"`
	SMTPPort      int           `yaml:"smtp_port" json:"smtp_port"`
	POP3Host      string        `yaml:"pop3_host" json:"pop3_host"`
	POP3Port      int           `yaml:"pop3_port" json:"pop3_port"`
	IMAPHost      string        `yaml:"imap_host" json:"imap_host"`
	IMAPPort      int           `yaml:"imap_port" json:"imap_port"`
	Username      string        `yaml:"username" json:"username"`
	Password      string        `yaml:"password" json:"password"`
	UseTLS        bool          `yaml:"use_tls" json:"use_tls"`
	AuthType      string        `yaml:"auth_type" json:"auth_type"` // "plain", "login", "cram-md5"
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
	MaxRetries    int           `yaml:"max_retries" json:"max_retries"`
	RetryInterval time.Duration `yaml:"retry_interval" json:"retry_interval"`
}

// NewConfig 创建新的邮件客户端配置
func NewConfig(smtpHost string, smtpPort int, pop3Host string, pop3Port int, imapHost string, imapPort int, username, password string, useTLS bool, authType string, timeout time.Duration, maxRetries int, retryInterval time.Duration) *Config {
	return &Config{
		SMTPHost:      smtpHost,
		SMTPPort:      smtpPort,
		POP3Host:      pop3Host,
		POP3Port:      pop3Port,
		IMAPHost:      imapHost,
		IMAPPort:      imapPort,
		Username:      username,
		Password:      password,
		UseTLS:        useTLS,
		AuthType:      authType,
		Timeout:       timeout,
		MaxRetries:    maxRetries,
		RetryInterval: retryInterval,
	}
}

// Client 邮件客户端
type Client struct {
	smtpHost      string
	smtpPort      int
	pop3Host      string
	pop3Port      int
	imapHost      string
	imapPort      int
	username      string
	password      string
	useTLS        bool
	authType      string // "plain", "login", "cram-md5"
	timeout       time.Duration
	maxRetries    int
	retryInterval time.Duration
}

// NewClientWithConfig 通过配置创建邮件客户端
func NewClientWithConfig(config *Config) *Client {
	return &Client{
		smtpHost:      config.SMTPHost,
		smtpPort:      config.SMTPPort,
		pop3Host:      config.POP3Host,
		pop3Port:      config.POP3Port,
		imapHost:      config.IMAPHost,
		imapPort:      config.IMAPPort,
		username:      config.Username,
		password:      config.Password,
		useTLS:        config.UseTLS,
		authType:      config.AuthType,
		timeout:       config.Timeout,
		maxRetries:    config.MaxRetries,
		retryInterval: config.RetryInterval,
	}
}

// SetSMTP 设置SMTP服务器
func (c *Client) SetSMTP(host string, port int) *Client {
	c.smtpHost = host
	c.smtpPort = port
	return c
}

// SetPOP3 设置POP3服务器
func (c *Client) SetPOP3(host string, port int) *Client {
	c.pop3Host = host
	c.pop3Port = port
	return c
}

// SetIMAP 设置IMAP服务器
func (c *Client) SetIMAP(host string, port int) *Client {
	c.imapHost = host
	c.imapPort = port
	return c
}

// SetAuth 设置认证信息
func (c *Client) SetAuth(username, password string) *Client {
	c.username = username
	c.password = password
	return c
}

// SetTLS 设置是否使用TLS
func (c *Client) SetTLS(useTLS bool) *Client {
	c.useTLS = useTLS
	return c
}

// SetAuthType 设置认证类型
func (c *Client) SetAuthType(authType string) *Client {
	c.authType = authType
	return c
}

// SetTimeout 设置超时时间
func (c *Client) SetTimeout(timeout time.Duration) *Client {
	c.timeout = timeout
	return c
}

// SetRetry 设置重试参数
func (c *Client) SetRetry(maxRetries int, retryInterval time.Duration) *Client {
	c.maxRetries = maxRetries
	c.retryInterval = retryInterval
	return c
}

// Send 发送邮件（至少尝试一次，maxRetries 为额外重试次数）
func (c *Client) Send(email *Email) error {
	var err error
	attempts := c.maxRetries + 1
	if attempts < 1 {
		attempts = 1
	}
	for i := 0; i < attempts; i++ {
		err = c.send(email)
		if err == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(c.retryInterval)
		}
	}
	return err
}

// send 实际发送邮件的方法
func (c *Client) send(email *Email) error {
	// 构建收件人列表（安全复制，避免修改 email.To 底层数组）
	recipients := make([]string, 0, len(email.To)+len(email.Cc)+len(email.Bcc))
	recipients = append(recipients, email.To...)
	recipients = append(recipients, email.Cc...)
	recipients = append(recipients, email.Bcc...)

	// 准备认证信息
	auth := smtp.PlainAuth("", c.username, c.password, c.smtpHost)

	// 连接SMTP服务器
	addr := fmt.Sprintf("%s:%d", c.smtpHost, c.smtpPort)
	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	// 启用TLS
	if c.useTLS {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: c.smtpHost,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}

	// 认证
	if err := client.Auth(auth); err != nil {
		return err
	}

	// 设置发件人
	if err := client.Mail(email.From); err != nil {
		return err
	}

	// 设置收件人
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}

	// 发送邮件内容
	data, err := email.ToMIME()
	if err != nil {
		return err
	}

	// 写入数据
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	// 退出
	return client.Quit()
}

// ReceivePOP3 通过POP3接收邮件
//
//nolint:gocyclo // POP3 协议状态机分支多，拆分收益低
func (c *Client) ReceivePOP3(maxMessages int) ([]*Email, error) {
	// 构建POP3服务器地址
	addr := fmt.Sprintf("%s:%d", c.pop3Host, c.pop3Port)

	// 连接POP3服务器
	conn, err := net.DialTimeout("tcp", addr, c.timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 启用TLS
	if c.useTLS {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: c.pop3Host,
		}
		conn = tls.Client(conn, tlsConfig)
	}

	// 创建POP3客户端
	client := textproto.NewConn(conn)
	defer client.Close()

	// 读取欢迎信息
	_, err = client.ReadLine()
	if err != nil {
		return nil, err
	}

	// 认证
	if err := client.PrintfLine("USER %s", c.username); err != nil {
		return nil, err
	}
	_, err = client.ReadLine()
	if err != nil {
		return nil, err
	}

	if err := client.PrintfLine("PASS %s", c.password); err != nil {
		return nil, err
	}
	_, err = client.ReadLine()
	if err != nil {
		return nil, err
	}

	// 获取邮件数量
	if err := client.PrintfLine("STAT"); err != nil {
		return nil, err
	}
	statResp, err := client.ReadLine()
	if err != nil {
		return nil, err
	}

	// 解析邮件数量
	var count int
	_, err = fmt.Sscanf(statResp, "+OK %d", &count)
	if err != nil {
		return nil, err
	}

	// 限制邮件数量
	if maxMessages > 0 && count > maxMessages {
		count = maxMessages
	}

	// 接收邮件
	emails := make([]*Email, 0, count)
	for i := 1; i <= count; i++ {
		// 获取邮件内容
		if err := client.PrintfLine("RETR %d", i); err != nil {
			return nil, err
		}

		// 读取邮件内容
		var data []byte
		for {
			line, err := client.ReadLine()
			if err != nil {
				break
			}
			if line == "." {
				break
			}
			// 处理转义
			line = strings.TrimPrefix(line, ".")
			data = append(data, []byte(line+"\r\n")...)
		}

		// 解析邮件
		email, err := ParseMIME(data)
		if err != nil {
			continue
		}

		emails = append(emails, email)
	}

	// 退出
	if err := client.PrintfLine("QUIT"); err != nil {
		return nil, err
	}
	_, err = client.ReadLine()
	if err != nil {
		return nil, err
	}

	return emails, nil
}

// ReceiveIMAP 通过IMAP接收邮件
//
//nolint:gocyclo // IMAP 协议状态机分支多，拆分收益低
func (c *Client) ReceiveIMAP(folder string, maxMessages int) ([]*Email, error) {
	// 构建IMAP服务器地址
	addr := fmt.Sprintf("%s:%d", c.imapHost, c.imapPort)

	// 连接IMAP服务器
	conn, err := net.DialTimeout("tcp", addr, c.timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 启用TLS
	if c.useTLS {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: c.imapHost,
		}
		conn = tls.Client(conn, tlsConfig)
	}

	// 创建IMAP客户端
	client := textproto.NewConn(conn)
	defer client.Close()

	// 读取欢迎信息
	_, err = client.ReadLine()
	if err != nil {
		return nil, err
	}

	// 认证
	if err := client.PrintfLine("A001 LOGIN %s %s", c.username, c.password); err != nil {
		return nil, err
	}
	_, err = client.ReadLine()
	if err != nil {
		return nil, err
	}

	// 选择文件夹
	if folder == "" {
		folder = "INBOX"
	}
	if err := client.PrintfLine("A002 SELECT %s", folder); err != nil {
		return nil, err
	}
	_, err = client.ReadLine()
	if err != nil {
		return nil, err
	}

	// 搜索邮件
	if err := client.PrintfLine("A003 SEARCH ALL"); err != nil {
		return nil, err
	}

	// 读取搜索结果
	var messageIDs []string
	for {
		line, err := client.ReadLine()
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "*") {
			parts := strings.Split(line, " ")
			messageIDs = append(messageIDs, parts[1:len(parts)-1]...)
		}
		if strings.HasPrefix(line, "A003 OK") {
			break
		}
	}

	// 限制邮件数量
	if maxMessages > 0 && len(messageIDs) > maxMessages {
		messageIDs = messageIDs[len(messageIDs)-maxMessages:]
	}

	// 接收邮件
	emails := make([]*Email, 0, len(messageIDs))
	for _, id := range messageIDs {
		// 获取邮件内容
		if err := client.PrintfLine("A004 FETCH %s BODY[]", id); err != nil {
			continue
		}

		// 读取邮件内容
		var data []byte
		for {
			line, err := client.ReadLine()
			if err != nil {
				break
			}
			if strings.HasPrefix(line, "A004 OK") {
				break
			}
			// 跳过头部行
			if strings.HasPrefix(line, "*") && strings.Contains(line, "FETCH") {
				continue
			}
			if line == ")" {
				continue
			}
			data = append(data, []byte(line+"\r\n")...)
		}

		// 解析邮件
		email, err := ParseMIME(data)
		if err != nil {
			continue
		}

		emails = append(emails, email)
	}

	// 退出
	if err := client.PrintfLine("A005 LOGOUT"); err != nil {
		return nil, err
	}
	_, err = client.ReadLine()
	if err != nil {
		return nil, err
	}

	return emails, nil
}

// GetUnreadCountIMAP 获取未读邮件数量
func (c *Client) GetUnreadCountIMAP(folder string) (int, error) {
	// 构建IMAP服务器地址
	addr := fmt.Sprintf("%s:%d", c.imapHost, c.imapPort)

	// 连接IMAP服务器
	conn, err := net.DialTimeout("tcp", addr, c.timeout)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	// 启用TLS
	if c.useTLS {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: c.imapHost,
		}
		conn = tls.Client(conn, tlsConfig)
	}

	// 创建IMAP客户端
	client := textproto.NewConn(conn)
	defer client.Close()

	// 读取欢迎信息
	_, err = client.ReadLine()
	if err != nil {
		return 0, err
	}

	// 认证
	if err := client.PrintfLine("A001 LOGIN %s %s", c.username, c.password); err != nil {
		return 0, err
	}
	_, err = client.ReadLine()
	if err != nil {
		return 0, err
	}

	// 选择文件夹
	if folder == "" {
		folder = "INBOX"
	}
	if err := client.PrintfLine("A002 SELECT %s", folder); err != nil {
		return 0, err
	}

	// 读取邮箱状态
	var unreadCount int
	for {
		line, err := client.ReadLine()
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "*") {
			// 解析状态行，格式如: * 10 EXISTS
			if strings.Contains(line, "RECENT") {
				_, err := fmt.Sscanf(line, "* %d RECENT", &unreadCount)
				if err == nil {
					break
				}
			}
		}
		if strings.HasPrefix(line, "A002 OK") {
			break
		}
	}

	// 退出
	if err := client.PrintfLine("A003 LOGOUT"); err != nil {
		return 0, err
	}
	_, err = client.ReadLine()
	if err != nil {
		return 0, err
	}

	return unreadCount, nil
}

// DeleteMessagePOP3 通过POP3删除邮件
func (c *Client) DeleteMessagePOP3(messageID int) error {
	// 构建POP3服务器地址
	addr := fmt.Sprintf("%s:%d", c.pop3Host, c.pop3Port)

	// 连接POP3服务器
	conn, err := net.DialTimeout("tcp", addr, c.timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 启用TLS
	if c.useTLS {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: c.pop3Host,
		}
		conn = tls.Client(conn, tlsConfig)
	}

	// 创建POP3客户端
	client := textproto.NewConn(conn)
	defer client.Close()

	// 读取欢迎信息
	_, err = client.ReadLine()
	if err != nil {
		return err
	}

	// 认证
	if err := client.PrintfLine("USER %s", c.username); err != nil {
		return err
	}
	_, err = client.ReadLine()
	if err != nil {
		return err
	}

	if err := client.PrintfLine("PASS %s", c.password); err != nil {
		return err
	}
	_, err = client.ReadLine()
	if err != nil {
		return err
	}

	// 删除邮件
	if err := client.PrintfLine("DELE %d", messageID); err != nil {
		return err
	}
	_, err = client.ReadLine()
	if err != nil {
		return err
	}

	// 退出
	if err := client.PrintfLine("QUIT"); err != nil {
		return err
	}
	_, err = client.ReadLine()
	if err != nil {
		return err
	}

	return nil
}
