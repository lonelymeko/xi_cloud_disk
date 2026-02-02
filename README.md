# 玺云盘

轻量级云盘系统
```text
# 创建 API 服务
goctl api new core
cd core
# 运行
go run core.go -f etc/core-api.yaml
# 生成代码
goctl api go -api core.api -dir . -style go_zero

```