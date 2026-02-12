export const API_BASE = import.meta.env.VITE_API_BASE ?? ''

export interface ApiResp<T> {
  code: number
  msg: string
  data: T
}

export interface LoginResp {
  token: string
  name: string
}

export interface RegisterResp {
  token: string
  name: string
}

export interface SendVerificationCodeResp {
  message: string
}

export interface ChangePasswordResp {
  message: string
}

export interface ResetPasswordResp {
  message: string
}

export interface UserDetailResp {
  name: string
  email: string
}

export interface UserFile {
  id: number
  identity: string
  name: string
  ext: string
  size: number
  repository_identity: string
  updated_at: string
}

export interface UserFileListResp {
  list: UserFile[]
  count: number
}

export interface UploadFileResp {
  message: string
}

export interface CreateFolderResp {
  id: number
  identity: string
}

export interface DownloadURLResp {
  url: string
  expires: number
}

export interface CreateShareResp {
  identity: string
}

export interface ShareDetailResp {
  repository_identity: string
  name: string
  ext: string
  size: number
}

export interface ShareURLResp {
  url: string
  expires: number
}

export interface SaveShareResp {
  identity: string
}

function base64Encode(input: string): string {
  const bytes = new TextEncoder().encode(input)
  let bin = ''
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i])
  return btoa(bin)
}

async function readJson<T>(res: Response): Promise<ApiResp<T>> {
  const json = (await res.json().catch(() => null)) as ApiResp<T> | null
  if (!json) throw new Error(`HTTP ${res.status}`)
  return json
}

function withAuth(token: string) {
  return { Authorization: `Bearer ${token}` }
}

export async function login(name: string, password: string): Promise<LoginResp> {
  const res = await fetch(`${API_BASE}/api/users/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, password: base64Encode(password) }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<LoginResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = (await res.json()) as ApiResp<LoginResp>
  if (json.code !== 0) throw new Error(json.msg || '登录失败')
  return json.data
}

export async function register(name: string, email: string, password: string, code: string): Promise<RegisterResp> {
  const res = await fetch(`${API_BASE}/api/users/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, email, password: base64Encode(password), code }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<RegisterResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = (await res.json()) as ApiResp<RegisterResp>
  if (json.code !== 0) throw new Error(json.msg || '注册失败')
  return json.data
}

export async function sendVerificationCode(email: string): Promise<SendVerificationCodeResp> {
  const res = await fetch(`${API_BASE}/api/users/send-verification-code`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<SendVerificationCodeResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = (await res.json()) as ApiResp<SendVerificationCodeResp>
  if (json.code !== 0) throw new Error(json.msg || '发送验证码失败')
  return json.data
}

export async function changePassword(identity: string, oldPassword: string, newPassword: string, token: string): Promise<ChangePasswordResp> {
  const res = await fetch(`${API_BASE}/api/users/password/update`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ identity, old_password: base64Encode(oldPassword), new_password: base64Encode(newPassword) }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<ChangePasswordResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = (await res.json()) as ApiResp<ChangePasswordResp>
  if (json.code !== 0) throw new Error(json.msg || '修改密码失败')
  return json.data
}

export async function resetPassword(email: string, code: string, newPassword: string): Promise<ResetPasswordResp> {
  const res = await fetch(`${API_BASE}/api/users/password/reset`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, code, new_password: base64Encode(newPassword) }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<ResetPasswordResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = (await res.json()) as ApiResp<ResetPasswordResp>
  if (json.code !== 0) throw new Error(json.msg || '重置密码失败')
  return json.data
}

export async function authProbe(token: string): Promise<boolean> {
  const res = await fetch(`${API_BASE}/api/file/user/list`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...withAuth(token),
    },
    body: JSON.stringify({ id: 0, page: 1, size: 1 }),
  })
  if (!res.ok) return false
  const json = await res.json().catch(() => null)
  return json && typeof json.code === 'number' && json.code === 0
}

export async function getUserDetail(identity: string): Promise<UserDetailResp> {
  const res = await fetch(`${API_BASE}/api/users/detail`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ identity }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<UserDetailResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = (await res.json()) as ApiResp<UserDetailResp>
  if (json.code !== 0) throw new Error(json.msg || '获取用户信息失败')
  return json.data
}

