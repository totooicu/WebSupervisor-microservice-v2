package MyTool

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

func SendEmail(userName, password string, tos []string, subject, content string) error {
	auth := smtp.PlainAuth("", userName, password, "smtp.qq.com")

	msg := fmt.Sprintf("From: %s\r\n", userName)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(tos, ","))
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "Content-Type: text/html; charset=UTF-8\r\n"
	msg += "\r\n"
	msg += content

	// 使用SSL/TLS加密连接
	tlsConfig := &tls.Config{
		ServerName: "smtp.qq.com",
	}

	conn, err := tls.Dial("tcp", "smtp.qq.com:465", tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %v", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, "smtp.qq.com")
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}

	if err := client.Mail(userName); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}

	for _, to := range tos {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient %s: %v", to, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start data: %v", err)
	}
	defer w.Close()

	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}

	return nil
}
