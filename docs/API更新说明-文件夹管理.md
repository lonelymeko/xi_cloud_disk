# file.yaml 更新说明 - 文件夹管理功能

## 📝 本次更新内容

### 新增接口

#### 1. POST /api/file/user/file/name/update
**功能：** 文件重命名

**已更新文档：** ✅

---

#### 2. POST /api/file/user/folder/create
**功能：** 创建文件夹

**已更新文档：** ✅

---

## 📋 完整接口清单

### 当前 file.yaml 包含的所有接口

| 序号 | 接口路径 | 方法 | 功能 | 状态 |
|------|---------|------|------|------|
| 1 | `/upload` | POST | 文件上传（智能压缩） | ✅ |
| 2 | `/user/repository` | POST | 保存到用户网盘 | ✅ |
| 3 | `/user/list` | POST | 获取文件列表 | ✅ |
| 4 | `/user/file/name/update` | POST | 文件重命名 | ✅ 新增 |
| 5 | `/user/folder/create` | POST | 创建文件夹 | ✅ 新增 |

---

## 🆕 新增接口详解

### 1️⃣ 文件重命名接口

#### 接口信息
```
POST /api/file/user/file/name/update
Content-Type: application/json
Authorization: Bearer <token>
```

#### 请求参数
```json
{
  "identity": "user_repo_identity_abc123",  // 用户文件记录 ID
  "name": "新文件名.pdf"                      // 新的文件名
}
```

#### 响应结果
```json
{}  // 空响应，HTTP 200 表示成功
```

#### 功能说明
- ✅ 修改 `user_repository` 表中的文件名
- ✅ 只影响当前用户的文件名
- ✅ 不修改物理文件（`repository_pool` 不变）
- ✅ 其他用户的文件名不受影响

#### 使用场景
```javascript
// 用户点击"重命名"按钮
const renameFile = async (fileId, newName) => {
  const response = await fetch('/api/file/user/file/name/update', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({
      identity: fileId,
      name: newName
    })
  });
  
  if (response.ok) {
    alert('重命名成功！');
    refreshFileList();
  }
};
```

---

### 2️⃣ 创建文件夹接口

#### 接口信息
```
POST /api/file/user/folder/create
Content-Type: application/json
Authorization: Bearer <token>
```

#### 请求参数
```json
{
  "parent_id": 0,        // 父文件夹 ID（0 = 根目录）
  "name": "我的文档"      // 文件夹名称
}
```

#### 响应结果
```json
{
  "identity": "folder_identity_abc123"  // 文件夹 ID
}
```

#### 功能说明
- ✅ 在 `user_repository` 表中创建文件夹记录
- ✅ 支持多级文件夹（通过 `parent_id`）
- ✅ 文件夹没有关联物理文件（`repository_identity` 为空）
- ✅ 其他文件可以移动到此文件夹下

#### 实现原理
```sql
-- 创建文件夹的 SQL
INSERT INTO user_repository (
  identity, 
  user_identity, 
  parent_id, 
  name, 
  ext,                      -- 空或特殊标记（如 'folder'）
  repository_identity       -- 为空（文件夹无实际文件）
) VALUES (
  'folder_abc123',
  'user_xyz789',
  0,                        -- 根目录
  '我的文档',
  '',                       -- ext 为空表示文件夹
  NULL                      -- 无关联文件
);
```

#### 使用场景

**场景 1：在根目录创建文件夹**
```javascript
const createFolder = async (folderName) => {
  const response = await fetch('/api/file/user/folder/create', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({
      parent_id: 0,           // 根目录
      name: folderName
    })
  });
  
  const data = await response.json();
  console.log('文件夹 ID:', data.identity);
};

// 创建"工作文档"文件夹
createFolder('工作文档');
```

**场景 2：创建子文件夹**
```javascript
// 在"工作文档"下创建"2024 年度报告"子文件夹
const createSubFolder = async (parentFolderId, folderName) => {
  await fetch('/api/file/user/folder/create', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({
      parent_id: parentFolderId,  // 父文件夹 ID
      name: folderName
    })
  });
};

// 假设"工作文档"的 ID 是 123
createSubFolder(123, '2024 年度报告');
```

**场景 3：完整的文件夹管理流程**
```javascript
// 1. 创建文件夹
const folder = await createFolder('项目资料');

// 2. 上传文件到文件夹
const uploadFile = await fetch('/api/file/upload', {
  method: 'POST',
  body: formData
});
const fileData = await uploadFile.json();

// 3. 将文件保存到文件夹
await fetch('/api/file/user/repository', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify({
    parent_id: folder.identity,           // 放到文件夹中
    repository_identity: fileData.identity,
    name: '需求文档.docx',
    ext: '.docx'
  })
});

// 4. 查看文件夹内容
const files = await fetch('/api/file/user/list', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify({
    id: folder.identity,    // 查询此文件夹下的文件
    page: 1,
    size: 20
  })
});
```

---

## 🗂️ 文件夹结构示例

### 用户网盘目录树

```
根目录 (parent_id = 0)
├── 工作文档/ (folder_001)
│   ├── 2024 年度报告/ (folder_002, parent_id = folder_001)
│   │   ├── Q1 报告.pdf
│   │   └── Q2 报告.pdf
│   └── 会议纪要.docx
├── 个人资料/ (folder_003)
│   ├── 照片/ (folder_004, parent_id = folder_003)
│   │   ├── 2024-01.jpg
│   │   └── 2024-02.jpg
│   └── 简历.pdf
└── 临时文件/ (folder_005)
    └── temp.txt
```

### 对应的数据库结构

