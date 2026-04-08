<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { API_BASE } from '../lib/api'
import { getToken } from '../lib/auth'

type UploadItem = {
  file: File
  progress: number
  status: 'pending' | 'uploading' | 'success' | 'error'
  message?: string
  xhr?: XMLHttpRequest | null
}

const props = defineProps<{ visible: boolean; parentId: number }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'uploaded'): void }>()
const files = ref<UploadItem[]>([])
const inputRef = ref<HTMLInputElement | null>(null)
const error = ref('')

const hasPending = computed(() => files.value.some((item) => item.status === 'pending'))
const hasUploading = computed(() => files.value.some((item) => item.status === 'uploading'))

watch(() => props.visible, (value) => {
  if (!value) {
    files.value = []
    error.value = ''
  }
})

function hide() {
  if (hasUploading.value) return
  emit('close')
}

function openPicker() {
  inputRef.value?.click()
}

function enqueueFiles(selected: FileList | null) {
  if (!selected || selected.length === 0) return
  const list = Array.from(selected).map((file) => ({ file, progress: 0, status: 'pending' as const }))
  files.value = [...files.value, ...list]
}

function onFilesSelected(e: Event) {
  const target = e.target as HTMLInputElement
  enqueueFiles(target.files)
  if (target) target.value = ''
}

function onDrop(e: DragEvent) {
  e.preventDefault()
  enqueueFiles(e.dataTransfer?.files || null)
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
}

function formatSize(size: number) {
  if (!size || size <= 0) return '0B'
  if (size < 1024) return `${size}B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)}KB`
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)}MB`
  return `${(size / 1024 / 1024 / 1024).toFixed(1)}GB`
}

async function startUpload() {
  if (!hasPending.value || hasUploading.value) return
  const token = getToken()
  if (!token) {
    error.value = '登录已失效，请重新登录'
    return
  }
  error.value = ''
  await Promise.all(files.value.map((item) => doUpload(item, token)))
  emit('uploaded')
}

function doUpload(item: UploadItem, token: string) {
  if (item.status !== 'pending') return Promise.resolve()
  item.status = 'uploading'
  item.progress = 0
  const form = new FormData()
  form.append('file', item.file)
  form.append('parent_id', String(props.parentId))
  return new Promise<void>((resolve) => {
    const xhr = new XMLHttpRequest()
    item.xhr = xhr
    xhr.open('POST', `${API_BASE}/api/file/upload`)
    xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) {
        const p = Math.floor((e.loaded / e.total) * 100)
        item.progress = Math.min(99, Math.max(0, p))
      }
    }
    xhr.onreadystatechange = () => {
      if (xhr.readyState !== 4) return
      item.xhr = null
      if (xhr.status === 0) {
        item.status = 'error'
        item.progress = 0
        item.message = '已取消'
        resolve()
        return
      }
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const json = JSON.parse(xhr.responseText)
          if (json && typeof json.code === 'number' && json.code === 0) {
            item.status = 'success'
            item.progress = 100
            item.message = json.data?.message || '上传任务已入队'
            resolve()
            return
          }
          item.status = 'error'
          item.progress = 0
          item.message = json?.msg || '上传失败'
        } catch {
          item.status = 'error'
          item.progress = 0
          item.message = '响应异常'
        }
        resolve()
        return
      }
      if (xhr.status === 413) {
        item.status = 'error'
        item.progress = 0
        item.message = '上传被拒绝(413)：可能超过 Nginx/网关限制，请检查 client_max_body_size'
        resolve()
        return
      }
      item.status = 'error'
      item.progress = 0
      item.message = `HTTP ${xhr.status}`
      resolve()
    }
    xhr.onerror = () => {
      item.status = 'error'
      item.progress = 0
      item.message = '网络异常'
      item.xhr = null
      resolve()
    }
    xhr.send(form)
  })
}

function cancelItem(item: UploadItem) {
  if (item.status !== 'uploading' || !item.xhr) return
  try { item.xhr.abort() } catch {}
}

function retryItem(item: UploadItem) {
  if (item.status !== 'error') return
  item.status = 'pending'
  item.progress = 0
  item.message = ''
}
</script>

<template>
  <div v-show="props.visible" class="fixed bottom-6 right-6 z-50 w-[28rem] max-w-[calc(100vw-2rem)]">
    <div class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-[0_20px_60px_rgba(15,23,42,0.18)]">
      <div class="p-4 border-b border-gray-light bg-slate-50">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-base font-bold text-slate-900">上传任务</h2>
            <p class="text-xs text-slate-500 mt-1">当前目录 ID: {{ props.parentId }}</p>
          </div>
          <button class="btn-icon-secondary" @click="hide">
            <i class="fa fa-times"></i>
          </button>
        </div>
      </div>
      <div class="p-4 max-h-[70vh] overflow-y-auto">
        <div class="border-2 border-dashed border-gray-light rounded-xl p-5 text-center mb-4" @drop="onDrop" @dragover="onDragOver">
          <div class="w-14 h-14 rounded-full bg-primary bg-opacity-10 text-primary flex items-center justify-center mx-auto mb-3">
            <i class="fa fa-cloud-upload text-2xl"></i>
          </div>
          <h3 class="font-medium mb-2">拖放文件到这里</h3>
          <p class="text-sm text-gray-medium mb-3">或选择文件后直接加入上传队列</p>
          <button class="btn-primary" @click="openPicker">
            <i class="fa fa-folder-open"></i>
            <span>选择文件</span>
          </button>
          <input ref="inputRef" type="file" class="hidden" multiple @change="onFilesSelected" />
          <p class="text-xs text-gray-medium mt-4">支持的文件类型：所有常见格式</p>
        </div>
        <div v-if="error" class="text-sm text-red-500 mb-4">{{ error }}</div>
        <div v-if="files.length === 0" class="text-sm text-gray-medium">暂无待上传文件</div>
        <div v-else class="space-y-4">
          <div v-for="item in files" :key="item.file.name + item.file.size + item.file.lastModified" class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-blue-50 text-blue-500 flex items-center justify-center">
                <i class="fa fa-file-o"></i>
              </div>
              <div>
                <h3 class="font-medium truncate max-w-[240px]">{{ item.file.name }}</h3>
                <p class="text-xs text-gray-medium">{{ formatSize(item.file.size) }}</p>
                <p v-if="item.status === 'error'" class="text-xs text-red-500">{{ item.message || '上传失败' }}</p>
                <p v-if="item.status === 'success' && item.message" class="text-xs text-green-600">{{ item.message }}</p>
              </div>
            </div>
            <div class="flex items-center gap-3">
              <span class="text-sm text-gray-medium">{{ item.status === 'success' ? '100' : item.progress }}%</span>
              <div class="w-24 bg-gray-light rounded-full h-1.5">
                <div class="bg-primary h-1.5 rounded-full" :style="{ width: (item.status === 'success' ? 100 : item.progress) + '%' }"></div>
              </div>
              <button v-if="item.status === 'uploading'" class="btn-icon-secondary" @click="cancelItem(item)"><i class="fa fa-stop"></i></button>
              <button v-if="item.status === 'error'" class="btn-icon-secondary" @click="retryItem(item)"><i class="fa fa-repeat"></i></button>
            </div>
          </div>
        </div>
      </div>
      <div class="p-4 border-t border-gray-light flex justify-end gap-3 bg-white">
        <button class="btn-secondary" :disabled="hasUploading" @click="hide">取消</button>
        <button class="btn-primary" :disabled="!hasPending || hasUploading" @click="startUpload">继续上传</button>
      </div>
    </div>
  </div>
</template>
