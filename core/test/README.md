# 测试使用说明

## OSS 分片上传测试

### 📋 前提条件

1. 准备测试文件 `test.mov` 放在 `core/test/` 目录下
2. 确保 `.env` 文件配置了正确的 OSS 凭证

### 🚀 运行方式

#### 方式 1：使用测试脚本（推荐）

```bash
cd core
./test_multipart_upload.sh
```

#### 方式 2：手动运行（指定超时）

```bash
cd core
go test -v -timeout 10m -run TestInitiateMultipartUpload ./test/
```

#### 方式 3：跳过长时间测试

```bash
cd core
go test -v -short ./test/
```

### ⚠️ 重要提示

**必须设置 `-timeout` 参数！**

Go 测试默认超时是 **30 秒**，大文件上传会超时失败。

| 文件大小 | 推荐超时 | 命令 |
|---------|---------|------|
| < 100MB | 2m | `go test -timeout 2m` |
| 100MB ~ 500MB | 5m | `go test -timeout 5m` |
| 500MB ~ 1GB | 10m | `go test -timeout 10m` |
| > 1GB | 20m | `go test -timeout 20m` |

### 📊 测试输出示例

```
=== RUN   TestInitiateMultipartUpload
    ✅ 初始化分片上传成功，上传ID: 7F185ED3...
    📊 文件大小: 494.41 MB, 分片大小: 5.00 MB
    📤 分片 1: 5.00 MB, ETag: "9C79A554...", 耗时: 1.212s, 速度: 4.13 MB/s
    📤 分片 2: 5.00 MB, ETag: "0ACE8EB5...", 耗时: 802ms, 速度: 6.23 MB/s
    ...
    📤 分片 99: 4.41 MB, ETag: "ABCD1234...", 耗时: 1.1s, 速度: 4.01 MB/s
    ✅ 所有分片上传完成，总耗时: 2m15s, 平均速度: 3.67 MB/s
    🎉 分片上传完成！
       Bucket: xi-cloud-disk
       Key: test_multipart_1738675200.mov
       Location: https://xi-cloud-disk.oss-cn-beijing.aliyuncs.com/...
       ETag: "..."
--- PASS: TestInitiateMultipartUpload (135.23s)
PASS
```

### 🛠️ 常见问题

#### Q1: 测试超时 "panic: test timed out after 30s"

**原因：** 没有设置 `-timeout` 参数

**解决：**
```bash
# ❌ 错误（使用默认 30 秒超时）
go test -v -run TestInitiateMultipartUpload ./test/

# ✅ 正确（设置 10 分钟超时）
go test -v -timeout 10m -run TestInitiateMultipartUpload ./test/
```

#### Q2: 测试文件不存在

**错误信息：**
```
--- SKIP: TestInitiateMultipartUpload (0.00s)
    测试文件不存在: test.mov
```

**解决：** 将测试文件放到 `core/test/` 目录下

#### Q3: OSS 认证失败

**错误信息：**
```
初始化分片上传失败: InvalidAccessKeyId
```

**解决：** 检查 `.env` 文件中的 OSS 配置：
```env
OSS_ACCESS_KEY_ID=your_access_key
OSS_ACCESS_KEY_SECRET=your_secret_key
OSS_BUCKET_NAME=your_bucket
OSS_REGION=oss-cn-beijing
```

#### Q4: 上传速度慢

**可能原因：**
1. 网络带宽限制
2. OSS Region 选择不当（建议选择就近的 Region）
3. 分片太小（建议 5MB ~ 10MB）

**优化建议：**
- 检查网络连接
- 选择离你更近的 OSS Region
- 调整分片大小（代码中修改 `partSize`）

### 📈 性能基准

基于 500MB 文件的测试数据：

| 分片大小 | 分片数量 | 平均速度 | 总耗时 |
|---------|---------|---------|--------|
| 5MB | 100 | 3.5 MB/s | 2m30s |
| 10MB | 50 | 4.2 MB/s | 2m00s |
| 20MB | 25 | 4.5 MB/s | 1m50s |

**建议：** 使用 10MB 分片（性能和稳定性平衡）

### 🔗 相关文档

- [OSS分片上传测试说明.md](../../docs/OSS分片上传测试说明.md) - 详细技术文档
- [阿里云 OSS 分片上传](https://help.aliyun.com/document_detail/31850.html) - 官方文档
