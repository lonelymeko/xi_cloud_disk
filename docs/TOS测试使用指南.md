# 🧪 TOS 存储测试使用指南

## 快速开始测试 TOS

### 设置环境变量

```bash
export STORAGE_TYPE=tos
export VOLCENGINE_ACCESS_KEY_ID=your_access_key
export VOLCENGINE_SECRET_ACCESS_KEY=your_secret_key
export VOLCENGINE_REGION=cn-beijing
export VOLCENGINE_ENDPOINT=tos-cn-beijing.volces.com
export VOLCENGINE_BUCKET_NAME=your-bucket-name
```

### 运行基础功能测试

```bash
cd core

# 测试上传/下载/删除
go test -v ./test -run TestTOSUpload

# 测试预签名 URL
go test -v ./test -run TestTOSPresignedURL

# 测试存储管理器
go test -v ./test -run TestTOSStorageManager

# 测试存储接口完整性
go test -v ./test -run TestTOSStorageInterface
```

### 运行分片上传测试

```bash
# 需要一个测试视频文件 test.mov
# 完整分片上传流程
go test -v ./test -run TestTOSMultipartUpload

# 测试分片上传取消
go test -v ./test -run TestTOSMultipartCancel
```

### 一次运行所有 TOS 测试

```bash
go test -v ./test -run TestTOS
```

## 测试文件说明

### `tos_test.go` - 基础功能测试

| 测试函数 | 功能 | 耗时 |
|---------|------|------|
| `TestTOSUpload` | 上传/下载/删除文件 | ~10s |
| `TestTOSPresignedURL` | 生成预签名 URL | ~10s |
| `TestTOSStorageManager` | 全局存储管理器 | ~5s |

**特点：**
- 轻量级测试
- 无需额外文件
- 快速验证基础功能
- 适合 CI/CD 集成

### `tos_multipart_upload_test.go` - 分片上传测试

| 测试函数 | 功能 | 耗时 | 要求 |
|---------|------|------|------|
| `TestTOSMultipartUpload` | 完整分片上传流程 | 取决于文件大小 | test.mov 文件 |
| `TestTOSMultipartCancel` | 分片上传中止 | ~5s | 无 |
| `TestTOSStorageInterface` | 接口一致性验证 | ~10s | 无 |

**特点：**
- 完整的真实场景测试
- 性能基准测试
- 支持大文件测试
- 需要真实的存储服务

## 测试结果解读

### 成功输出示例

```
=== RUN   TestTOSUpload
    tos_test.go:22: ✅ Upload successful: test/1711159800-test.txt
    tos_test.go:37: ✅ Download successful: test/1711159800-test.txt
    tos_test.go:44: ✅ Delete successful: test/1711159800-test.txt
--- PASS: TestTOSUpload (2.34s)

=== RUN   TestTOSMultipartUpload
    tos_multipart_upload_test.go:52: 📊 File size: 245.67 MB, Part size: 5.00 MB
    tos_multipart_upload_test.go:58: 📋 Initiating multipart upload: test_multipart_1711159810.mp4
    tos_multipart_upload_test.go:64: ✅ Initiated, upload ID: abc123xyz
    tos_multipart_upload_test.go:85: 📤 Part 1: 5.00 MB, ETag: "abc123", Time: 3.45s, Speed: 1.45 MB/s
    ...
    tos_multipart_upload_test.go:115: ✅ All parts uploaded, Total time: 2m30s, Avg speed: 1.64 MB/s
    tos_multipart_upload_test.go:127: 🎉 Multipart upload completed!
--- PASS: TestTOSMultipartUpload (150.23s)
```

### 失败诊断

**问题：** `TOS env not set`
- **原因：** 环境变量未设置
- **解决：** 检查是否正确导出了所有环境变量

**问题：** `connection refused`
- **原因：** 无法连接到 TOS 服务
- **解决：** 检查 VOLCENGINE_ENDPOINT 是否正确

**问题：** `access denied`
- **原因：** Access Key 或 Secret Key 无效
- **解决：** 验证火山云控制台的凭证

**问题：** `file not found: test.mov`
- **原因：** 分片上传测试需要测试文件
- **解决：** 放置一个视频文件到 `core/test/test.mov`

## 跳过或运行特定测试

```bash
# 跳过分片上传测试（-short 标志）
go test -short -v ./test -run TestTOS

# 只运行某个具体测试
go test -v ./test -run TestTOSUpload

# 并行运行测试
go test -v ./test -run TestTOS -parallel 4

# 显示测试覆盖率
go test -v ./test -run TestTOS -cover
```

## 性能参考

| 操作 | 平均耗时 | 说明 |
|------|--------|------|
| PutObject (小文件) | ~1-2s | <1MB |
| GetObject (小文件) | ~0.5-1s | <1MB |
| 单个分片上传 | ~3-5s | 5MB 分片 |
| 分片上传完成 | ~1s | 元数据操作 |
| 生成预签名 URL | ~0.1s | 本地计算 |
| DeleteObject | ~0.5s | 删除操作 |

实际时间取决于网络条件和 TOS 服务负载。

## 常见测试命令组合

### 快速验证连接

```bash
go test -v ./test -run TestTOSStorageManager -timeout 10s
```

### 验证所有基础功能

```bash
go test -v ./test -run 'TestTOS(Upload|Presigned|Manager|Interface)' -timeout 30s
```

### 性能测试

```bash
go test -v ./test -run TestTOSMultipartUpload -timeout 5m -benchmem
```

### 持续测试

```bash
while true; do
  go test -v ./test -run TestTOSStorageInterface
  sleep 60
done
```

## 注意事项

1. 💰 **成本提示**
   - 每次测试会在 TOS 上传/下载真实数据
   - 产生相应的存储和流量费用
   - 请定期清理测试数据

2. ⏱️ **超时设置**
   - 分片上传测试默认 5 分钟超时
   - 大文件可能需要更长时间
   - 使用 `-timeout 30m` 调整

3. 🔒 **凭证安全**
   - 勿在代码或日志中硬编码凭证
   - 使用环境变量传递敏感信息
   - 定期轮换 Access Key

4. 📝 **日志级别**
   - 测试会输出详细日志
   - 根据需要调整日志级别
   - 追踪网络问题时很有用

## 与 OSS 测试对比

所有 TOS 测试都镜像 OSS 的测试结构：

```
tos_test.go                    ↔  oss_test.go
├─ TestTOSUpload              ↔  TestUploadToOSS
├─ TestTOSPresignedURL        ↔  TestOSSHost
├─ TestTOSStorageManager      ↔  (新增)
└─ TestTOSStorageInterface    ↔  TestOSSUploadDownloadDelete_Integration

tos_multipart_upload_test.go  ↔  oss_initiate_multipart_upload_test.go
├─ TestTOSMultipartUpload     ↔  TestInitiateMultipartUpload
└─ TestTOSMultipartCancel     ↔  (新增)
```

两个存储的测试用例完全兼容，可以验证接口一致性。
