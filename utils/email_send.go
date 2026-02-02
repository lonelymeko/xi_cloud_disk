package utils

import (
	"crypto/tls"
	"net/smtp"

	"github.com/joho/godotenv"
	"github.com/jordan-wright/email"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
}

func SendEmail(emailAddress, code string) error {
	e := email.NewEmail()
	e.From = "Get <2477183238@qq.com>"
	e.To = []string{emailAddress}
	//e.Bcc = []string{"test_bcc@example.com"}
	//e.Cc = []string{"test_cc@example.com"}
	e.Subject = "GO 邮箱测试"
	//e.Text = []byte("Text Body is, of course, supported!")
	e.HTML = []byte("你的验证码：<h1>" + code + "</h1>")
	err := e.SendWithStartTLS("smtp.qq.com:587", smtp.PlainAuth("", "2477183238@qq.com", "", "smtp.qq.com"),
		&tls.Config{
			ServerName:         "smtp.qq.com",
			InsecureSkipVerify: true,
		})
	if err != nil {

	}
}
