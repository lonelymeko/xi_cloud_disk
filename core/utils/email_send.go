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
	addr := fmt.Sprintf("%s:%s", emailHost, emailPort)
	err := e.SendWithStartTLS(addr, smtp.PlainAuth("", emailUser, emailPassword, emailHost),
		&tls.Config{
			ServerName:         emailHost,
			InsecureSkipVerify: true,
		})
	if err != nil {
		return err
	}
	return nil
}

func EmailEnabled() bool { return emailEnabled }

func EmailConnectivity(ctx context.Context) error {
	if !emailEnabled {
		return nil
	}
	if emailHost == "" || emailPort == "" || emailUser == "" || emailPassword == "" {
		return fmt.Errorf("email env missing")
	}
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%s", emailHost, emailPort))
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
