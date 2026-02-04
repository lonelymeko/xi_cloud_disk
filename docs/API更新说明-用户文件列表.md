# API 更新说明 - 用户文件列表接口

## 📝 更新内容

### 新增接口：`POST /api/file/user/list`

已在 `file.yaml` 中添加完整的 API 文档。

---

## 🎯 接口作用

### 功能说明

获取当前登录用户的**个人网盘文件列表**，支持：
- ✅ 文件夹筛选（查看指定文件夹下的文件）
- ✅ 分页查询（大量文件时优化性能）
- ✅ 完整文件信息（名称、大小、扩展名等）

---

## 📊 数据来源

### 查询逻辑

```sql
-- 查询用户的文件列表
SELECT 
    ur.id,
    ur.identity,
    ur.name,
    ur.ext,
    rp.size,
    ur.repository_identity
FROM user_repository ur
LEFT JOIN repository_pool rp 
    ON ur.repository_identity = rp.identity
WHERE ur.user_identity = ? 
    AND ur.parent_id = ?
    AND ur.deleted_at IS NULL
ORDER BY ur.created_at DESC
LIMIT ? OFFSET ?
```

**关键点：**
1. 从 `user_repository` 表查询（用户的文件关联）
2. 关联 `repository_pool` 表获取文件大小
3. 按用户和文件夹 ID 筛选
4. 支持分页

---

## 🔄 完整工作流程

### 用户上传并查看文件

```
┌─────────────────────────────────────────────┐
│ 步骤 1：上传文件                              │
└─────────────────────────────────────────────┘
POST /api/file/upload
  ↓
返回：{
  "identity": "file_abc123",
  "name": "document.pdf",
  "ext": ".pdf"
}

┌─────────────────────────────────────────────┐
│ 步骤 2：保存到用户网盘                         │
└─────────────────────────────────────────────┘
POST /api/file/user/repository
{
  "repository_identity": "file_abc123",
  "name": "我的文档.pdf",
  "ext": ".pdf",
  "parent_id": 0
}
  ↓
返回：{
  "identity": "ur_xyz789"
}

┌─────────────────────────────────────────────┐
│ 步骤 3：查看文件列表（新接口）                  │
└─────────────────────────────────────────────┘
POST /api/file/user/list
{
  "id": 0,      // 查看根目录
  "page": 1,
  "size": 20
}
  ↓
返回：{
  "list": [
    {
      "id": 1,
      "identity": "ur_xyz789",
      "name": "我的文档.pdf",
      "ext": ".pdf",
      "size": 1048576,
      "repository_identity": "file_abc123"
    }
  ],
  "count": 1
}
```

---

## 📋 请求参数详解

### UserFileListRequest

```json
{
  "id": 0,       // 文件夹 ID（parent_id）
  "page": 1,     // 页码（从 1 开始）
  "size": 20     // 每页数量
}
```

| 参数 | 类型 | 必填 | 说明 | 示例 |
|------|------|------|------|------|
| `id` | int64 | 否 | 文件夹 ID，0 或不传表示根目录 | `0` |
| `page` | int64 | 否 | 页码，默认 1 | `1` |
| `size` | int64 | 否 | 每页数量，默认 20 | `20` |

---

## 📤 响应结果详解

### UserFileListResponse

```json
{
  "list": [
    {
      "id": 1,
      "identity": "ur_xyz789",
      "name": "我的文档.pdf",
      "ext": ".pdf",
      "size": 1048576,
      "repository_identity": "file_abc123"
    }
  ],
  "count": 100
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `list` | array | 文件列表 |
| `count` | int64 | 文件总数（用于计算总页数） |

### UserFile 对象

| 字段 | 类型 | 说明 | 来源表 |
|------|------|------|--------|
| `id` | int64 | 用户文件记录 ID | `user_repository.id` |
| `identity` | string | 用户文件记录标识 | `user_repository.identity` |
| `name` | string | 文件名称 | `user_repository.name` |
| `ext` | string | 文件扩展名 | `user_repository.ext` |
| `size` | int64 | 文件大小（字节） | `repository_pool.size` |
| `repository_identity` | string | 文件仓库标识 | `user_repository.repository_identity` |

---

## 🎨 前端使用示例

### React 示例

```jsx
import { useState, useEffect } from 'react';

