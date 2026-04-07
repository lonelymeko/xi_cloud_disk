<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { agentChatStream, agentResumeStream, createAgentSession, getAgentSessionDetail, getUserFileList, listAgentSessions, type AgentFileReference, type AgentPendingInterrupt, type AgentSession, type AgentStreamEvent, type UserFile } from '../lib/api'
import { getToken } from '../lib/auth'

type ChatMessage = {
  id: string
  role: 'user' | 'assistant' | 'tool_call' | 'tool_result' | 'approval'
  content: string
  toolName?: string
  pending?: AgentPendingInterrupt
}

type PickerCrumb = { id: number; name: string }

const sessions = ref<AgentSession[]>([])
const activeSessionId = ref('')
const messages = ref<ChatMessage[]>([])
const input = ref('')
const loading = ref(false)
const error = ref('')
const pickerOpen = ref(false)
const pickerLoading = ref(false)
const pickerError = ref('')
const pickerFolderId = ref(0)
const pickerPath = ref<PickerCrumb[]>([{ id: 0, name: '我的文件' }])
const pickerList = ref<UserFile[]>([])
const referencedFiles = ref<AgentFileReference[]>([])
const rejectReason = ref('')
const resolvingInterrupt = ref(false)
const messagePaneRef = ref<HTMLElement | null>(null)

