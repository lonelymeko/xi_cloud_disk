package test

import (
	"context"
	"errors"
	"net"
	"testing"

	"cloud_disk/core/utils"

	"github.com/jordan-wright/email"
)

func TestSendEmail(t *testing.T) {
	oldEnabled, oldHost, oldPort, oldUser, oldPass := utils.EmailEnabled(), utils.EmailHost(), utils.EmailPort(), utils.EmailUser(), utils.EmailPassword()
	oldSender := utils.EmailSender()
	utils.SetEmailConfig(true, "smtp.local", "2525", "u", "p")

	called := false
	utils.SetEmailSender(func(e *email.Email, host, port, user, pass string) error {
		called = true
		if host != "smtp.local" || port != "2525" || user != "u" || pass != "p" {
			return errors.New("bad config")
		}
		return nil
	})

	t.Cleanup(func() {
		utils.SetEmailConfig(oldEnabled, oldHost, oldPort, oldUser, oldPass)
		utils.SetEmailSender(oldSender)
	})

	if err := utils.SendEmail("a@b.com", "123456"); err != nil {
		t.Fatalf("send email failed: %v", err)
	}
	if !called {
		t.Fatal("sender not called")
	}
}

func TestEmailConnectivity(t *testing.T) {
	oldEnabled, oldHost, oldPort, oldUser, oldPass := utils.EmailEnabled(), utils.EmailHost(), utils.EmailPort(), utils.EmailUser(), utils.EmailPassword()
	oldDialer := utils.EmailDialer()
	utils.SetEmailConfig(true, "smtp.local", "2525", "u", "p")

	utils.SetEmailDialer(func(ctx context.Context, network, addr string) (net.Conn, error) {
		c1, c2 := net.Pipe()
		_ = c2.Close()
		return c1, nil
	})

	t.Cleanup(func() {
		utils.SetEmailConfig(oldEnabled, oldHost, oldPort, oldUser, oldPass)
		utils.SetEmailDialer(oldDialer)
	})

	if err := utils.EmailConnectivity(context.Background()); err != nil {
		t.Fatalf("connectivity failed: %v", err)
	}
}