function FileList() {
  const [files, setFiles] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [folderId, setFolderId] = useState(0);

  useEffect(() => {
    fetchFiles();
  }, [page, folderId]);

  const fetchFiles = async () => {
    const response = await fetch('http://localhost:8888/api/file/user/list', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({
        id: folderId,
        page: page,
        size: 20
      })
    });

    const data = await response.json();
    setFiles(data.list);
    setTotal(data.count);
  };

  const formatSize = (bytes) => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB';
    if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
    return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
  };

  return (
    <div>
      <h2>我的文件 ({total} 个文件)</h2>
      
      <table>
        <thead>
          <tr>
            <th>文件名</th>
            <th>大小</th>
            <th>类型</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {files.map(file => (
            <tr key={file.id}>
              <td>{file.name}</td>
              <td>{formatSize(file.size)}</td>
              <td>{file.ext}</td>
              <td>
                <button>下载</button>
                <button>删除</button>
                <button>分享</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {/* 分页 */}
      <div>
        <button 
          disabled={page === 1}
          onClick={() => setPage(page - 1)}
        >
          上一页
        </button>
        <span>第 {page} 页 / 共 {Math.ceil(total / 20)} 页</span>
        <button 
          disabled={page * 20 >= total}
          onClick={() => setPage(page + 1)}
        >
          下一页
        </button>
      </div>
    </div>
  );
}
```

---

## 🔍 使用场景

### 场景 1：显示网盘首页

```json
POST /api/file/user/list
{
  "id": 0,      // 根目录
  "page": 1,
  "size": 20
}
```

**用途：** 用户登录后，显示根目录下的所有文件

---

### 场景 2：进入文件夹

```json
POST /api/file/user/list
{
  "id": 123,    // 文件夹 ID
  "page": 1,
  "size": 20
}
```

**用途：** 用户点击文件夹，查看该文件夹下的文件

---

### 场景 3：分页加载

```json
// 第 1 页
POST /api/file/user/list
{ "id": 0, "page": 1, "size": 20 }

// 第 2 页
POST /api/file/user/list
{ "id": 0, "page": 2, "size": 20 }

// 第 3 页
POST /api/file/user/list
{ "id": 0, "page": 3, "size": 20 }
```

**用途：** 用户有大量文件时，分页加载提升性能

---

## 🆚 两个表的核心区别

### 快速对比

| 维度 | repository_pool | user_repository |
|------|----------------|-----------------|
| **中文名** | 文件存储池 | 用户文件仓库 |
| **作用** | 存储物理文件 | 用户与文件的关联 |
| **数据特点** | 全局唯一 | 用户隔离 |
| **去重依据** | MD5 hash | 无 |
| **对应关系** | 1 个文件 = 1 条记录 | 1 个文件可以有 N 条记录 |
| **存储位置** | OSS | 数据库 |
| **相关接口** | `/upload` | `/user/repository`、`/user/list` |

---

### 形象比喻

#### repository_pool = 图书馆仓库
- 📚 存放所有的实体书（物理文件）
- 🔍 每本书有唯一编号（identity）
- ✅ 同一本书只存一本（hash 去重）
- 📦 对应 OSS 存储

#### user_repository = 用户借书记录
- 👤 记录用户借了哪些书
- 📝 同一本书可以被多个用户借阅
- 🏷️ 用户可以给书起别名（重命名）
- 📂 用户可以整理自己的书架（文件夹）

---

### 数据关系示例

```
repository_pool（图书馆仓库）
┌────────────────────────────────┐
│ identity: book_001             │
│ hash: abc123                   │
│ name: "Go 语言编程.pdf"         │ ← 实体书（只有 1 本）
│ size: 5MB                      │
└────────────────────────────────┘
            ↑
            │ (多个用户可以借阅同一本书)
            │
    ┌───────┴──────┐
    │              │
┌─────────┐  ┌─────────┐
│ 用户 A   │  │ 用户 B   │
│ 借书记录 │  │ 借书记录 │
├─────────┤  ├─────────┤
│ book_001│  │ book_001│ ← 关联同一本书
│ "Go教程" │  │ "编程书" │ ← 自定义书名
│ 我的书架 │  │ 技术书架 │ ← 自己的分类
└─────────┘  └─────────┘
```

---

## 📈 性能优化建议

### 数据库索引

```sql
-- user_repository 表必须添加索引
CREATE INDEX idx_user_parent 
ON user_repository(user_identity, parent_id);

-- 查询用户文件列表时使用此索引
-- 大幅提升查询性能（从秒级降到毫秒级）
```

### 分页优化

```go
// 推荐：使用游标分页（性能更好）
WHERE id > last_id 
ORDER BY id 
LIMIT 20

// 不推荐：OFFSET 分页（数据量大时很慢）
LIMIT 20 OFFSET 1000
```

---

## ✅ 总结

### 三个核心接口

| 接口 | 作用 | 操作表 |
|------|------|--------|
| `/upload` | 上传文件到存储池 | `repository_pool` |
| `/user/repository` | 保存到用户网盘 | `user_repository` |
| `/user/list` | 查看用户文件列表 | `user_repository` + `repository_pool` |

### 典型使用流程

```
1. 用户上传文件
   ↓ /upload
2. 文件保存到 repository_pool（去重）
   ↓ 返回 identity
3. 关联到用户网盘
   ↓ /user/repository
4. 保存到 user_repository（用户关联）
   ↓ 返回成功
5. 查看文件列表
   ↓ /user/list
6. 显示用户的所有文件
```

### 关键优势

✅ **文件去重** - 多人上传同一文件，只存一份  
✅ **秒传功能** - 已存在文件瞬间完成  
✅ **用户隔离** - 每个用户有独立的文件列表  
✅ **灵活管理** - 支持重命名、移动、文件夹  
✅ **文件共享** - 天然支持多人共享同一文件  

🎉 完整的云盘文件管理系统！
