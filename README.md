# 玺云盘 ☁️

> 基于 Go-Zero + MySQL + Redis + Aliyun OSS 的轻量级云盘系统

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Go-Zero](https://img.shields.io/badge/Go--Zero-v1.6.6-blue)](https://go-zero.dev)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 📖 项目简介

玺云盘是一个功能完善的个人云存储系统，支持文件上传、管理、分享等核心功能。项目采用 Go-Zero 微服务框架，结合阿里云 OSS 对象存储，实现了高性能、可扩展的云盘服务。

### ✨ 核心特性

- 🚀 **智能压缩**：视频自动压缩（ffmpeg H.264）、图片智能缩放（最大 1920x1080）
- ⚡ **秒传机制**：基于 MD5 hash 的文件去重，相同文件无需重复上传
- 📁 **文件夹管理**：多级目录结构、递归删除、批量操作
- 🔗 **文件分享**：支持链接分享、过期时间控制、资源保存
- 🔐 **JWT 认证**：自定义中间件，避免 Go-Zero 内置 JWT 的性能问题
- 🗄️ **双表架构**：`repository_pool`（全局文件池）+ `user_repository`（用户关联），实现文件去重

---

## 🏗️ 技术栈

| 技术 | 版本 | 说明 |
|------|------|------|
| **Go** | 1.20+ | 后端语言 |
| **Go-Zero** | v1.6.6 | 微服务框架 |
| **MySQL** | 8.0+ | 主数据库（支持 CTE 递归查询） |
| **Redis** | v9 | 缓存 + 验证码存储 |
| **Xorm** | latest | ORM 框架 |
| **Aliyun OSS** | SDK v2 | 对象存储 |
| **FFmpeg** | 4.0+ | 视频压缩 |
| **golang.org/x/image** | latest | 图片压缩 |

---

## 🚀 快速开始

### 1. 环境要求

- Go 1.20+
- MySQL 8.0+
- Redis 5.0+
- FFmpeg 4.0+（可选，用于视频压缩）

### 2. 安装依赖

```bash
# 克隆项目
git clone https://github.com/your-username/cloud_disk.git
cd cloud_disk/core

# 安装 Go 依赖
go mod download

# 或手动安装
go get xorm.io/xorm
go get github.com/jordan-wright/email
go get github.com/go-redis/redis/v8
go get github.com/satori/go.uuid
go get github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss
go get golang.org/x/image
```

### 3. 配置文件

创建 `.env` 文件并配置：

```bash
# 阿里云 OSS 配置
OSS_ACCESS_KEY_ID=your_access_key
OSS_ACCESS_KEY_SECRET=your_secret_key
OSS_BUCKET_NAME=your_bucket
OSS_REGION=oss-cn-beijing

# MySQL 配置
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=cloud_disk

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# 邮箱配置（用于验证码）
EMAIL_HOST=smtp.qq.com
EMAIL_PORT=587
EMAIL_USER=your_email@qq.com
EMAIL_PASSWORD=your_auth_code
```

### 4. 数据库初始化

```sql
-- 执行 SQL 脚本（位于 docs/database/ 目录）
source docs/database/schema.sql
```

### 5. 运行服务

```bash
# 开发模式
go run core.go -f etc/core-api.yaml

# 编译运行
go build -o cloud_disk core.go
./cloud_disk -f etc/core-api.yaml
```

服务启动后，访问 `http://localhost:8888`

---

## 📁 项目结构

```
cloud_disk/
├── core/                      # 核心服务
│   ├── core.api              # API 定义文件
│   ├── core.go               # 主入口
│   ├── common/               # 公共组件
│   │   └── response.go       # 统一响应处理
│   ├── internal/
│   │   ├── config/           # 配置
│   │   ├── handler/          # HTTP 处理器
│   │   ├── logic/            # 业务逻辑
│   │   ├── middleware/       # 中间件（JWT 认证）
│   │   ├── svc/              # 服务上下文
│   │   └── types/            # 请求/响应类型
│   ├── models/               # 数据模型
│   ├── utils/                # 工具函数
│   │   ├── email_send.go     # 邮件发送
│   │   ├── jwt_enter.go      # JWT 工具
│   │   ├── md5_encode.go     # MD5 加密
│   │   └── upload_to_oss.go  # OSS 上传
│   └── docs/                 # 文档
│       └── api/              # OpenAPI 文档
│           ├── user.yaml     # 用户服务 API
│           ├── file.yaml     # 文件服务 API
│           ├── share.yaml    # 分享服务 API
│           └── README.md     # API 文档说明
├── docs/                     # 项目文档
│   ├── 数据库架构设计.md
│   ├── 文件夹下载方案.md
│   └── 代码审查-递归删除问题分析.md
├── test/                     # 测试代码
├── go.mod
└── README.md
```

---

## 🎯 核心功能

### 1. 用户认证

- ✅ 用户注册（邮箱验证码）
- ✅ 用户登录（JWT Token）
- ✅ 用户信息查询
- ✅ 自定义 JWT 中间件（避免 Go-Zero 内置 JWT 性能问题）

### 2. 文件管理

- ✅ 文件上传（支持智能压缩和秒传）
- ✅ 文件列表（分页 + 文件夹筛选）
- ✅ 文件重命名
- ✅ 文件移动
- ✅ 文件/文件夹删除（CTE 递归优化）

### 3. 文件夹操作

- ✅ 创建文件夹（多级目录）
- ✅ 递归删除文件夹（使用 CTE 递归查询，性能提升 95%）
- ✅ 移动文件夹
- ⏳ 文件夹下载（异步打包）- 详见 `docs/文件夹下载方案.md`

### 4. 文件分享

- ✅ 创建分享链接（支持过期时间）
- ✅ 获取分享详情（公开访问）
- ✅ 保存分享资源到个人网盘

---

## 🌟 项目亮点

### 架构设计

1. **自定义 JWT 中间件**
   - 问题：Go-Zero 内置 JWT 会读取整个 multipart/form-data，导致大文件上传性能问题
   - 解决：自定义 `FileAuthMiddleware`，只在需要时解析请求体
   - 参考：[GitHub Issue #5401](https://github.com/zeromicro/go-zero/issues/5401)

2. **统一响应处理**
   - 修改 Go-Zero 代码生成模板，添加 `common.Response()` 统一处理
   - 避免在每个 handler 中重复封装响应格式
   - 自动处理错误码和消息

3. **双表架构设计**
   ```
   repository_pool (全局文件存储池)
   ├── hash (唯一索引) - 实现文件去重
   └── path (OSS 路径)
   
   user_repository (用户文件关联表)
   ├── user_identity (用户 ID)
   ├── repository_identity (关联 repository_pool)
   └── parent_id (文件夹层级)
   ```
   - **优势：** 文件去重、秒传、独立管理

### 性能优化

1. **CTE 递归查询优化删除**
   ```sql
   WITH RECURSIVE folder_tree AS (
       SELECT id FROM user_repository WHERE identity = ?
       UNION ALL
       SELECT ur.id FROM user_repository ur
       INNER JOIN folder_tree ft ON ur.parent_id = ft.id
   )
   DELETE FROM user_repository WHERE id IN (SELECT id FROM folder_tree);
   ```
   - **性能提升：** 95%（避免 N+1 查询）

2. **智能压缩**
   - **视频：** ffmpeg H.264 CRF=23，音频 128k
   - **图片：** 最大 1920x1080，JPEG 质量 85
   - **节省空间：** 平均压缩率 60%

3. **秒传机制**
   - 基于 MD5 hash 判断文件是否已存在
   - 相同文件直接返回，无需上传
   - **用户体验：** 大文件秒传完成

---

## 📚 API 文档

完整的 OpenAPI 3.0 文档位于 `core/docs/api/` 目录：

- **[user.yaml](core/docs/api/user.yaml)** - 用户服务（登录、注册、验证码）
- **[file.yaml](core/docs/api/file.yaml)** - 文件服务（上传、管理、文件夹操作）
- **[share.yaml](core/docs/api/share.yaml)** - 分享服务（创建分享、保存资源）

### 快速查看

```bash
# 使用 Swagger UI
npx swagger-ui-watcher core/docs/api/file.yaml

# 使用 Redoc
npx redoc-cli serve core/docs/api/file.yaml
```

### 导入到测试工具

- **Postman**：File → Import → 选择 YAML 文件
- **Apifox**：导入 → OpenAPI → 选择 YAML 文件

---

## 🔧 开发指南

### 生成代码

```bash
cd core

# 根据 core.api 生成代码
goctl api go -api core.api -dir . -style go_zero
```

### 数据库迁移

```bash
# 使用 xorm 工具生成模型
xorm reverse mysql "root:password@tcp(127.0.0.1:3306)/cloud_disk?charset=utf8mb4" templates/goxorm
```

### 自定义模板

项目使用自定义的 Go-Zero 模板，位于 `templates/` 目录（如果有）。

---

## 📋 TODO

### 高优先级

- [ ] **文件夹下载**：异步任务 + 后台打包（详见 `docs/文件夹下载方案.md`）
- [ ] **分片上传**：支持大文件（> 1GB）断点续传
- [ ] **上传进度推送**：WebSocket 实时推送进度
- [ ] **Redis 缓存优化**：缓存文件列表 COUNT 结果

### 中优先级

- [ ] **异步压缩**：通过 MQ 推送压缩任务，避免阻塞上传
- [ ] **文件预览**：支持图片、视频、PDF 在线预览
- [ ] **回收站功能**：软删除文件可恢复
- [ ] **文件版本管理**：保留文件历史版本

### 低优先级

- [ ] **下载统计**：Redis 维护文件下载次数
- [ ] **分享密码**：为分享链接添加密码保护
- [ ] **批量操作**：批量删除、移动、下载
- [ ] **容量配额**：用户存储空间限制

---

## 🐛 已知问题

1. ~~删除文件夹使用循环递归，存在 N+1 查询问题~~ ✅ 已修复（使用 CTE 递归）
2. ~~图片上传报 "file already closed" 错误~~ ✅ 已修复（同步上传，移除 goroutine）
3. ~~文件大小显示为原始大小，未使用压缩后大小~~ ✅ 已修复

---

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

---

## 📄 开源协议

本项目采用 MIT 协议，详见 [LICENSE](LICENSE) 文件。

---

## 📞 联系方式

- **作者：** xixiu
- **博客：** [https://lonelymeko.top/blog]
- **Email：** [your-email@example.com]

---

## 🙏 致谢

- [Go-Zero](https://go-zero.dev) - 优秀的 Go 微服务框架
- [Xorm](https://xorm.io) - 简洁的 ORM 框架
- [Aliyun OSS](https://www.aliyun.com/product/oss) - 稳定的对象存储服务

---

<p align="center">
  <strong>⭐ 如果觉得项目不错，欢迎 Star 支持！</strong>
</p>