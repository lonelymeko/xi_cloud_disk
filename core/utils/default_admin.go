package utils

import (
	"cloud_disk/core/models"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"

	"xorm.io/xorm"
)

// EnsureDefaultAdmin 确保默认管理员账号存在。
func EnsureDefaultAdmin(engine *xorm.Engine) error {
	name := os.Getenv("ADMIN_NAME")
	if name == "" {
		name = "admin"
	}
	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		email = "admin@example.com"
	}
	pass := os.Getenv("ADMIN_PASSWORD")
	if pass == "" {
		pass = randomPassword(12)
	}

	u := new(models.UserBasic)
	has, err := engine.Where("email = ?", email).Get(u)
	if err != nil {
		return err
	}
	if has {
		return nil
	}

	user := &models.UserBasic{
		Name:     name,
		Email:    email,
		Password: Md5(pass),
		Identity: UUID(),
		Role:     "admin",
	}
	_, err = engine.InsertOne(user)
	if err != nil {
		return err
	}

	fmt.Println("================ 默认最高管理用户已创建 ================")
	fmt.Printf("用户名: %s\n", name)
	fmt.Printf("邮箱:   %s\n", email)
	fmt.Printf("密码:   %s\n", pass)
	fmt.Println("=====================================================")
	return nil
}

// randomPassword 生成随机密码。
func randomPassword(length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	out := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			out[i] = 'x'
			continue
		}
		out[i] = letters[n.Int64()]
	}
	return string(out)
}
