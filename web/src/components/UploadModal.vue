<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { uploadFile } from '../lib/api'
import { getToken } from '../lib/auth'

type UploadItem = {
  file: File
  progress: number
  status: 'pending' | 'uploading' | 'success' | 'error'
  message?: string
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
  for (const item of files.value) {
    if (item.status !== 'pending') continue
    item.status = 'uploading'
    item.progress = 10
    try {
      const result = await uploadFile(item.file, props.parentId, token)
      item.progress = 100
      item.status = 'success'
      item.message = result.message
    } catch (e: any) {
      const message = e?.message || '上传失败'
      item.status = 'error'
      item.progress = 0
      item.message = message
      error.value = message
    }
  }
  emit('uploaded')
}
</script>

<template>
  <div v-show="props.visible" class="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center">
    <div class="bg-white rounded-xl shadow-lg w-full max-w-2xl max-h-[90vh] overflow-y-auto">
      <div class="p-6 border-b border-gray-light">
        <div class="flex items-center justify-between">
          <h2 class="text-xl font-bold">上传文件</h2>
          <button class="btn-icon-secondary" @click="hide">
            <i class="fa fa-times"></i>
          </button>
        </div>
      </div>
      <div class="p-6">
        <div class="border-2 border-dashed border-gray-light rounded-xl p-8 text-center mb-6" @drop="onDrop" @dragover="onDragOver">
          <div class="w-16 h-16 rounded-full bg-primary bg-opacity-10 text-primary flex items-center justify-center mx-auto mb-4">
            <i class="fa fa-cloud-upload text-2xl"></i>
          </div>
          <h3 class="font-medium mb-2">拖放文件到此处</h3>
          <p class="text-sm text-gray-medium mb-4">或者</p>
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
            </div>
          </div>
        </div>
      </div>
      <div class="p-6 border-t border-gray-light flex justify-end gap-3">
        <button class="btn-secondary" :disabled="hasUploading" @click="hide">取消</button>
        <button class="btn-primary" :disabled="!hasPending || hasUploading" @click="startUpload">继续上传</button>
      </div>
    </div>
  </div>
</template>
