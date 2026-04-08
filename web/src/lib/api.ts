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

export interface UploadTaskStatusItem {
  hash: string
  task_key: string
  name: string
  ext: string
  size: number
  state: number
  uploaded_parts: number[]
  updated_at: string
}

export interface QueryUploadTaskStatusResp {
  list: UploadTaskStatusItem[]
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

export interface AgentFileReference {
  file_identity?: string
  name?: string
  url?: string
  mime_type?: string
}

export interface AgentPendingInterrupt {
  interrupt_id: string
  tool_name: string
  arguments_json: string
}

export interface AgentChatEvent {
  type: string
  role?: string
  content?: string
  tool_name?: string
  arguments_json?: string
}

export interface AgentSession {
  id: string
  title: string
  pending_interrupt_id?: string
  created_at?: string
  updated_at?: string
}

export interface AgentConversationMessage {
  role: string
  content: string
  created_at?: string
}

export interface AgentChatResponse {
  session?: AgentSession
  reply?: string
  events?: AgentChatEvent[]
  pending_interrupt?: AgentPendingInterrupt
  referenced_files?: AgentFileReference[]
}

export interface AgentSessionListResponse {
  list: AgentSession[]
}

export interface AgentSessionDetailResponse {
  session?: AgentSession
  messages?: AgentConversationMessage[]
}

export interface AgentStreamEvent {
  type: string
  role?: string
  content?: string
  tool_name?: string
  arguments_json?: string
  session?: AgentSession
  pending_interrupt?: AgentPendingInterrupt
  referenced_files?: AgentFileReference[]
  reply?: string
  error?: string
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

export async function queryUploadTaskStatus(token: string, hash = ''): Promise<QueryUploadTaskStatusResp> {
  const res = await fetch(`${API_BASE}/api/file/upload/status`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...withAuth(token),
    },
    body: JSON.stringify({ hash }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<QueryUploadTaskStatusResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<QueryUploadTaskStatusResp>(res)
  if (json.code !== 0) throw new Error(json.msg || '获取上传任务状态失败')
  return json.data
}

export async function createAgentSession(token: string): Promise<AgentChatResponse> {
  const res = await fetch(`${API_BASE}/api/file/session/create`, {
    method: 'POST',
    headers: {
      ...withAuth(token),
    },
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<AgentChatResponse> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<AgentChatResponse>(res)
  if (json.code !== 0) throw new Error(json.msg || '创建会话失败')
  return json.data
}

export async function listAgentSessions(token: string): Promise<AgentSessionListResponse> {
  const res = await fetch(`${API_BASE}/api/file/session/list`, {
    method: 'GET',
    headers: {
      ...withAuth(token),
    },
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<AgentSessionListResponse> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<AgentSessionListResponse>(res)
  if (json.code !== 0) throw new Error(json.msg || '获取会话列表失败')
  return json.data
}

export async function getAgentSessionDetail(sessionId: string, token: string): Promise<AgentSessionDetailResponse> {
  const qs = new URLSearchParams({ session_id: sessionId })
  const res = await fetch(`${API_BASE}/api/file/session/detail?${qs.toString()}`, {
    method: 'GET',
    headers: {
      ...withAuth(token),
    },
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<AgentSessionDetailResponse> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<AgentSessionDetailResponse>(res)
  if (json.code !== 0) throw new Error(json.msg || '获取会话详情失败')
  return json.data
}

async function readSSEStream(res: Response, onEvent: (event: AgentStreamEvent) => void): Promise<void> {
  if (!res.body) {
    throw new Error('响应不支持流式读取')
  }
  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })

    let boundary = buffer.indexOf('\n\n')
    while (boundary >= 0) {
      const chunk = buffer.slice(0, boundary)
      buffer = buffer.slice(boundary + 2)
      const lines = chunk.split('\n')
      const dataLines = lines
        .filter((line) => line.startsWith('data:'))
        .map((line) => line.slice(5).trim())
      if (dataLines.length > 0) {
        const raw = dataLines.join('\n')
        if (raw) {
          onEvent(JSON.parse(raw) as AgentStreamEvent)
        }
      }
      boundary = buffer.indexOf('\n\n')
    }
  }
}

export async function agentChatStream(
  sessionId: string,
  message: string,
  attachments: AgentFileReference[],
  token: string,
  onEvent: (event: AgentStreamEvent) => void,
): Promise<void> {
  const res = await fetch(`${API_BASE}/api/file/chat/stream`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...withAuth(token),
    },
    body: JSON.stringify({
      session_id: sessionId,
      message,
      attachments,
    }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<AgentChatResponse> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  await readSSEStream(res, onEvent)
}

export async function agentResumeStream(
  sessionId: string,
  interruptId: string,
  approved: boolean,
  reason: string,
  token: string,
  onEvent: (event: AgentStreamEvent) => void,
): Promise<void> {
  const res = await fetch(`${API_BASE}/api/file/resume/stream`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...withAuth(token),
    },
    body: JSON.stringify({
      session_id: sessionId,
      interrupt_id: interruptId,
      approved,
      reason,
    }),
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<AgentChatResponse> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  await readSSEStream(res, onEvent)
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
    if (res.status === 413) throw new Error('上传被拒绝(413)：文件大小超过网关或服务端限制，请检查 Nginx 的 client_max_body_size 与后端 MaxBytes 配置')
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

export async function getShare(identity: string,token: string): Promise<ShareDetailResp> {
  const url = `${API_BASE}/api/share/get?identity=${encodeURIComponent(identity)}`
  const res = await fetch(url, { method: 'GET' ,
  headers: withAuth(token)
  })
  if (!res.ok) {
    const json = (await res.json().catch(() => null)) as ApiResp<ShareDetailResp> | null
    if (json?.msg) throw new Error(json.msg)
    throw new Error(`HTTP ${res.status}`)
  }
  const json = await readJson<ShareDetailResp>(res)
  if (json.code !== 0) throw new Error(json.msg || '获取分享失败')
  return json.data
}

export async function getShareUrl(shareIdentity: string, expires: number, token: string): Promise<ShareURLResp> {
  const qs = new URLSearchParams({
    share_identity: shareIdentity,
    expires: String(expires),
  })
  const res = await fetch(`${API_BASE}/api/share/download?${qs.toString()}`, {
    method: 'GET',
    headers: {
      ...withAuth(token),
    },
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
