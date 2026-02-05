package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/jordan-wright/email"
)

var (
	emailEnabled  bool
	emailHost     string
	emailPort     string
	emailUser     string
	emailPassword string
	emailSender   func(*email.Email, string, string, string, string) error
	emailDialer   func(ctx context.Context, network, addr string) (net.Conn, error)
)

func init() {
	_ = godotenv.Load(".env")
	switch strings.ToLower(os.Getenv("EMAIL_ENABLED")) {
	case "false", "0", "off", "no":
		emailEnabled = false
	default:
		emailEnabled = true
	}
	emailHost = os.Getenv("EMAIL_HOST")
	emailPort = os.Getenv("EMAIL_PORT")
	emailUser = os.Getenv("EMAIL_USER")
	emailPassword = os.Getenv("EMAIL_PASSWORD")
	if emailSender == nil {
		emailSender = func(e *email.Email, host, port, user, pass string) error {
			addr := fmt.Sprintf("%s:%s", host, port)
			return e.SendWithStartTLS(addr, smtp.PlainAuth("", user, pass, host), &tls.Config{
				ServerName:         host,
				InsecureSkipVerify: true,
			})
		}
	}
	if emailDialer == nil {
		emailDialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 2 * time.Second}
			return d.DialContext(ctx, network, addr)
		}
	}
}

func setEmailConfig(enabled bool, host, port, user, pass string) {
	emailEnabled = enabled
	emailHost = host
	emailPort = port
	emailUser = user
	emailPassword = pass
}

func SetEmailConfig(enabled bool, host, port, user, pass string) {
	setEmailConfig(enabled, host, port, user, pass)
}

func setEmailSender(sender func(*email.Email, string, string, string, string) error) {
	emailSender = sender
}

func SetEmailSender(sender func(*email.Email, string, string, string, string) error) {
	setEmailSender(sender)
}

func setEmailDialer(dialer func(ctx context.Context, network, addr string) (net.Conn, error)) {
	emailDialer = dialer
}

func SetEmailDialer(dialer func(ctx context.Context, network, addr string) (net.Conn, error)) {
	setEmailDialer(dialer)
}

func EmailEnabled() bool {
	return emailEnabled
}

func EmailHost() string {
	return emailHost
}

func EmailPort() string {
	return emailPort
}

func EmailUser() string {
	return emailUser
}

func EmailPassword() string {
	return emailPassword
}

func EmailSender() func(*email.Email, string, string, string, string) error {
	return emailSender
}

func EmailDialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	return emailDialer
}

func SendEmail(emailAddress, code string) error {
	if !emailEnabled {
		return nil
	}
	if emailHost == "" || emailPort == "" || emailUser == "" || emailPassword == "" {
		return fmt.Errorf("email env missing")
	}
	e := email.NewEmail()
	e.From = fmt.Sprintf("CloudDisk <%s>", emailUser)
	e.To = []string{emailAddress}
	//e.Bcc = []string{"test_bcc@example.com"}
	//e.Cc = []string{"test_cc@example.com"}
	e.Subject = "玺朽GO 邮箱Server"
	//e.Text = []byte("Text Body is, of course, supported!")
	e.HTML = []byte("你的验证码：<h1>" + code + "</h1>")
	err := emailSender(e, emailHost, emailPort, emailUser, emailPassword)
	if err != nil {
		return err
	}
	return nil
}

func EmailConnectivity(ctx context.Context) error {
	if !emailEnabled {
		return nil
	}
	if emailHost == "" || emailPort == "" || emailUser == "" || emailPassword == "" {
		return fmt.Errorf("email env missing")
	}
	conn, err := emailDialer(ctx, "tcp", fmt.Sprintf("%s:%s", emailHost, emailPort))
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
