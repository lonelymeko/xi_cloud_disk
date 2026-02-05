// goctl 生成代码，可安全编辑。
// goctl 1.9.2

package config

import "github.com/zeromicro/go-zero/rest"

// Config 服务配置。
type Config struct {
	rest.RestConf
	Auth struct {
		// AccessSecret JWT 密钥。
		AccessSecret string
		// AccessExpire JWT 过期时间（小时）。
		AccessExpire int64
	}
	MySQL struct {
		// DataSource MySQL 连接串。
		DataSource string
	}
	Redis struct {
		// Addr Redis 地址。
		Addr string
		// Password Redis 密码。
		Password string
		// DB Redis 数据库编号。
		DB int
	}
	RabbitMQ struct {
		Host     string
		Port     int
		Username string
		Password string
		Vhost    string
	}
}
