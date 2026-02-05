package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"cloud_disk/core/utils"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// FileAuthMiddleware 文件上传专用的认证中间件。
// 只从 Header 和 Query 参数中获取 token，不读取表单。
type FileAuthMiddleware struct {
	accessSecret string
	accessExpire int64
}

// NewFileAuthMiddleware 创建文件上传认证中间件。
func NewFileAuthMiddleware(accessSecret string, accessExpire int64) *FileAuthMiddleware {
	return &FileAuthMiddleware{
		accessSecret: accessSecret,
		accessExpire: accessExpire,
	}
}

// Handle 实现认证处理。
func (m *FileAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 优先从 Authorization Header 获取 token
		token := r.Header.Get("Authorization")
		if token != "" {
			// 移除 "Bearer " 前缀
			token = strings.TrimPrefix(token, "Bearer ")
			token = strings.TrimSpace(token)
		}

		// 2. 如果 Header 没有，从 X-Token Header 获取
		if token == "" {
			token = r.Header.Get("X-Token")
		}

		// 3. 如果 Header 都没有，从 Query 参数获取
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		// 4. 如果还是没有 token，返回未授权
		if token == "" {
			httpx.ErrorCtx(r.Context(), w, errors.New("未授权访问，请提供有效的 token"))
			return
		}

		// 5. 验证 token
		claims, err := utils.ParseToken(token, m.accessSecret, m.accessExpire)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, errors.New("token 无效或已过期"))
			return
		}

		// 6. 将用户信息存入 context
		ctx := context.WithValue(r.Context(), "user_id", claims.Id)
		ctx = context.WithValue(ctx, "user_identity", claims.Identity)
		ctx = context.WithValue(ctx, "user_name", claims.Name)
		r = r.WithContext(ctx)

		// 打印日志（可选）
		// logx.Infof("用户 %s (ID: %d) 正在上传文件", claims.Name, claims.Id)

		// 验证通过，继续处理
		next(w, r)
	}
}
