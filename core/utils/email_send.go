package utils

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"

	"github.com/joho/godotenv"
	"github.com/jordan-wright/email"
)

var password string

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	password = os.Getenv("QQ_MAIL_PASSWORD")
}

func SendEmail(emailAddress, code string) error {
	e := email.NewEmail()
	e.From = "Get <2477183238@qq.com>"
	e.To = []string{emailAddress}
	//e.Bcc = []string{"test_bcc@example.com"}
	//e.Cc = []string{"test_cc@example.com"}
	e.Subject = "玺朽GO 邮箱Server"
	//e.Text = []byte("Text Body is, of course, supported!")
	e.HTML = []byte("你的验证码：<h1>" + code + "</h1>")
	err := e.SendWithStartTLS("smtp.qq.com:587", smtp.PlainAuth("", "2477183238@qq.com", password, "smtp.qq.com"),
		&tls.Config{
			ServerName:         "smtp.qq.com",
			InsecureSkipVerify: true,
		})
	if err != nil {
		fmt.Println("发送邮件失败：", err)
	}
	return nil
}