```
user_repository 表：

┌────┬──────────────┬──────────┬──────────────────┬───────┬─────────────────────┐
│ id │ identity     │ parent_id│ name             │ ext   │ repository_identity │
├────┼──────────────┼──────────┼──────────────────┼───────┼─────────────────────┤
│ 1  │ folder_001   │ 0        │ 工作文档         │ NULL  │ NULL                │ ← 文件夹
│ 2  │ folder_002   │ 1        │ 2024 年度报告    │ NULL  │ NULL                │ ← 子文件夹
│ 3  │ file_001     │ 2        │ Q1 报告.pdf      │ .pdf  │ repo_abc123         │ ← 文件
│ 4  │ file_002     │ 2        │ Q2 报告.pdf      │ .pdf  │ repo_abc124         │ ← 文件
│ 5  │ file_003     │ 1        │ 会议纪要.docx    │ .docx │ repo_abc125         │ ← 文件
│ 6  │ folder_003   │ 0        │ 个人资料         │ NULL  │ NULL                │ ← 文件夹
└────┴──────────────┴──────────┴──────────────────┴───────┴─────────────────────┘
```

---

## 🔄 完整工作流程

### 用户整理文件的流程

```
1. 用户创建"工作文档"文件夹
   ↓
   POST /api/file/user/folder/create
   { parent_id: 0, name: "工作文档" }
   ↓
   返回: { identity: "folder_001" }

2. 用户创建子文件夹"2024 年度报告"
   ↓
   POST /api/file/user/folder/create
   { parent_id: "folder_001", name: "2024 年度报告" }
   ↓
   返回: { identity: "folder_002" }

3. 用户上传文件"Q1报告.pdf"
   ↓
   POST /api/file/upload
   (multipart/form-data)
   ↓
   返回: { identity: "repo_abc123", ... }

4. 用户将文件保存到"2024 年度报告"文件夹
   ↓
   POST /api/file/user/repository
   {
     parent_id: "folder_002",
     repository_identity: "repo_abc123",
     name: "Q1报告.pdf",
     ext: ".pdf"
   }

5. 用户查看"2024 年度报告"文件夹内容
   ↓
   POST /api/file/user/list
   { id: "folder_002", page: 1, size: 20 }
   ↓
   返回: { list: [...], count: 2 }

6. 用户重命名文件
   ↓
   POST /api/file/user/file/name/update
   { identity: "file_001", name: "2024Q1季度报告.pdf" }
```

---

## 🎨 前端集成示例

### React 完整示例

```jsx
import { useState, useEffect } from 'react';

function FileManager() {
  const [files, setFiles] = useState([]);
  const [currentFolderId, setCurrentFolderId] = useState(0);
  const token = localStorage.getItem('token');

  // 获取文件列表
  const fetchFiles = async (folderId) => {
    const response = await fetch('/api/file/user/list', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        id: folderId,
        page: 1,
        size: 100
      })
    });
    const data = await response.json();
    setFiles(data.list);
  };

  // 创建文件夹
  const createFolder = async () => {
    const folderName = prompt('请输入文件夹名称：');
    if (!folderName) return;

    const response = await fetch('/api/file/user/folder/create', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        parent_id: currentFolderId,
        name: folderName
      })
    });

    if (response.ok) {
      alert('文件夹创建成功！');
      fetchFiles(currentFolderId);
    }
  };

  // 重命名文件
  const renameFile = async (fileIdentity, oldName) => {
    const newName = prompt('请输入新名称：', oldName);
    if (!newName || newName === oldName) return;

    const response = await fetch('/api/file/user/file/name/update', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        identity: fileIdentity,
        name: newName
      })
    });

    if (response.ok) {
      alert('重命名成功！');
      fetchFiles(currentFolderId);
    }
  };

  // 进入文件夹
  const enterFolder = (folderId) => {
    setCurrentFolderId(folderId);
    fetchFiles(folderId);
  };

  // 返回上级
  const goBack = () => {
    // 需要记录面包屑导航或父级 ID
    setCurrentFolderId(0);
    fetchFiles(0);
  };

  useEffect(() => {
    fetchFiles(currentFolderId);
  }, []);

  return (
    <div>
      <h2>我的网盘</h2>
      
      {/* 操作按钮 */}
      <div>
        <button onClick={createFolder}>新建文件夹</button>
        <button onClick={uploadFile}>上传文件</button>
        {currentFolderId > 0 && (
          <button onClick={goBack}>返回上级</button>
        )}
      </div>

      {/* 文件列表 */}
      <table>
        <thead>
          <tr>
            <th>名称</th>
            <th>类型</th>
            <th>大小</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {files.map(file => (
            <tr key={file.id}>
              <td>
                {file.ext ? (
                  <span>{file.name}</span>
                ) : (
                  <button onClick={() => enterFolder(file.id)}>
                    📁 {file.name}
                  </button>
                )}
              </td>
              <td>{file.ext || '文件夹'}</td>
              <td>{file.size ? formatSize(file.size) : '-'}</td>
              <td>
                <button onClick={() => renameFile(file.identity, file.name)}>
                  重命名
                </button>
                <button onClick={() => deleteFile(file.identity)}>
                  删除
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB';
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
}

export default FileManager;
```

---

## ✅ 更新总结

### 本次更新
- ✅ 添加文件重命名接口文档
- ✅ 添加创建文件夹接口文档
- ✅ 完善请求参数和响应说明
- ✅ 添加使用场景示例
- ✅ 提供前端集成代码

### 文档位置
- 📄 `docs/api/file.yaml` - OpenAPI 3.0 规范文档
- 📄 `docs/API更新说明-文件夹管理.md` - 本文档

### 接口总数
当前 file.yaml 包含 **5 个接口**：
1. ✅ 文件上传
2. ✅ 保存到网盘
3. ✅ 文件列表
4. ✅ 文件重命名（新增）
5. ✅ 创建文件夹（新增）

🎉 file.yaml 更新完成！
