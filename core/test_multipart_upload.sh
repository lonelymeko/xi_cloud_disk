#!/bin/bash

# OSS 分片上传测试脚本
# 设置 10 分钟超时，足够完成大文件上传

echo "🚀 开始 OSS 分片上传测试..."
echo "⏰ 测试超时设置: 10 分钟"
echo ""

cd "$(dirname "$0")"

# 运行测试，设置 10 分钟超时
go test -v -timeout 10m -run TestInitiateMultipartUpload ./test/

# 检查测试结果
if [ $? -eq 0 ]; then
    echo ""
    echo "✅ 测试通过！"
else
    echo ""
    echo "❌ 测试失败！"
    exit 1
fi
