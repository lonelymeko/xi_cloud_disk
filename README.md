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

其他依赖
```text
# xorm
go get xorm.io/xorm
# 邮箱验证
go get github.com/jordan-wright/email
# 缓存
go get github.com/go-redis/redis/v8
# uuid
go get github.com/satori/go.uuid
# aliyun oss
go get github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss

```

## 亮点
### 代码架构方面：
1. 不使用 go_zero自带的 JWT 组件（原因详见我提的 issue：[JWT Middleware 读取整个多部分/表单数据，导致文件上传性能问题](https://github.com/zeromicro/go-zero/issues/5401)）。引入JWT 认证，使用JwtPayLoad结构体自定义签名，在统一的中间件检查权限并将用户信息保存在ctx中，直接在 logic 方法查询当前上下文的信息，避免在每个handler中检查请求头来获取用户信息，避免在数据库中查询用户信息。
2. 修改了go-zero的 API 代码生成模板文件，使其添加了自定义Response 统一响应处理，无需在 api 文件里重复封装，修改可以参考我的博客。

### 业务方面：
1. 使用使用  Mysql 的 CTE 递归查询来递归删除文件树，避免在业务层手动递归遍历查询数据库


## TODO
1. 改造文件上传
1.1 支持分片上传
1.2 将上传进度实时推送给前端（通过mq 推送 websocket消息）
1.3 异步处理上传文件（通过 mq 推送执行压缩和上传至 OSS方法）

2. 将 用户文件列表总数缓存到 Redis

3. 添加下载文件夹里所有文件并打包的功能

4. (次要) 用 Redis 维护文件的下载次数