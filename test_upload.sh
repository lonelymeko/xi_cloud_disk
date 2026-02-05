#!/bin/bash

# 测试文件上传功能
# 使用方法: ./test_upload.sh <token>

if [ -z "$1" ]; then
    echo "用法: $0 <JWT_TOKEN>"
    echo "请先登录获取 token"
    exit 1
fi

TOKEN=$1
API_URL="http://localhost:8888"

# 创建测试文件
TEST_FILE="/tmp/test_upload_file.txt"
echo "This is a test file for upload $(date)" > $TEST_FILE
echo "文件大小: $(wc -c < $TEST_FILE) bytes"

echo "=== 开始测试文件上传 ==="
echo "上传文件: $TEST_FILE"
echo ""

# 上传文件
RESPONSE=$(curl -s -X POST \
  -H "Authorization: $TOKEN" \
  -F "file=@$TEST_FILE" \
  -F "parent_id=0" \
  "$API_URL/file/upload")

echo "响应结果:"
echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"

# 清理测试文件
rm -f $TEST_FILE

echo ""
echo "=== 测试完成 ==="
