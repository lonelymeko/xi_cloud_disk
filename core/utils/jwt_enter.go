package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// JwtPayLoad JWT Payload 结构
type JwtPayLoad struct {
	Id       int
	Identity string
	Name     string
}

// CustomClaims 自定义JWT声明
type CustomClaims struct {
	jwt.RegisteredClaims
	JwtPayLoad
}

// GenToken 生成JWT Token
func GenToken(user JwtPayLoad, accessSecret string, expires int64) (string, error) {
	claims := CustomClaims{
		JwtPayLoad: user,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expires))),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(accessSecret))
}

// ParseToken 解析JWT Token
func ParseToken(tokenStr string, accessSecret string, expires int64) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(accessSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
