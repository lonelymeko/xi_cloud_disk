package test

import (
	"crypto/tls"
	"net/smtp"
	"testing"

	"github.com/jordan-wright/email"
)

func TestEmail(t *testing.T) {
	e := email.NewEmail()
	e.From = "Get <2477183238@qq.com>"
	e.To = []string{"2477183238@qq.com"}
	//e.Bcc = []string{"test_bcc@example.com"}
	//e.Cc = []string{"test_cc@example.com"}
	e.Subject = "GO 邮箱测试"
	//e.Text = []byte("Text Body is, of course, supported!")
	e.HTML = []byte("你的验证码：<h1>345627</h1>")
	err := e.SendWithStartTLS("smtp.qq.com:587", smtp.PlainAuth("", "2477183238@qq.com", "mfylldqkhapydjja", "smtp.qq.com"),
		&tls.Config{
			ServerName:         "smtp.qq.com",
			InsecureSkipVerify: true,
		})
	if err != nil {
		t.Fatal(err)
	}

}