export async function getUserFileList(parentId: number, page: number, size: number, token: string): Promise<UserFileListResp> {
  const res = await fetch(`${API_BASE}/api/file/user/list`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...withAuth(token),
    },
    body: JSON.stringify({ id: parentId, page, size }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<UserFileListResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<UserFileListResp>(res)
  if (json.code !== 0) throw new Error(json.msg || '获取文件列表失败')
  return json.data
}

export async function uploadFile(file: File, parentId: number, token: string): Promise<UploadFileResp> {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('parent_id', String(parentId))
  const res = await fetch(`${API_BASE}/api/file/upload`, {
    method: 'POST',
    headers: {
      ...withAuth(token),
    },
    body: formData,
  })
  if (!res.ok) {
    if (res.status === 413) throw new Error('文件过大，超过10GB限制')
    const json = (await res.json().catch(() => null)) as ApiResp<UploadFileResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<UploadFileResp>(res)
  if (json.code !== 0) throw new Error(json.msg || '上传失败')
  return json.data
}

export async function createFolder(parentId: number, name: string, token: string): Promise<CreateFolderResp> {
  const res = await fetch(`${API_BASE}/api/file/user/folder/create`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...withAuth(token),
    },
    body: JSON.stringify({ parent_id: parentId, name }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<CreateFolderResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<CreateFolderResp>(res)
  if (json.code !== 0) throw new Error(json.msg || '创建文件夹失败')
  return json.data
}

export async function renameUserFile(identity: string, name: string, token: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/file/user/file/name/update`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...withAuth(token),
    },
    body: JSON.stringify({ identity, name }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<Record<string, never>> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<Record<string, never>>(res)
  if (json.code !== 0) throw new Error(json.msg || '重命名失败')
}

export async function moveUserFile(identity: string, name: string, parentId: number, token: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/file/user/file/move`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      ...withAuth(token),
    },
    body: JSON.stringify({ identity, name, parent_id: parentId }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<Record<string, never>> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<Record<string, never>>(res)
  if (json.code !== 0) throw new Error(json.msg || '移动失败')
}

export async function deleteUserItem(identity: string, token: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/file/user/folder/delete`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
      ...withAuth(token),
    },
    body: JSON.stringify({ identity }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<Record<string, never>> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<Record<string, never>>(res)
  if (json.code !== 0) throw new Error(json.msg || '删除失败')
}

export async function getDownloadUrl(repositoryIdentity: string, expires: number, token: string): Promise<DownloadURLResp> {
  const res = await fetch(`${API_BASE}/api/file/url`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...withAuth(token),
    },
    body: JSON.stringify({ repository_identity: repositoryIdentity, expires }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<DownloadURLResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<DownloadURLResp>(res)
  if (json.code !== 0) throw new Error(json.msg || '获取下载链接失败')
  return json.data
}

export async function createShare(repositoryIdentity: string, expiredTime: number, token: string): Promise<CreateShareResp> {
  const res = await fetch(`${API_BASE}/api/share/create`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...withAuth(token),
    },
    body: JSON.stringify({ identity: repositoryIdentity, expired_time: expiredTime }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<CreateShareResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<CreateShareResp>(res)
  if (json.code !== 0) throw new Error(json.msg || '创建分享失败')
  return json.data
}

export async function getShare(identity: string): Promise<ShareDetailResp> {
  const url = new URL(`${API_BASE}/api/share/get`)
  url.searchParams.set('identity', identity)
  const res = await fetch(url.toString(), { method: 'GET' })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<ShareDetailResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<ShareDetailResp>(res)
  if (json.code !== 0) throw new Error(json.msg || '获取分享失败')
  return json.data
}

export async function getShareUrl(shareIdentity: string, expires: number): Promise<ShareURLResp> {
  const res = await fetch(`${API_BASE}/api/share/url`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ share_identity: shareIdentity, expires }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<ShareURLResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<ShareURLResp>(res)
  if (json.code !== 0) throw new Error(json.msg || '获取分享下载链接失败')
  return json.data
}

export async function saveShare(repositoryIdentity: string, parentId: number, name: string, token: string): Promise<SaveShareResp> {
  const res = await fetch(`${API_BASE}/api/share/save`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...withAuth(token),
    },
    body: JSON.stringify({ repository_identity: repositoryIdentity, parent_id: parentId, name }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<SaveShareResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<SaveShareResp>(res)
  if (json.code !== 0) throw new Error(json.msg || '保存分享失败')
  return json.data
}
