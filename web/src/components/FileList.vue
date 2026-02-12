<script setup lang="ts">
import { computed } from 'vue'
import type { UserFile } from '../lib/api'

const props = defineProps<{ items: UserFile[]; view: 'detail' | 'medium' | 'large'; sortKey: 'name' | 'type' | 'size' | 'updated'; sortOrder: 'asc' | 'desc' }>()
const emit = defineEmits<{
  (e: 'download', item: UserFile): void
  (e: 'rename', item: UserFile): void
  (e: 'delete', item: UserFile): void
  (e: 'open', item: UserFile): void
  (e: 'change-sort', value: 'name' | 'type' | 'size' | 'updated'): void
  (e: 'share', item: UserFile): void
}>()

const gridClass = computed(() => props.view === 'large'
  ? 'grid-cols-2 md:grid-cols-3 lg:grid-cols-4'
  : 'grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6')
const iconBoxClass = computed(() => props.view === 'large' ? 'w-16 h-16' : 'w-12 h-12')
const iconClass = computed(() => props.view === 'large' ? 'text-3xl' : 'text-2xl')

function formatSize(size: number) {
  if (!size || size <= 0) return '-'
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`
  return `${(size / 1024 / 1024 / 1024).toFixed(1)} GB`
}

function formatDate(value: string) {
  if (!value) return '-'
  const trimmed = value.replace('T', ' ')
  return trimmed.length > 16 ? trimmed.slice(0, 16) : trimmed
}

function sortIcon(key: 'name' | 'type' | 'size' | 'updated') {
  if (props.sortKey !== key) return ''
  return props.sortOrder === 'asc' ? 'fa-sort-amount-asc' : 'fa-sort-amount-desc'
}

function isFolder(item: UserFile) {
  return !item.repository_identity
}

function getTypeMeta(ext: string) {
  const lower = (ext || '').toLowerCase()
  if (['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp'].includes(lower)) {
    return { label: '图片', icon: 'fa-file-image-o', color: 'text-blue-500', bg: 'bg-blue-50' }
  }
  if (['.mp4', '.avi', '.mov', '.mkv', '.flv', '.wmv', '.webm', '.m4v'].includes(lower)) {
    return { label: '视频', icon: 'fa-file-video-o', color: 'text-red-500', bg: 'bg-red-50' }
  }
  if (['.mp3', '.wav', '.aac', '.flac', '.ogg', '.m4a'].includes(lower)) {
    return { label: '音频', icon: 'fa-file-audio-o', color: 'text-indigo-500', bg: 'bg-indigo-50' }
  }
  if (['.pdf'].includes(lower)) {
    return { label: 'PDF文档', icon: 'fa-file-pdf-o', color: 'text-red-500', bg: 'bg-red-50' }
  }
  if (['.doc', '.docx'].includes(lower)) {
    return { label: 'Word文档', icon: 'fa-file-word-o', color: 'text-blue-500', bg: 'bg-blue-50' }
  }
  if (['.xls', '.xlsx'].includes(lower)) {
    return { label: '表格', icon: 'fa-file-excel-o', color: 'text-green-500', bg: 'bg-green-50' }
  }
  if (['.ppt', '.pptx'].includes(lower)) {
    return { label: '演示文稿', icon: 'fa-file-powerpoint-o', color: 'text-orange-500', bg: 'bg-orange-50' }
  }
  if (['.zip', '.rar', '.7z', '.tar', '.gz'].includes(lower)) {
    return { label: '压缩包', icon: 'fa-file-archive-o', color: 'text-purple-500', bg: 'bg-purple-50' }
  }
  if (['.txt', '.md'].includes(lower)) {
    return { label: '文档', icon: 'fa-file-text-o', color: 'text-green-500', bg: 'bg-green-50' }
  }
  return { label: '文件', icon: 'fa-file-o', color: 'text-gray-500', bg: 'bg-gray-50' }
}

function getItemMeta(item: UserFile) {
  if (isFolder(item)) {
    return { label: '文件夹', icon: 'fa-folder', color: 'text-yellow-500', bg: 'bg-yellow-50' }
  }
  return getTypeMeta(item.ext)
}
</script>

<template>
  <div class="bg-white rounded-xl shadow-card overflow-hidden">
    <div v-show="props.view === 'detail'" class="hidden md:grid grid-cols-12 gap-4 px-6 py-3 border-b border-gray-light text-sm font-medium text-gray-medium">
      <button class="col-span-5 flex items-center gap-2 text-left" @click="emit('change-sort', 'name')">
        <span>文件名</span>
        <i v-if="sortIcon('name')" class="fa text-xs" :class="sortIcon('name')"></i>
      </button>
      <button class="col-span-2 flex items-center gap-2" @click="emit('change-sort', 'type')">
        <span>类型</span>
        <i v-if="sortIcon('type')" class="fa text-xs" :class="sortIcon('type')"></i>
      </button>
      <button class="col-span-2 flex items-center gap-2" @click="emit('change-sort', 'size')">
        <span>大小</span>
        <i v-if="sortIcon('size')" class="fa text-xs" :class="sortIcon('size')"></i>
      </button>
      <button class="col-span-2 flex items-center gap-2" @click="emit('change-sort', 'updated')">
        <span>修改日期</span>
        <i v-if="sortIcon('updated')" class="fa text-xs" :class="sortIcon('updated')"></i>
      </button>
      <div class="col-span-1 text-right">操作</div>
    </div>
    <div class="divide-y divide-gray-light">
      <div v-show="props.view !== 'detail'" class="grid gap-4 p-4" :class="gridClass">
        <div v-for="item in props.items" :key="item.identity" class="bg-white rounded-xl border border-gray-light p-4 file-hover" @dblclick="isFolder(item) && emit('open', item)">
          <div class="flex items-center justify-between mb-3">
            <div class="rounded-lg flex items-center justify-center" :class="iconBoxClass + ' ' + getItemMeta(item).bg + ' ' + getItemMeta(item).color">
              <i class="fa" :class="getItemMeta(item).icon + ' ' + iconClass"></i>
            </div>
            <div class="relative group">
              <button class="btn-icon-secondary w-8 h-8">
                <i class="fa fa-ellipsis-v"></i>
              </button>
              <div class="absolute right-0 top-full mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-light opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200 z-10">
                <div class="py-1">
                  <button v-if="!isFolder(item)" class="w-full flex items-center gap-3 px-4 py-2 text-sm hover:bg-gray-50" @click="emit('download', item)">
                    <i class="fa fa-download w-5 text-gray-medium"></i>
                    <span>下载</span>
                  </button>
                  <button v-if="!isFolder(item)" class="w-full flex items-center gap-3 px-4 py-2 text-sm hover:bg-gray-50" @click="emit('share', item)">
                    <i class="fa fa-share-alt w-5 text-gray-medium"></i>
                    <span>创建分享</span>
                  </button>
                  <button class="w-full flex items-center gap-3 px-4 py-2 text-sm hover:bg-gray-50" @click="emit('rename', item)">
                    <i class="fa fa-pencil w-5 text-gray-medium"></i>
                    <span>重命名</span>
                  </button>
                  <button class="w-full flex items-center gap-3 px-4 py-2 text-sm hover:bg-gray-50 text-red-500" @click="emit('delete', item)">
                    <i class="fa fa-trash w-5"></i>
                    <span>删除</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
          <h3 class="font-medium truncate" @click="isFolder(item) && emit('open', item)">{{ item.name }}</h3>
          <p class="text-xs text-gray-medium mt-1">{{ isFolder(item) ? '文件夹' : formatSize(item.size) }} · —</p>
        </div>
      </div>
      <div v-show="props.view === 'detail'">
        <div v-for="item in props.items" :key="item.identity" class="grid grid-cols-12 gap-4 px-6 py-4 items-center hover:bg-gray-50">
          <div class="col-span-5 flex items-center gap-3">
            <div class="w-10 h-10 rounded-lg flex items-center justify-center" :class="getItemMeta(item).bg + ' ' + getItemMeta(item).color">
              <i class="fa" :class="getItemMeta(item).icon"></i>
            </div>
            <div>
              <h3 class="font-medium cursor-pointer" @click="isFolder(item) && emit('open', item)">{{ item.name }}</h3>
              <p class="text-xs text-gray-medium md:hidden">{{ getItemMeta(item).label }} · {{ isFolder(item) ? '-' : formatSize(item.size) }} · —</p>
            </div>
          </div>
          <div class="col-span-2 hidden md:flex items-center text-sm">
            <span class="px-2 py-1 rounded text-xs" :class="getItemMeta(item).bg + ' ' + getItemMeta(item).color">{{ getItemMeta(item).label }}</span>
          </div>
          <div class="col-span-2 hidden md:flex items-center text-sm text-gray-medium">{{ isFolder(item) ? '-' : formatSize(item.size) }}</div>
          <div class="col-span-2 hidden md:flex items-center text-sm text-gray-medium">{{ isFolder(item) ? '-' : formatDate(item.updated_at) }}</div>
          <div class="col-span-1 flex justify-end">
            <div class="relative group">
              <button class="btn-icon-secondary w-8 h-8">
                <i class="fa fa-ellipsis-v"></i>
              </button>
              <div class="absolute right-0 top-full mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-light opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200 z-10">
                <div class="py-1">
                  <button v-if="!isFolder(item)" class="w-full flex items-center gap-3 px-4 py-2 text-sm hover:bg-gray-50" @click="emit('download', item)">
                    <i class="fa fa-download w-5 text-gray-medium"></i>
                    <span>下载</span>
                  </button>
                  <button v-if="!isFolder(item)" class="w-full flex items-center gap-3 px-4 py-2 text-sm hover:bg-gray-50" @click="emit('share', item)">
                    <i class="fa fa-share-alt w-5 text-gray-medium"></i>
                    <span>创建分享</span>
                  </button>
                  <button class="w-full flex items-center gap-3 px-4 py-2 text-sm hover:bg-gray-50" @click="emit('rename', item)">
                    <i class="fa fa-pencil w-5 text-gray-medium"></i>
                    <span>重命名</span>
                  </button>
                  <button class="w-full flex items-center gap-3 px-4 py-2 text-sm hover:bg-gray-50 text-red-500" @click="emit('delete', item)">
                    <i class="fa fa-trash w-5"></i>
                    <span>删除</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
