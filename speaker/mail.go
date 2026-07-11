package speaker

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
)

// MailConfig 是 mail 通知所需的配置
type MailConfig struct {
	Host     string // smtp 地址，格式 host:port，例如 smtp.qq.com:465 或 smtp.163.com:25
	User     string // 发件账号（通常就是邮箱地址）
	Password string // 发件密码（或授权码）
	From     string // 发件来源邮箱
	To       string // 收件邮箱
}

// SendMailPlain 通过明文端口（25/587）发送邮件
func SendMailPlain(host, from, to string, auth smtp.Auth, subject, body string) error {
	_, _, err := net.SplitHostPort(host)
	if err != nil {
		return fmt.Errorf("parse mail host: %w", err)
	}
	header := map[string]string{
		"Subject":      subject,
		"From":         from,
		"To":           to,
		"Content-Type": "text/plain; charset=UTF-8",
	}
	message := ""
	for k, v := range header {
		message += k + ": " + v + "\r\n"
	}
	message += "\r\n" + body
	return smtp.SendMail(host, auth, from, []string{to}, []byte(message))
}

// StarttlsAuth 仅在 STARTTLS（587）等明文升级场景使用
type StarttlsAuth struct {
	identity, username, password, host string
}

func (a *StarttlsAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *StarttlsAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, fmt.Errorf("unexpected server challenge: %s", fromServer)
		}
	}
	return nil, nil
}

// SendMail 根据端口自动选择 SSL（465）/ 明文（25/587）发送邮件
func SendMail(cfg MailConfig, subject, body string) error {
	_, port, err := net.SplitHostPort(cfg.Host)
	if err != nil {
		return fmt.Errorf("parse mail host %q: %w", cfg.Host, err)
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid mail port %q: %w", port, err)
	}

	if portNum == 465 {
		// SSL/TLS 直连
		return sendMailTLS(cfg, subject, body)
	}
	auth := smtp.PlainAuth("", cfg.User, cfg.Password, hostnameOf(cfg.Host))
	return SendMailPlain(cfg.Host, cfg.From, cfg.To, auth, subject, body)
}

// sendMailTLS 走 465 端口 SSL 发送
func sendMailTLS(cfg MailConfig, subject, body string) error {
	addr := cfg.Host
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: hostnameOf(addr)})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	host, _, _ := net.SplitHostPort(addr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", cfg.User, cfg.Password, host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(cfg.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(cfg.To); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	defer wc.Close()

	header := fmt.Sprintf("Subject: %s\r\nFrom: %s\r\nTo: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n", subject, cfg.From, cfg.To)
	if _, err := wc.Write([]byte(header + body)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	return nil
}

func hostnameOf(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
