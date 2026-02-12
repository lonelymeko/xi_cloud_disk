# Core API 文档（面向人类）

## 通用约定

- **响应包裹**：所有接口统一返回结构 `{code,msg,data}`
- **鉴权**：需要登录的接口在 Header 中携带 `Authorization: Bearer <token>`
- **时间单位**：expires、expired_time 均为秒
- **密码字段编码**：所有涉及密码的请求字段使用 **Base64 编码** 传输（login.password、register.password、password/update.old_password、password/update.new_password、password/reset.new_password）

### 响应示例

```json
{"code":0,"msg":"ok","data":{}}
```

## 用户服务（/api/users）

| 方法 | 路径 | 认证 | 说明 | 请求体 | data 结构 |
| --- | --- | --- | --- | --- | --- |
| POST | /login | 否 | 登录（密码 Base64） | {name,password} | {token,name} |
| POST | /register | 否 | 注册（密码 Base64） | {name,email,password,code} | {token,name} |
| POST | /send-verification-code | 否 | 发送邮箱验证码 | {email} | {message} |
| POST | /password/reset | 否 | 重置密码（新密码 Base64） | {email,code,new_password} | {message} |
| POST | /detail | 否 | 用户详情 | {identity} | {name,email} |
| POST | /password/update | 是 | 修改密码（旧/新密码 Base64） | {identity,old_password,new_password} | {message} |

## 文件服务（/api/file）

### 上传

- **路径**：POST /upload
- **认证**：需要
- **请求**：`multipart/form-data`，字段 `file`
- **限制**：单文件最大 10GB，超限返回 413
- **异步**：请求成功即入队，压缩/分片上传在后台处理
- **压缩规则**：
  - 视频：`.mp4 .avi .mov .mkv .flv .wmv .webm .m4v`，ffmpeg H.264 CRF=23
  - 图片：`.jpg .jpeg .png .gif .bmp .webp`，最大 1920×1080，JPEG 质量 85
- **data**：`{message}`（示例：上传任务已入队）

### 下载链接

- **路径**：POST /url
- **认证**：需要
- **请求体**：`{repository_identity, expires}`
- **规则**：expires <= 0 使用 3600，最大 604800
- **data**：`{url, expires}`

### 文件与文件夹操作

| 方法 | 路径 | 认证 | 说明 | 请求体 | data 结构 |
| --- | --- | --- | --- | --- | --- |
| POST | /user/list | 是 | 文件列表 | {id,page,size} | {list:UserFile[],count} |
| PUT | /user/file/move | 是 | 移动 | {identity,name,parent_id} | {} |
| POST | /user/file/name/update | 是 | 重命名 | {identity,name} | {} |
| POST | /user/folder/create | 是 | 创建文件夹 | {parent_id,name} | {id,identity} |
| DELETE | /user/folder/delete | 是 | 删除文件/文件夹 | {identity} | {} |

UserFile:

```json
{"id":1,"identity":"user_repo_identity","name":"file.txt","ext":".txt","size":123,"repository_identity":"repo_id"}
```

## 分享服务（/api/share）

| 方法 | 路径 | 认证 | 说明 | 请求体/参数 | data 结构 |
| --- | --- | --- | --- | --- | --- |
| POST | /create | 是 | 创建分享 | {identity(repository_identity),expired_time} | {identity} |
| GET | /get | 否 | 分享详情 | query: identity | {repository_identity,name,ext,size} |
| POST | /url | 否 | 分享下载链接 | {share_identity,expires} | {url,expires} |
| POST | /save | 是 | 保存到网盘 | {repository_identity,parent_id,name} | {identity} |

### 分享接口说明

- `/get` 与 `/url` 为公开接口
- `/save` 仅创建关联关系，不复制物理文件
