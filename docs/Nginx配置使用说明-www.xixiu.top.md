# Nginx 配置使用说明

## 📋 配置概览

这份 Nginx 配置实现了前后端分离路由，支持以下场景：

| 路径 | 目标 | 说明 |
|------|------|------|
| `www.xixiu.top/` | 前端静态文件 | Vue 前端应用首页 |
| `www.xixiu.top/cloud_disk/` | 云盘后端（8888） | 云盘 Go API |
| `www.xixiu.top/api/` | Spring Boot 后端（6060） | 原有 Spring Boot 服务 |
| `www.xixiu.top/ws` | WebSocket（6061） | Netty WebSocket 连接 |

---

## 🚀 部署步骤

### 1. 准备前端静态文件

```bash
# 编译前端（如果还未编译）
cd /path/to/cloud_disk/web
npm run build

# 将编译后的文件复制到指定目录
mkdir -p /home/cloud_disk/frontend
cp -r dist /home/cloud_disk/frontend/

# 确保权限正确
sudo chown -R www-data:www-data /home/cloud_disk/frontend/dist
sudo chmod -R 755 /home/cloud_disk/frontend/dist
```

### 2. 配置 Nginx

```bash
# 复制配置文件到 Nginx 配置目录
sudo cp nginx-www.xixiu.top-cloud-disk.conf /etc/nginx/sites-available/www.xixiu.top

# 创建符号链接启用此配置
sudo ln -s /etc/nginx/sites-available/www.xixiu.top /etc/nginx/sites-enabled/

# 验证 Nginx 配置语法
sudo nginx -t

# 重启 Nginx
sudo systemctl restart nginx
```

### 3. 验证部署

```bash
# 测试各路由是否正常
curl -I https://www.xixiu.top/                 # 前端
curl -I https://www.xixiu.top/cloud_disk/      # 云盘
curl -I https://www.xixiu.top/api/             # Spring Boot
curl -I https://www.xixiu.top/ws               # WebSocket
```

---

## 🔧 关键配置说明

### 前端路由

```nginx
# 静态文件目录
root /home/cloud_disk/frontend/dist;

# 单页应用支持
location / {
    try_files $uri $uri/ /index.html;
}
```

**说明：**
- `try_files` 确保所有未匹配的路由都返回 `index.html`，支持单页应用客户端路由
- HTML 不缓存，API 和资源文件长期缓存（30 天）

### 云盘后端路由

```nginx
location ^~ /cloud_disk/ {
    proxy_pass http://127.0.0.1:8888/;
}
```

**关键点：**
- 使用 `^~` 前缀匹配，优先级最高
- `proxy_pass` 末尾的 `/` 会去掉 `/cloud_disk` 路径前缀
- 支持 500MB 大文件上传

### Spring Boot 后端路由

```nginx
location /api/ {
    proxy_pass http://127.0.0.1:6060;
}
```

### WebSocket 支持

```nginx
location /ws {
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 86400s;  # 24 小时超时
}
```

---

## 🔐 安全特性

配置包含以下安全头部：

| 头部 | 作用 |
|------|------|
| `X-Frame-Options: SAMEORIGIN` | 防止 Clickjacking |
| `X-Content-Type-Options: nosniff` | 防止 MIME 嗅探 |
| `X-XSS-Protection: 1; mode=block` | 启用 XSS 防护 |
| `Referrer-Policy` | 控制引用信息泄露 |
| `Permissions-Policy` | 限制浏览器功能访问 |

---

## 📊 缓存策略

```nginx
# HTML 文件：不缓存
# ├─ Cache-Control: no-cache, no-store, must-revalidate
# ├─ Expires: off

# 资源文件（JS/CSS/图片/字体）：30 天缓存
# ├─ Expires: 30d
# └─ Cache-Control: public, max-age=2592000
```

---

## 🆘 故障排查

### 前端 404

**问题：** 访问 `www.xixiu.top/` 得到 404 或空白

**解决：**
```bash
# 1. 检查文件是否存在
ls -la /home/cloud_disk/frontend/dist/

# 2. 检查权限
sudo chown -R www-data:www-data /home/cloud_disk/frontend
sudo chmod -R 755 /home/cloud_disk/frontend

# 3. 查看 Nginx 错误日志
sudo tail -f /var/log/nginx/www.xixiu.top_error.log
```

### 云盘 API 503

**问题：** `/cloud_disk/` 返回 503

**解决：**
```bash
# 检查后端是否运行
netstat -tlnp | grep 8888
# 或
lsof -i :8888

# 启动后端
cd /path/to/cloud_disk/core
go run core.go -f etc/core-api.yaml
```

### WebSocket 连接失败

**问题：** WebSocket 连接错误

**解决：**
```bash
# 检查 WebSocket 后端是否运行
netstat -tlnp | grep 6061

# 查看 Nginx 代理是否正确
sudo nginx -T | grep -A 10 "location /ws"
```

### 跨域问题

**问题：** 前端跨域请求被拒绝

**解决：** 后端需要配置 CORS 头部。Nginx 不处理 CORS，由后端返回：
```
Access-Control-Allow-Origin: https://www.xixiu.top
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
```

---

## 📈 性能优化

### 启用 Gzip 压缩

```nginx
gzip on;
gzip_min_length 1024;
gzip_types text/plain text/css application/json application/javascript;
```

### 启用 HTTP/2

```nginx
listen 443 ssl http2;  # 已启用
```

### 连接池优化

```nginx
upstream backend {
    server 127.0.0.1:8888;
    keepalive 32;  # 连接复用
}
```

---

## 🔄 更新部署

每当前端代码更新时：

```bash
# 1. 重新编译前端
cd /path/to/cloud_disk/web
npm run build

# 2. 更新静态文件
rm -rf /home/cloud_disk/frontend/dist
cp -r dist /home/cloud_disk/frontend/

# 3. 清除浏览器缓存（或等待 30 天自动过期）
# 或在 Nginx 中强制清空缓存：
# sudo systemctl reload nginx
```

---

## 📝 相关文件

- 原始配置：[nginx-lonelymeko.top-cloud-disk.conf](nginx-lonelymeko.top-cloud-disk.conf)（针对 lonelymeko.top 域名）
- 此配置：[nginx-www.xixiu.top-cloud-disk.conf](nginx-www.xixiu.top-cloud-disk.conf)（针对 www.xixiu.top 域名）

---

## 💡 提示

- 定期检查 Nginx 日志：`tail -f /var/log/nginx/www.xixiu.top_access.log`
- 监控后端服务：`systemctl status cloud_disk` 和 `systemctl status spring-boot`
- 使用 `certbot renew` 自动续期 SSL 证书