function nextId(prefix: string) {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

function isFolder(item: UserFile) {
  return !item.repository_identity
}

const activeSession = computed(() => sessions.value.find((item) => item.id === activeSessionId.value) || null)

async function loadSessions() {
  const token = getToken()
  if (!token) return
  const data = await listAgentSessions(token)
  sessions.value = data.list || []
  if (!activeSessionId.value && sessions.value.length > 0) {
    activeSessionId.value = sessions.value[0].id
    await loadSessionDetail(activeSessionId.value)
  }
}

async function ensureSession() {
  if (activeSessionId.value) return activeSessionId.value
  const token = getToken()
  if (!token) throw new Error('登录已失效，请重新登录')
  const data = await createAgentSession(token)
  if (!data.session?.id) throw new Error('创建会话失败')
  activeSessionId.value = data.session.id
  sessions.value = [data.session, ...sessions.value.filter((item) => item.id !== data.session?.id)]
  messages.value = []
  return activeSessionId.value
}

function resetPicker() {
  pickerFolderId.value = 0
  pickerPath.value = [{ id: 0, name: '我的文件' }]
  pickerList.value = []
  pickerError.value = ''
}

async function loadPickerList() {
  const token = getToken()
  if (!token) {
    pickerError.value = '登录已失效，请重新登录'
    return
  }
  pickerLoading.value = true
  pickerError.value = ''
  try {
    const data = await getUserFileList(pickerFolderId.value, 1, 200, token)
    pickerList.value = data.list || []
  } catch (e: any) {
    pickerError.value = e?.message || '加载文件列表失败'
  } finally {
    pickerLoading.value = false
  }
}

function openPicker() {
  pickerOpen.value = true
  resetPicker()
  loadPickerList()
}

function closePicker() {
  pickerOpen.value = false
}

function openFolder(item: UserFile) {
  if (!item.id) return
  pickerFolderId.value = item.id
  pickerPath.value = [...pickerPath.value, { id: item.id, name: item.name }]
  loadPickerList()
}

function navigatePicker(index: number) {
  const next = pickerPath.value.slice(0, index + 1)
  pickerPath.value = next
  pickerFolderId.value = next[next.length - 1]?.id || 0
  loadPickerList()
}

function appendReference(item: UserFile) {
  const exists = referencedFiles.value.some((ref) => ref.file_identity === item.identity)
  if (exists) return
  referencedFiles.value = [
    ...referencedFiles.value,
    {
      file_identity: item.identity,
      name: item.name,
      mime_type: undefined,
    },
  ]
}

function removeReference(identity?: string) {
  referencedFiles.value = referencedFiles.value.filter((item) => item.file_identity !== identity)
}

function applyStreamEvent(event: AgentStreamEvent) {
  if (event.session?.id) {
    activeSessionId.value = event.session.id
    sessions.value = [event.session, ...sessions.value.filter((item) => item.id !== event.session?.id)]
  }
  if (event.type === 'referenced_files' && event.referenced_files?.length) {
    referencedFiles.value = event.referenced_files
    return
  }
  if (event.type === 'assistant_delta' && event.content) {
    const last = messages.value[messages.value.length - 1]
    if (last?.role === 'assistant') {
      last.content += event.content
    } else {
      messages.value.push({ id: nextId('assistant'), role: 'assistant', content: event.content })
    }
    void scrollMessagesToBottom()
    return
  }
  if (event.type === 'tool_call') {
    messages.value.push({
      id: nextId('tool-call'),
      role: 'tool_call',
      content: event.arguments_json || '',
      toolName: event.tool_name,
    })
    return
  }
  if (event.type === 'tool_result') {
    messages.value.push({
      id: nextId('tool-result'),
      role: 'tool_result',
      content: event.content || '',
    })
    return
  }
  if (event.type === 'approval_required' && event.pending_interrupt) {
    messages.value.push({
      id: nextId('approval'),
      role: 'approval',
      content: event.pending_interrupt.arguments_json,
      toolName: event.pending_interrupt.tool_name,
      pending: event.pending_interrupt,
    })
    return
  }
  if (event.type === 'error' && event.error) {
    error.value = event.error
  }
}

async function sendMessage() {
  const content = input.value.trim()
  if (!content || loading.value) return
  error.value = ''
  loading.value = true
  try {
    const sessionId = await ensureSession()
    messages.value.push({ id: nextId('user'), role: 'user', content })
    await scrollMessagesToBottom()
    input.value = ''
    const token = getToken()
    if (!token) throw new Error('登录已失效，请重新登录')
    await agentChatStream(sessionId, content, referencedFiles.value, token, applyStreamEvent)
  } catch (e: any) {
    error.value = e?.message || '发送失败'
  } finally {
    loading.value = false
  }
}

async function resolveApproval(pending: AgentPendingInterrupt | undefined, approved: boolean) {
  if (!pending || resolvingInterrupt.value) return
  const token = getToken()
  if (!token) {
    error.value = '登录已失效，请重新登录'
    return
  }
  resolvingInterrupt.value = true
  error.value = ''
  try {
    await agentResumeStream(activeSessionId.value, pending.interrupt_id, approved, approved ? '' : rejectReason.value.trim(), token, applyStreamEvent)
    rejectReason.value = ''
    messages.value = messages.value.filter((item) => item.pending?.interrupt_id !== pending.interrupt_id)
  } catch (e: any) {
    error.value = e?.message || '处理审批失败'
  } finally {
    resolvingInterrupt.value = false
  }
}

async function startNewSession() {
  activeSessionId.value = ''
  messages.value = []
  referencedFiles.value = []
  rejectReason.value = ''
  await ensureSession()
}

async function switchSession(session: AgentSession) {
  activeSessionId.value = session.id
  messages.value = []
  referencedFiles.value = []
  rejectReason.value = ''
  await loadSessionDetail(session.id)
  void scrollMessagesToBottom()
}

async function loadSessionDetail(sessionId: string) {
  const token = getToken()
  if (!token || !sessionId) return
  const data = await getAgentSessionDetail(sessionId, token)
  if (data.session?.id) {
    sessions.value = [data.session, ...sessions.value.filter((item) => item.id !== data.session?.id)]
  }
  messages.value = (data.messages || []).map((item) => ({
    id: nextId(`history-${item.role}`),
    role: item.role === 'user' ? 'user' : 'assistant',
    content: item.content || '',
  }))
  if (data.session?.pending_interrupt_id) {
    messages.value.push({
      id: nextId('approval-stub'),
      role: 'approval',
      content: '该会话存在待处理审批，请继续或取消当前工具执行。',
      toolName: 'pending',
      pending: {
        interrupt_id: data.session.pending_interrupt_id,
        tool_name: '待恢复操作',
        arguments_json: '请确认是否继续执行上一次中断的工具调用。',
      },
    })
  }
}

async function scrollMessagesToBottom() {
  await nextTick()
  const el = messagePaneRef.value
  if (!el) return
  el.scrollTop = el.scrollHeight
}

onMounted(() => {
  loadSessions().catch((e: any) => {
    error.value = e?.message || '加载会话失败'
  })
})

watch(() => messages.value.length, () => {
  void scrollMessagesToBottom()
})
</script>

<template>
  <section class="h-full min-h-0 overflow-hidden rounded-[28px] border border-slate-200 bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.12),_transparent_28%),linear-gradient(180deg,_#f8fbff_0%,_#eef4f8_100%)] shadow-[0_24px_60px_rgba(15,23,42,0.08)]">
    <div class="grid h-full min-h-0 grid-cols-12">
      <aside class="col-span-3 xl:col-span-2 flex h-full min-h-0 flex-col border-r border-slate-200/80 bg-white/80">
        <div class="border-b border-slate-200 px-4 py-4">
          <div class="text-[11px] font-semibold uppercase tracking-[0.22em] text-sky-700">AI Workspace</div>
          <div class="mt-2 text-2xl font-semibold text-slate-900">文件智能管家</div>
          <p class="mt-2 text-sm leading-6 text-slate-500">引用你的云盘文件，交给 Agent 做检索、比对和总结。</p>
          <button class="mt-4 w-full rounded-2xl bg-slate-900 px-4 py-3 text-sm font-medium text-white transition hover:bg-sky-700" @click="startNewSession">
            新建会话
          </button>
        </div>
        <div class="flex-1 overflow-y-auto px-2 py-3">
          <button
            v-for="session in sessions"
            :key="session.id"
            class="mb-2 w-full rounded-2xl border px-4 py-3 text-left transition"
            :class="session.id === activeSessionId ? 'border-sky-500 bg-sky-50 text-slate-900 shadow-sm' : 'border-transparent bg-white text-slate-600 hover:border-slate-200 hover:bg-slate-50'"
            @click="switchSession(session)"
          >
            <div class="truncate text-sm font-medium">{{ session.title || '未命名会话' }}</div>
            <div class="mt-1 text-xs text-slate-400">{{ session.updated_at || '-' }}</div>
            <div v-if="session.pending_interrupt_id" class="mt-2 inline-flex rounded-full bg-amber-100 px-2 py-1 text-[11px] font-medium text-amber-700">
              待审批
            </div>
          </button>
        </div>
      </aside>

      <div class="col-span-9 xl:col-span-10 flex h-full min-h-0 flex-col">
        <div class="border-b border-slate-200/80 px-5 py-4">
          <div class="flex items-center justify-between gap-4">
            <div>
              <div class="text-sm text-slate-500">当前会话</div>
              <div class="mt-1 text-xl font-semibold text-slate-900">{{ activeSession?.title || '开始新的文件对话' }}</div>
            </div>
            <button class="rounded-2xl border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-sky-500 hover:text-sky-700" @click="openPicker">
              引用文件
            </button>
          </div>
          <div v-if="referencedFiles.length" class="mt-4 flex flex-wrap gap-2">
            <div
              v-for="item in referencedFiles"
              :key="item.file_identity || item.url || item.name"
              class="inline-flex items-center gap-2 rounded-full bg-slate-900 px-3 py-2 text-xs font-medium text-white"
            >
              <i class="fa fa-paperclip"></i>
              <span class="max-w-[180px] truncate">{{ item.name || item.file_identity }}</span>
              <button class="text-white/70 transition hover:text-white" @click="removeReference(item.file_identity)">×</button>
            </div>
          </div>
        </div>

        <div ref="messagePaneRef" class="flex-1 min-h-0 overflow-y-auto px-5 py-4">
          <div v-if="messages.length === 0" class="mx-auto max-w-3xl rounded-[28px] border border-dashed border-slate-300 bg-white/70 px-8 py-14 text-center">
            <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-3xl bg-sky-100 text-sky-700">
              <i class="fa fa-comments-o text-2xl"></i>
            </div>
            <h3 class="mt-5 text-2xl font-semibold text-slate-900">把文件引用进来，再开始提问</h3>
            <p class="mt-3 text-sm leading-7 text-slate-500">
              适合做文档总结、合同问答、方案比对、日志定位。当前版本优先支持文本、Markdown、JSON、CSV、YAML、XML。
            </p>
          </div>

          <div v-else class="mx-auto flex max-w-5xl flex-col gap-3">
            <article
              v-for="message in messages"
              :key="message.id"
              class="rounded-[24px] border px-5 py-3.5 shadow-sm"
              :class="{
                'ml-auto max-w-[82%] border-sky-200 bg-sky-50': message.role === 'user',
                'max-w-[88%] border-white bg-white': message.role === 'assistant',
                'max-w-[88%] border-violet-200 bg-violet-50': message.role === 'tool_call',
                'max-w-[88%] border-emerald-200 bg-emerald-50': message.role === 'tool_result',
                'max-w-[88%] border-amber-300 bg-amber-50': message.role === 'approval',
              }"
            >
              <div class="mb-2 text-[11px] font-semibold uppercase tracking-[0.2em] text-slate-400">
                {{
                  message.role === 'user'
                    ? '你'
                    : message.role === 'assistant'
                      ? '文件智能管家'
                      : message.role === 'tool_call'
                        ? `工具调用 · ${message.toolName || ''}`
                        : message.role === 'tool_result'
                          ? '工具结果'
                          : `等待审批 · ${message.toolName || ''}`
                }}
              </div>
              <pre class="whitespace-pre-wrap break-words text-sm leading-7 text-slate-700">{{ message.content }}</pre>

              <div v-if="message.role === 'approval' && message.pending" class="mt-4 rounded-2xl border border-amber-200 bg-white/70 p-4">
                <textarea
                  v-model="rejectReason"
                  rows="2"
                  class="w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-700 outline-none transition focus:border-sky-500"
                  placeholder="如需拒绝，可填写原因"
                />
                <div class="mt-3 flex gap-3">
                  <button class="rounded-2xl bg-emerald-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-50" :disabled="resolvingInterrupt" @click="resolveApproval(message.pending, true)">
                    {{ resolvingInterrupt ? '处理中...' : '继续执行' }}
                  </button>
                  <button class="rounded-2xl bg-rose-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-rose-700 disabled:cursor-not-allowed disabled:opacity-50" :disabled="resolvingInterrupt" @click="resolveApproval(message.pending, false)">
                    {{ resolvingInterrupt ? '处理中...' : '取消执行' }}
                  </button>
                </div>
              </div>
            </article>
          </div>
        </div>

        <div class="border-t border-slate-200/80 bg-white/80 px-5 py-4">
          <p v-if="error" class="mb-3 rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">{{ error }}</p>
          <div class="rounded-[24px] border border-slate-200 bg-white px-4 py-3 shadow-sm">
            <textarea
              v-model="input"
              rows="2"
              class="max-h-28 min-h-[56px] w-full resize-y bg-transparent text-sm leading-6 text-slate-800 outline-none"
              placeholder="例如：总结我引用的需求文档，并列出关键风险点"
              @keydown.enter.exact.prevent="sendMessage"
            />
            <div class="mt-3 flex items-center justify-between gap-4">
              <div class="text-xs text-slate-400">
                Enter 发送，Shift + Enter 换行
              </div>
              <div class="flex items-center gap-3">
                <button class="rounded-2xl border border-slate-300 px-4 py-2 text-sm font-medium text-slate-600 transition hover:border-sky-500 hover:text-sky-700" @click="openPicker">
                  选择文件
                </button>
                <button class="rounded-2xl bg-slate-900 px-5 py-2.5 text-sm font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-50" :disabled="loading || !input.trim()" @click="sendMessage">
                  {{ loading ? '发送中...' : '发送给管家' }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="pickerOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/35 p-6">
      <div class="flex h-[78vh] w-full max-w-4xl flex-col overflow-hidden rounded-[32px] bg-white shadow-[0_30px_80px_rgba(15,23,42,0.24)]">
        <div class="border-b border-slate-200 px-6 py-5">
          <div class="flex items-center justify-between gap-4">
            <div>
              <div class="text-xs font-semibold uppercase tracking-[0.2em] text-sky-700">File Picker</div>
              <div class="mt-2 text-xl font-semibold text-slate-900">引用云盘文件</div>
            </div>
            <button class="rounded-2xl border border-slate-300 px-4 py-2 text-sm text-slate-600 transition hover:border-slate-500 hover:text-slate-900" @click="closePicker">
              关闭
            </button>
          </div>
          <div class="mt-4 flex flex-wrap items-center gap-2 text-sm text-slate-500">
            <button
              v-for="(crumb, index) in pickerPath"
              :key="`${crumb.id}-${index}`"
              class="rounded-full px-3 py-1 transition hover:bg-slate-100 hover:text-slate-900"
              @click="navigatePicker(index)"
            >
              {{ crumb.name }}
            </button>
          </div>
        </div>

        <div class="flex-1 overflow-y-auto px-6 py-5">
          <p v-if="pickerError" class="mb-4 rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">{{ pickerError }}</p>
          <div v-if="pickerLoading" class="py-10 text-center text-sm text-slate-500">正在枚举文件列表...</div>
          <div v-else class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <button
              v-for="item in pickerList"
              :key="item.identity"
              class="flex items-start gap-3 rounded-[24px] border border-slate-200 bg-slate-50 px-4 py-4 text-left transition hover:border-sky-400 hover:bg-sky-50"
              @click="isFolder(item) ? openFolder(item) : appendReference(item)"
            >
              <div class="mt-1 flex h-11 w-11 items-center justify-center rounded-2xl" :class="isFolder(item) ? 'bg-amber-100 text-amber-700' : 'bg-sky-100 text-sky-700'">
                <i class="fa" :class="isFolder(item) ? 'fa-folder-open-o' : 'fa-file-text-o'"></i>
              </div>
              <div class="min-w-0 flex-1">
                <div class="truncate text-sm font-medium text-slate-900">{{ item.name }}</div>
                <div class="mt-1 text-xs text-slate-400">
                  {{ isFolder(item) ? '文件夹' : (item.ext || '文件') }}<span v-if="!isFolder(item)"> · {{ item.updated_at || '-' }}</span>
                </div>
              </div>
              <div class="shrink-0 text-xs font-medium text-slate-500">
                {{ isFolder(item) ? '进入' : '引用' }}
              </div>
            </button>
          </div>
        </div>

        <div class="border-t border-slate-200 bg-slate-50 px-6 py-4">
          <div class="flex items-center justify-between">
            <div class="text-sm text-slate-500">已选择 {{ referencedFiles.length }} 个文件引用</div>
            <button class="rounded-2xl bg-slate-900 px-5 py-2.5 text-sm font-medium text-white transition hover:bg-sky-700" @click="closePicker">
              完成
            </button>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
