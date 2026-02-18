<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import FileBreadcrumb from './FileBreadcrumb.vue'
import FileToolbar from './FileToolbar.vue'
import FileList from './FileList.vue'
import { createFolder, deleteUserItem, getDownloadUrl, getUserFileList, renameUserFile, createShare, type UserFile } from '../lib/api'
import { getToken } from '../lib/auth'

type Crumb = { id: number; name: string }

const props = defineProps<{ active: string; search: string; refreshKey: number }>()
const emit = defineEmits<{ (e: 'open-upload', parentId: number): void }>()

const loading = ref(false)
const error = ref('')
const view = ref<'detail' | 'medium' | 'large'>('detail')
const sortKey = ref<'name' | 'type' | 'size' | 'updated'>('updated')
const sortOrder = ref<'asc' | 'desc'>('desc')
const currentFolderId = ref(0)
const path = ref<Crumb[]>([{ id: 0, name: '我的文件' }])
const source = ref<'api' | 'mock'>((localStorage.getItem('fw_source') as 'api' | 'mock') || 'api')
const list = ref<UserFile[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const infiniteMode = ref(false)
const aggregated = ref<UserFile[]>([])
const sentinelRef = ref<HTMLDivElement | null>(null)
let sentinelObserver: IntersectionObserver | null = null

const createFolderOpen = ref(false)
const createFolderName = ref('')
const renameOpen = ref(false)
const renameName = ref('')
const renameTarget = ref<UserFile | null>(null)
const deleteOpen = ref(false)
const deleteTarget = ref<UserFile | null>(null)

const shareOpen = ref(false)
const shareTarget = ref<UserFile | null>(null)
const shareLoading = ref(false)
const shareError = ref('')
const shareExpired = ref(0)
const shareIdentity = ref('')
const shareLink = ref('')

type MockNode = UserFile & { parent_id: number }
const mockNodes = ref<MockNode[]>([
  { id: 1, identity: 'mk-folder-1', name: '工作文档', ext: '', size: 0, repository_identity: '', updated_at: '2026-02-01 09:00:00', parent_id: 0 },
  { id: 2, identity: 'mk-folder-2', name: '照片', ext: '', size: 0, repository_identity: '', updated_at: '2026-02-01 09:10:00', parent_id: 0 },
  { id: 3, identity: 'mk-folder-3', name: '空文件夹', ext: '', size: 0, repository_identity: '', updated_at: '2026-02-01 09:20:00', parent_id: 0 },
  { id: 4, identity: 'mk-file-4', name: '发布说明.txt', ext: '.txt', size: 1024 * 121, repository_identity: 'repo-m-4', updated_at: '2026-02-01 11:30:00', parent_id: 0 },
  { id: 5, identity: 'mk-file-5', name: '照片合集.zip', ext: '.zip', size: 1024 * 1024 * 80.3, repository_identity: 'repo-m-5', updated_at: '2026-02-02 09:30:00', parent_id: 0 },

  { id: 11, identity: 'mk-folder-11', name: '2024 年度报告', ext: '', size: 0, repository_identity: '', updated_at: '2026-02-02 10:00:00', parent_id: 1 },
  { id: 12, identity: 'mk-file-12', name: '会议纪要.docx', ext: '.docx', size: 1024 * 1024 * 11.2, repository_identity: 'repo-m-12', updated_at: '2026-02-02 11:00:00', parent_id: 1 },
  { id: 13, identity: 'mk-file-13', name: '预算表.xlsx', ext: '.xlsx', size: 1024 * 1024 * 1.5, repository_identity: 'repo-m-13', updated_at: '2026-02-02 11:40:00', parent_id: 1 },

  { id: 21, identity: 'mk-file-21', name: '2024-01.jpg', ext: '.jpg', size: 1024 * 1200, repository_identity: 'repo-m-21', updated_at: '2026-02-03 08:00:00', parent_id: 2 },
  { id: 22, identity: 'mk-file-22', name: '产品视频.mp4', ext: '.mp4', size: 1024 * 1024 * 555.5, repository_identity: 'repo-m-22', updated_at: '2026-02-03 20:00:00', parent_id: 2 },
  { id: 23, identity: 'mk-folder-23', name: '子文件夹A', ext: '', size: 0, repository_identity: '', updated_at: '2026-02-03 21:00:00', parent_id: 2 },

  { id: 24, identity: 'mk-file-24', name: '音频样例.mp3', ext: '.mp3', size: 1024 * 1024 * 27.1, repository_identity: 'repo-m-24', updated_at: '2026-02-04 12:00:00', parent_id: 23 },
  { id: 25, identity: 'mk-folder-25', name: '嵌套子文件夹B', ext: '', size: 0, repository_identity: '', updated_at: '2026-02-04 13:00:00', parent_id: 23 },
  { id: 26, identity: 'mk-file-26', name: '文档示例.pdf', ext: '.pdf', size: 1024 * 1024 * 8.3, repository_identity: 'repo-m-26', updated_at: '2026-02-04 14:00:00', parent_id: 25 },
])

function getMockList(pid: number, pageNum: number, sizeNum: number) {
  const rows = mockNodes.value.filter((n) => n.parent_id === pid)
  const totalCount = rows.length
  const start = (pageNum - 1) * sizeNum
  const listData = rows.slice(start, start + sizeNum)
  return { list: listData, count: totalCount }
}

function setSource(value: 'api' | 'mock') {
  source.value = value
  localStorage.setItem('fw_source', value)
  page.value = 1
  refresh()
}

const supported = computed(() => ['文件资源管理器', '图片', '视频', '音频', '文档', '压缩包'].includes(props.active))
const rootLabel = computed(() => (props.active === '文件资源管理器' ? '我的文件' : props.active))

watch(rootLabel, (value) => {
  const saved = localStorage.getItem('fw_state')
  if (saved) {
    try {
      const state = JSON.parse(saved)
      if (state && typeof state.id === 'number' && Array.isArray(state.path)) {
        currentFolderId.value = state.id
        path.value = state.path
      } else {
        path.value = [{ id: 0, name: value }]
        currentFolderId.value = 0
      }
    } catch {
      path.value = [{ id: 0, name: value }]
      currentFolderId.value = 0
    }
  } else {
    path.value = [{ id: 0, name: value }]
    currentFolderId.value = 0
  }
  page.value = 1
})

watch(() => props.active, () => {
  if (!supported.value) return
  page.value = 1
  refresh()
})

watch(() => props.refreshKey, () => {
  if (!supported.value) return
  page.value = 1
  refresh()
})

watch(() => props.search, () => {
  if (!supported.value) return
  page.value = 1
  refresh()
})

watch(currentFolderId, () => {
  if (!supported.value) return
  localStorage.setItem('fw_state', JSON.stringify({ id: currentFolderId.value, path: path.value }))
  refresh()
})

function isFolder(item: UserFile) {
  return !item.repository_identity
}

const filtered = computed(() => {
  const keyword = props.search?.trim()
  // 使用 Set 去重，避免重复数据
  const uniqueItems = Array.from(new Map(list.value.map(item => [item.identity, item])).values())
  let items = uniqueItems.slice()
  
  if (keyword) items = items.filter((item) => item.name.includes(keyword))
  if (props.active === '图片') {
    items = items.filter((item) => ['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp'].includes(item.ext?.toLowerCase()))
  } else if (props.active === '视频') {
    items = items.filter((item) => ['.mp4', '.avi', '.mov', '.mkv', '.flv', '.wmv', '.webm', '.m4v'].includes(item.ext?.toLowerCase()))
  } else if (props.active === '音频') {
    items = items.filter((item) => ['.mp3', '.wav', '.aac', '.flac', '.ogg', '.m4a'].includes(item.ext?.toLowerCase()))
  } else if (props.active === '文档') {
    items = items.filter((item) => ['.pdf', '.doc', '.docx', '.xls', '.xlsx', '.ppt', '.pptx', '.txt', '.md'].includes(item.ext?.toLowerCase()))
  } else if (props.active === '压缩包') {
    items = items.filter((item) => ['.zip', '.rar', '.7z', '.tar', '.gz'].includes(item.ext?.toLowerCase()))
  }
  return items
})

const files = computed(() => filtered.value.filter((item) => !isFolder(item)))

const sortedItems = computed(() => {
  // 先去重再排序
  const uniqueData = Array.from(new Map(filtered.value.map(item => [item.identity, item])).values())
  const data = uniqueData.slice()
  
  // 调试信息
  if (data.length !== uniqueData.length) {
    console.warn('排序前发现重复数据:', {
      originalLength: filtered.value.length,
      uniqueLength: uniqueData.length,
      sortedLength: data.length
    })
  }
  
  data.sort((a, b) => {
    const aFolder = isFolder(a)
    const bFolder = isFolder(b)
    if (aFolder !== bFolder) return aFolder ? -1 : 1
    let result = 0
    if (sortKey.value === 'name') {
      result = a.name.localeCompare(b.name, 'zh-Hans-CN')
    } else if (sortKey.value === 'size') {
      result = (a.size || 0) - (b.size || 0)
    } else if (sortKey.value === 'type') {
      result = (a.ext || '').localeCompare(b.ext || '', 'zh-Hans-CN')
    } else {
      const aTime = Date.parse(a.updated_at || '') || 0
      const bTime = Date.parse(b.updated_at || '') || 0
      result = aTime - bTime
    }
    return sortOrder.value === 'asc' ? result : -result
  })
  return data
})

const stats = computed(() => {
  const buckets = {
    image: { label: '图片', size: 0, count: 0, color: 'bg-blue-500', bg: 'bg-blue-100', icon: 'fa-file-image-o' },
    video: { label: '视频', size: 0, count: 0, color: 'bg-red-500', bg: 'bg-red-100', icon: 'fa-file-video-o' },
    doc: { label: '文档', size: 0, count: 0, color: 'bg-green-500', bg: 'bg-green-100', icon: 'fa-file-text-o' },
    zip: { label: '压缩包', size: 0, count: 0, color: 'bg-purple-500', bg: 'bg-purple-100', icon: 'fa-file-archive-o' },
  }
  const imageExt = ['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp']
  const videoExt = ['.mp4', '.avi', '.mov', '.mkv', '.flv', '.wmv', '.webm', '.m4v']
  const docExt = ['.pdf', '.doc', '.docx', '.xls', '.xlsx', '.ppt', '.pptx', '.txt', '.md']
  const zipExt = ['.zip', '.rar', '.7z', '.tar', '.gz']
  for (const item of files.value) {
    const ext = item.ext?.toLowerCase() || ''
    if (imageExt.includes(ext)) {
      buckets.image.size += item.size || 0
      buckets.image.count += 1
    } else if (videoExt.includes(ext)) {
      buckets.video.size += item.size || 0
      buckets.video.count += 1
    } else if (docExt.includes(ext)) {
      buckets.doc.size += item.size || 0
      buckets.doc.count += 1
    } else if (zipExt.includes(ext)) {
      buckets.zip.size += item.size || 0
      buckets.zip.count += 1
    }
  }
  const totalSize = Object.values(buckets).reduce((sum, b) => sum + b.size, 0)
  return Object.values(buckets).map((b) => ({
    ...b,
    percent: totalSize > 0 ? Math.round((b.size / totalSize) * 100) : 0,
  }))
})

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const pageNumbers = computed(() => {
  const count = pageCount.value
  const current = page.value
  const start = Math.max(1, current - 2)
  const end = Math.min(count, start + 4)
  const numbers = [] as number[]
  for (let i = start; i <= end; i += 1) numbers.push(i)
  return numbers
})

function formatSize(size: number) {
  if (!size || size <= 0) return '0B'
  if (size < 1024) return `${size}B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)}KB`
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)}MB`
  return `${(size / 1024 / 1024 / 1024).toFixed(1)}GB`
}

async function refresh() {
  const token = getToken()
  if (!token) return
  loading.value = true
  error.value = ''
  try {
    let data: { list: UserFile[]; count: number }
    if (source.value === 'mock') {
      const mock = getMockList(currentFolderId.value, page.value, pageSize.value)
      data = { list: mock.list, count: mock.count }
    } else {
      data = await getUserFileList(currentFolderId.value, page.value, pageSize.value, token)
    }
    
    // 数据去重处理
    const uniqueList = Array.from(new Map((data.list || []).map(item => [item.identity, item])).values())
    
    // 调试信息
    console.log('刷新数据:', {
      source: source.value,
      page: page.value,
      originalCount: (data.list || []).length,
      uniqueCount: uniqueList.length,
      infiniteMode: infiniteMode.value,
      aggregatedLength: aggregated.value.length
    })
    
    list.value = uniqueList
    total.value = data.count || 0
    
    if (infiniteMode.value) {
      if (page.value === 1) {
        // 第一页时清空聚合数据
        aggregated.value = uniqueList.slice()
      } else {
        // 后续页面合并时也要去重
        const combined = [...aggregated.value, ...uniqueList]
        aggregated.value = Array.from(new Map(combined.map(item => [item.identity, item])).values())
      }
      console.log('滚动加载模式聚合数据长度:', aggregated.value.length)
    }
    
    const maxPage = Math.max(1, Math.ceil(total.value / pageSize.value))
    if (page.value > maxPage) {
      page.value = maxPage
      await refresh()
      return
    }
  } catch (e: any) {
    error.value = e?.message || '加载失败'
  } finally {
    loading.value = false
  }
}

function onNavigate(index: number) {
  const next = path.value.slice(0, index + 1)
  path.value = next
  currentFolderId.value = next[next.length - 1]?.id || 0
  page.value = 1
  localStorage.setItem('fw_state', JSON.stringify({ id: currentFolderId.value, path: path.value }))
}

function onOpenFolder(item: UserFile) {
  if (!item.id) return
  path.value = [...path.value, { id: item.id, name: item.name }]
  currentFolderId.value = item.id
  page.value = 1
  localStorage.setItem('fw_state', JSON.stringify({ id: currentFolderId.value, path: path.value }))
}

function openCreateFolder() {
  createFolderName.value = ''
  createFolderOpen.value = true
}

async function submitCreateFolder() {
  if (!createFolderName.value.trim()) return
  const token = getToken()
  if (!token) return
  loading.value = true
  try {
    await createFolder(currentFolderId.value, createFolderName.value.trim(), token)
    createFolderOpen.value = false
    await refresh()
  } catch (e: any) {
    error.value = e?.message || '创建失败'
  } finally {
    loading.value = false
  }
}

function openRename(item: UserFile) {
  renameTarget.value = item
  renameName.value = item.name
  renameOpen.value = true
}

async function submitRename() {
  if (!renameTarget.value) return
  const token = getToken()
  if (!token) return
  loading.value = true
  try {
    await renameUserFile(renameTarget.value.identity, renameName.value.trim(), token)
    renameOpen.value = false
    await refresh()
  } catch (e: any) {
    error.value = e?.message || '重命名失败'
  } finally {
    loading.value = false
  }
}

function openDelete(item: UserFile) {
  deleteTarget.value = item
  deleteOpen.value = true
}

async function submitDelete() {
  if (!deleteTarget.value) return
  const token = getToken()
  if (!token) return
  loading.value = true
  try {
    await deleteUserItem(deleteTarget.value.identity, token)
    deleteOpen.value = false
    await refresh()
  } catch (e: any) {
    error.value = e?.message || '删除失败'
  } finally {
    loading.value = false
  }
}

function openShare(item: UserFile) {
  shareTarget.value = item
  shareIdentity.value = ''
  shareLink.value = ''
  shareError.value = ''
  shareExpired.value = 0
  shareOpen.value = true
}

function closeShare() {
  if (shareLoading.value) return
  shareOpen.value = false
}

function copyShareLink() {
  if (!shareLink.value) return
  if (navigator.clipboard) navigator.clipboard.writeText(shareLink.value)
}

function saveShareRecord(record: { identity: string; repository_identity: string; name: string; ext: string; size: number; created_at: string }) {
  const saved = localStorage.getItem('my_shares')
  const list = saved ? JSON.parse(saved) : []
  list.unshift(record)
  localStorage.setItem('my_shares', JSON.stringify(list.slice(0, 200)))
}

async function submitShare() {
  if (!shareTarget.value?.repository_identity) {
    shareError.value = '仅支持文件分享'
    return
  }
  const token = getToken()
  if (!token) {
    shareError.value = '登录已失效'
    return
  }
  shareLoading.value = true
  shareError.value = ''
  try {
    const data = await createShare(shareTarget.value.repository_identity, Math.max(0, Number(shareExpired.value) || 0), token)
    shareIdentity.value = data.identity
    shareLink.value = `${location.origin}/s/${data.identity}`
    saveShareRecord({
      identity: data.identity,
      repository_identity: shareTarget.value.repository_identity,
      name: shareTarget.value.name,
      ext: shareTarget.value.ext,
      size: shareTarget.value.size,
      created_at: new Date().toISOString(),
    })
  } catch (e: any) {
    shareError.value = e?.message || '创建失败'
  } finally {
    shareLoading.value = false
  }
}

async function onDownload(item: UserFile) {
  if (!item.repository_identity) return
  const token = getToken()
  if (!token) return
  try {
    const data = await getDownloadUrl(item.repository_identity, 3600, token)
    window.open(data.url, '_blank')
  } catch (e: any) {
    error.value = e?.message || '下载失败'
  }
}

function onOpenUpload() {
  emit('open-upload', currentFolderId.value)
}

function onChangeSort(key: 'name' | 'type' | 'size' | 'updated') {
  if (sortKey.value === key) {
    sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
    return
  }
  sortKey.value = key
  sortOrder.value = key === 'updated' ? 'desc' : 'asc'
}

function onToolbarSortKeyChange(key: 'name' | 'type' | 'size' | 'updated') {
  if (sortKey.value !== key) {
    sortKey.value = key
    sortOrder.value = key === 'updated' ? 'desc' : 'asc'
  }
}

function setPage(value: number) {
  if (value < 1 || value > pageCount.value || value === page.value) return
  page.value = value
  refresh()
}

function goPrev() {
  setPage(page.value - 1)
}

function goNext() {
  setPage(page.value + 1)
}

onMounted(() => {
  const saved = localStorage.getItem('fw_state')
  if (saved) {
    try {
      const state = JSON.parse(saved)
      if (state && typeof state.id === 'number' && Array.isArray(state.path)) {
        currentFolderId.value = state.id
        path.value = state.path
      }
    } catch {
      currentFolderId.value = 0
    }
  }
  if (supported.value) refresh()
  if (typeof IntersectionObserver !== 'undefined') {
    sentinelObserver = new IntersectionObserver(async (entries) => {
      if (!infiniteMode.value) return
      const entry = entries[0]
      if (entry && entry.isIntersecting && !loading.value) {
        if (list.value.length === 0) return
        if (aggregated.value.length >= total.value) return
        page.value += 1
        await refresh()
      }
    })
  }
})

watch(infiniteMode, async (value) => {
  // 切换模式时清空聚合数据并重置
  aggregated.value = []
  page.value = 1
  await refresh()
  if (value && sentinelRef.value && sentinelObserver) {
    sentinelObserver.observe(sentinelRef.value)
  } else if (sentinelObserver && sentinelRef.value) {
    sentinelObserver.unobserve(sentinelRef.value)
  }
})

watch(() => sentinelRef.value, (el) => {
  if (!infiniteMode.value) return
  if (sentinelObserver && el) sentinelObserver.observe(el)
})
</script>

<template>
  <main class="flex-1 overflow-y-auto p-6">
    <FileBreadcrumb :path="path" @navigate="onNavigate" />
    <div v-if="!supported" class="bg-white rounded-xl shadow-card p-10 text-center text-gray-medium">
      {{ props.active }} 暂未开放
    </div>
    <template v-else>
      <FileToolbar :view="view" :sort-key="sortKey" :sort-order="sortOrder" :loading="loading" @open-upload="onOpenUpload" @create-folder="openCreateFolder" @change-view="view = $event" @change-sort-key="onToolbarSortKeyChange" @change-sort-order="sortOrder = $event" />
      <div class="flex items-center gap-2 mb-4">
        <button class="btn-secondary" :class="{ 'active-view': source === 'api' }" @click="setSource('api')">真实数据</button>
        <button class="btn-secondary" :class="{ 'active-view': source === 'mock' }" @click="setSource('mock')">模拟数据</button>
        <button class="btn-secondary" :class="{ 'active-view': infiniteMode }" @click="infiniteMode = !infiniteMode">滚动加载</button>
      </div>
      <div v-if="error" class="mb-4 text-sm text-red-500">{{ error }}</div>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div v-for="stat in stats" :key="stat.label" class="bg-white rounded-xl shadow-card p-5 file-hover">
          <div class="flex items-center justify-between mb-4">
            <div class="w-10 h-10 rounded-lg flex items-center justify-center" :class="stat.bg + ' text-primary'">
              <i class="fa text-xl" :class="stat.icon"></i>
            </div>
            <span class="text-xs text-gray-medium">{{ formatSize(stat.size) }}</span>
          </div>
          <h3 class="font-medium mb-1">{{ stat.label }}</h3>
          <p class="text-sm text-gray-medium">{{ stat.count }} 个文件</p>
          <div class="w-full bg-gray-light rounded-full h-1 mt-3">
            <div class="h-1 rounded-full" :class="stat.color" :style="{ width: stat.percent + '%' }"></div>
          </div>
        </div>
      </div>

      <div>
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-medium">文件资源管理器</h2>
          <button class="btn-secondary" :disabled="path.length <= 1" @click="onNavigate(path.length - 2)">
            <i class="fa fa-level-up"></i>
            <span>返回上一级</span>
          </button>
        </div>
        <div v-if="(infiniteMode ? aggregated.length : sortedItems.length) === 0" class="bg-white rounded-xl shadow-card p-6 text-sm text-gray-medium">暂无内容</div>
        <FileList 
          v-else 
          :items="infiniteMode ? aggregated : sortedItems" 
          :view="view" 
          :sort-key="sortKey" 
          :sort-order="sortOrder" 
          @download="onDownload" 
          @rename="openRename" 
          @delete="openDelete" 
          @open="onOpenFolder" 
          @change-sort="onChangeSort" 
          @share="openShare" 
        />
        <div v-if="infiniteMode" ref="sentinelRef" class="h-8"></div>
        <div v-if="!infiniteMode && sortedItems.length > 0" class="flex flex-wrap items-center justify-between gap-3 mt-4">
          <div class="text-sm text-gray-medium">共 {{ total }} 项 · 第 {{ page }} / {{ pageCount }} 页</div>
          <div class="flex items-center gap-2">
            <button class="btn-secondary" :disabled="page <= 1" @click="goPrev">上一页</button>
            <button v-for="num in pageNumbers" :key="num" class="btn-icon-secondary w-9 h-9" :class="{ 'active-view': num === page }" @click="setPage(num)">{{ num }}</button>
            <button class="btn-secondary" :disabled="page >= pageCount" @click="goNext">下一页</button>
          </div>
        </div>
      </div>
    </template>

    <div v-if="createFolderOpen" class="fixed inset-0 z-50 bg-black bg-opacity-40 flex items-center justify-center">
      <div class="bg-white rounded-lg shadow-card w-full max-w-sm p-6">
        <div class="text-lg font-semibold text-gray-800 mb-4">新建文件夹</div>
        <input v-model="createFolderName" class="w-full border border-gray-300 rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500" placeholder="文件夹名称" />
        <div class="flex justify-end gap-2 mt-6">
          <button class="btn-secondary" :disabled="loading" @click="createFolderOpen = false">取消</button>
          <button class="btn-primary" :disabled="loading || !createFolderName.trim()" @click="submitCreateFolder">创建</button>
        </div>
      </div>
    </div>

    <div v-if="renameOpen" class="fixed inset-0 z-50 bg-black bg-opacity-40 flex items-center justify-center">
      <div class="bg-white rounded-lg shadow-card w-full max-w-sm p-6">
        <div class="text-lg font-semibold text-gray-800 mb-4">重命名</div>
        <input v-model="renameName" class="w-full border border-gray-300 rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500" placeholder="新名称" />
        <div class="flex justify-end gap-2 mt-6">
          <button class="btn-secondary" :disabled="loading" @click="renameOpen = false">取消</button>
          <button class="btn-primary" :disabled="loading || !renameName.trim()" @click="submitRename">确认</button>
        </div>
      </div>
    </div>

    <div v-if="deleteOpen" class="fixed inset-0 z-50 bg-black bg-opacity-40 flex items-center justify-center">
      <div class="bg-white rounded-lg shadow-card w-full max-w-sm p-6">
        <div class="text-lg font-semibold text-gray-800 mb-2">确认删除</div>
        <div class="text-sm text-gray-medium mb-4">删除后将进入回收站</div>
        <div class="flex justify-end gap-2">
          <button class="btn-secondary" :disabled="loading" @click="deleteOpen = false">取消</button>
          <button class="btn-primary" :disabled="loading" @click="submitDelete">删除</button>
        </div>
      </div>
    </div>

    <div v-if="shareOpen" class="fixed inset-0 z-50 bg-black bg-opacity-40 flex items-center justify-center">
      <div class="bg-white rounded-lg shadow-card w-full max-w-sm p-6">
        <div class="text-lg font-semibold text-gray-800 mb-4">创建分享</div>
        <div class="space-y-3">
          <div class="text-sm text-gray-medium">{{ shareTarget?.name || '' }}</div>
          <label class="block">
            <input v-model.number="shareExpired" type="number" min="0" placeholder="过期时间(秒)，0为永久或服务默认" class="w-full border border-gray-300 rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          </label>
          <p v-if="shareError" class="text-red-600 text-sm">{{ shareError }}</p>
          <div v-if="shareIdentity" class="space-y-2">
            <div class="text-sm">分享链接</div>
            <div class="flex items-center gap-2">
              <input class="flex-1 border border-gray-300 rounded px-3 py-2" :value="shareLink" readonly />
              <button class="btn-secondary" @click="copyShareLink">复制</button>
            </div>
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-6">
          <button class="btn-secondary" :disabled="shareLoading" @click="closeShare">关闭</button>
          <button class="btn-primary" :disabled="shareLoading || !shareTarget" @click="submitShare">{{ shareLoading ? '创建中...' : '创建分享' }}</button>
        </div>
      </div>
    </div>
  </main>
</template>
