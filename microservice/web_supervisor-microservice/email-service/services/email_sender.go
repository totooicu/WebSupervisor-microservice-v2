package services

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/totooicu/email-service/models"
)

// buildMessage 构建邮件内容
func buildMessage(content *models.EmailContent, config *models.EmailConfig) []byte {
	var buf bytes.Buffer
	buf.WriteString("From: " + config.Username + "\r\n")
	buf.WriteString("To: " + strings.Join(content.Tos, ",") + "\r\n")
	buf.WriteString("Subject: " + content.Subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	buf.WriteString(content.Body)
	return buf.Bytes()
}

// connectSMTP 建立 SMTP 客户端，支持 465(SSL) 和 587(STARTTLS)
func connectSMTP(config *models.EmailConfig) (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	if config.Port == 465 {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: config.Host})
		if err != nil {
			return nil, err
		}
		return smtp.NewClient(conn, config.Host)
	}

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}
	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(&tls.Config{ServerName: config.Host}); err != nil {
			client.Close()
			return nil, err
		}
	}
	return client, nil
}

// SendEmailByConfig 发送邮件（主流程）
func SendEmailByConfig(content *models.EmailContent, config *models.EmailConfig) error {
	if config.WaitTimeMS > 0 {
		time.Sleep(time.Duration(config.WaitTimeMS) * time.Millisecond)
	}
	if content == nil || config == nil || len(content.Tos) == 0 {
		return fmt.Errorf("invalid content or config")
	}

	client, err := connectSMTP(config)
	if err != nil {
		return err
	}
	defer client.Close()

	// 认证
	if config.Username != "" && config.Password != "" {
		auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
		if err = client.Auth(auth); err != nil {
			return err
		}
	}

	// 设置信封
	if err = client.Mail(config.Username); err != nil {
		return err
	}
	for _, to := range content.Tos {
		if err = client.Rcpt(to); err != nil {
			return err
		}
	}

	// 发送数据
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(buildMessage(content, config))
	if err != nil {
		w.Close()
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return client.Quit()
}