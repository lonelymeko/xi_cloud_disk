# Go-Zero JWT 中间件文件上传性能问题 Bug Report

## 📋 问题描述

在使用 go-zero 自带的 JWT 中间件处理文件上传接口时，发现严重的性能问题：
- 整个文件内容被读取并打印到控制台
- JWT 验证耗时极长（几秒到几十秒）
- 内存占用过高（文件大小级别）

## 🔍 复现步骤

### 1. API 定义
```go
@server (
    prefix: /api/file
    jwt: Auth  // 使用内置 JWT 中间件
)
service core-api {
    @handler UploadFileHandler
    post /upload (UploadFileRequest) returns (UploadFileResponse)
}
```

### 2. 上传文件
```bash
curl -X POST http://localhost:8888/api/file/upload \
  -H "Authorization: Bearer <token>" \
  -F "file=@video.mp4"  # 100MB 视频
```

### 3. 观察现象
- ❌ 控制台打印大量二进制数据
- ❌ 响应时间 5-30 秒
- ❌ 内存占用飙升

## 🎯 期望行为

- ✅ JWT 中间件只检查 Header 和 Query 参数
- ✅ 快速响应（秒级）
- ✅ 控制台干净，无文件内容
- ✅ 低内存占用

## 🐛 根本原因

Go-Zero 的 JWT 中间件会从 **Form 表单**中查找 token：

```go
// 当前实现（伪代码）
func (h *AuthorizeHandler) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. 检查 Authorization Header ✅
        token := r.Header.Get("Authorization")
        
        // 2. 检查 Query 参数 ✅
        if token == "" {
            token = r.URL.Query().Get("token")
        }
        
        // 3. 检查 Form 表单 ❌ 这里有问题！
        if token == "" {
            token = r.FormValue("token")  // 会读取整个 multipart/form-data！
        }
    }
}
```

**问题**：`r.FormValue("token")` 会解析整个 `multipart/form-data` 请求体，包括所有上传的文件，仅仅是为了检查表单中是否有 `token` 字段。

## 💥 影响

这个 bug 导致：
- ❌ 无法在生产环境使用 JWT 中间件处理文件上传
- ❌ 大文件上传几乎不可用
- ❌ 高并发场景下服务器压力巨大
- ❌ 用户体验极差

## 💡 建议的解决方案

### 方案 1：根据 Content-Type 跳过 Form 解析（推荐）

```go
func (h *AuthorizeHandler) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        
        if token == "" {
            token = r.URL.Query().Get("token")
        }
        
        // 仅在非 multipart/form-data 时检查 Form
        contentType := r.Header.Get("Content-Type")
        if token == "" && !strings.Contains(contentType, "multipart/form-data") {
            token = r.FormValue("token")
        }
        
        // 继续验证...
    }
}
```

### 方案 2：添加配置选项

```yaml
Auth:
  AccessSecret: xxx
  AccessExpire: 36000
  SkipFormParsing: true  # 新增配置，跳过 Form 解析
```

### 方案 3：提供专用中间件

为文件上传场景提供一个轻量级的 JWT 中间件，只检查 Header 和 Query 参数。

## 🔧 当前解决方法（Workaround）

用户必须自己实现一个自定义中间件：

```go
// 自定义文件上传认证中间件
type FileAuthMiddleware struct {
    accessSecret string
    accessExpire int64
}

func (m *FileAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 只从 Header 和 Query 获取 token
        token := r.Header.Get("Authorization")
        token = strings.TrimPrefix(token, "Bearer ")
        
        if token == "" {
            token = r.Header.Get("X-Token")
        }
        
        if token == "" {
            token = r.URL.Query().Get("token")
        }
        
        // 不调用 r.FormValue()，避免读取文件！
        
        // 验证 token...
    }
}
```

然后在 API 中使用：
```go
@server (
    prefix: /api/file
    middleware: FileAuthMiddleware  // 使用自定义中间件
)
```

## 📊 性能对比

| 指标 | 内置 JWT | 自定义中间件 | 提升 |
|------|---------|-------------|------|
| 认证耗时 | 5-30秒 | < 100ms | **99%+** |
| 内存占用 | 文件大小 | < 1MB | **显著降低** |
| 控制台输出 | 大量二进制 | 干净 | ✅ |

## 🌍 相关案例

这是 Web 框架中的常见问题：
- Express.js：中间件顺序影响 multipart 解析
- Django：文件上传需要自定义认证
- Spring Boot：需要配置 MultipartResolver

## 🔗 相关信息

- **Go-Zero 版本**：1.9.2 (goctl 1.9.2)
- **Go 版本**：1.20+
- **影响范围**：所有使用 JWT + 文件上传的项目

## 📝 总结

这个问题严重影响了 go-zero 在文件密集型应用中的可用性。许多现代应用需要：
- ✅ 认证保护的文件上传
- ✅ 大文件支持（视频、数据集、备份）
- ✅ 良好的性能和用户体验

当前的 JWT 中间件使得同时实现这三点变得困难。

## 🤝 我愿意提交 PR

如果维护团队同意解决方案，我很乐意贡献代码修复这个问题。

---

## 📎 附件

- [详细技术文档](./JWT中间件优化-文件上传认证.md)
- [完整 Bug Report（英文）](./go-zero-bug-report.md)
