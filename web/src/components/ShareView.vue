<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getShare, getShareUrl, saveShare, getUserFileList, type ShareDetailResp } from '../lib/api'
import { getToken } from '../lib/auth'

const shareIdentity = (() => {
  const path = location.pathname || ''
  const idx = path.indexOf('/s/')
  if (idx >= 0) return path.slice(idx + 3)
  return ''
})()

const loading = ref(false)
const error = ref('')
const detail = ref<ShareDetailResp | null>(null)
const expires = ref(3600)
const url = ref('')
const parentId = ref(0)
const name = ref('')
const saveLoading = ref(false)
const saveError = ref('')
const folders = ref<{ id: number; name: string }[]>([])

async function loadDetail() {
  loading.value = true
  error.value = ''
  try {
    const d = await getShare(shareIdentity)
    detail.value = d
    name.value = d.name
  } catch (e: any) {
    error.value = e?.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function loadUrl() {
  if (!detail.value) return
  loading.value = true
  error.value = ''
  try {
    const data = await getShareUrl(shareIdentity, Math.max(0, Number(expires.value) || 0))
    url.value = data.url
  } catch (e: any) {
    error.value = e?.message || '获取链接失败'
  } finally {
    loading.value = false
  }
}

async function loadFolders() {
  const token = getToken()
  if (!token) return
  try {
    const data = await getUserFileList(0, 1, 200, token)
    const rows = (data.list || []).filter((it) => !it.repository_identity).map((it) => ({ id: it.id, name: it.name }))
    folders.value = [{ id: 0, name: '根目录' }, ...rows]
  } catch {}
}

async function onSave() {
  if (!detail.value) return
  const token = getToken()
  if (!token) {
    saveError.value = '请登录后保存'
    return
  }
  saveLoading.value = true
  saveError.value = ''
  try {
    await saveShare(detail.value.repository_identity, Number(parentId.value) || 0, name.value.trim() || detail.value.name, token)
  } catch (e: any) {
    saveError.value = e?.message || '保存失败'
  } finally {
    saveLoading.value = false
  }
}

function openDownload() {
  const link = url.value
  if (!link) return
  if (typeof window !== 'undefined') window.open(link, '_blank')
}

onMounted(async () => {
  await loadDetail()
  await loadFolders()
})
</script>

<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white shadow-sm h-16 flex items-center px-6">
      <div class="flex items-center gap-2">
        <i class="fa fa-share-alt text-primary text-2xl"></i>
        <h1 class="text-xl font-bold">分享查看</h1>
      </div>
    </header>
    <main class="container mx-auto p-6">
      <div v-if="error" class="text-red-600 text-sm mb-4">{{ error }}</div>
      <div v-else-if="loading" class="text-sm text-gray-medium mb-4">加载中...</div>
      <div v-else-if="detail" class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div class="lg:col-span-2 space-y-6">
          <div class="bg-white rounded-xl shadow-card p-6">
            <h2 class="text-lg font-medium mb-4">分享内容</h2>
            <div class="space-y-2 text-sm">
              <div class="flex items-center justify-between"><span class="text-gray-medium">名称</span><span class="font-medium">{{ detail.name }}</span></div>
              <div class="flex items-center justify-between"><span class="text-gray-medium">类型</span><span class="font-medium">{{ detail.ext || '-' }}</span></div>
              <div class="flex items-center justify-between"><span class="text-gray-medium">大小</span><span class="font-medium">{{ detail.size }}</span></div>
            </div>
          </div>
          <div class="bg-white rounded-xl shadow-card p-6">
            <h2 class="text-lg font-medium mb-4">下载</h2>
            <div class="flex items-center gap-2 mb-3">
              <input v-model.number="expires" type="number" min="0" class="border border-gray-light rounded-lg px-3 py-2 w-40" />
              <button class="btn-secondary" @click="loadUrl">获取链接</button>
            </div>
            <div class="flex items-center gap-2">
              <input class="flex-1 border border-gray-300 rounded px-3 py-2" :value="url" readonly />
              <button class="btn-primary" :disabled="!url" @click="openDownload">下载</button>
            </div>
          </div>
        </div>
        <div class="space-y-6">
          <div class="bg-white rounded-xl shadow-card p-6">
            <h2 class="text-lg font-medium mb-4">保存到我的网盘</h2>
            <div class="space-y-3">
              <select v-model.number="parentId" class="border border-gray-light rounded-lg px-3 py-2 w-full">
                <option v-for="f in folders" :key="f.id" :value="f.id">{{ f.name }}</option>
              </select>
              <input v-model="name" class="border border-gray-light rounded-lg px-3 py-2 w-full" />
              <button class="btn-primary w-full" :disabled="saveLoading" @click="onSave">{{ saveLoading ? '保存中...' : '保存' }}</button>
              <p v-if="saveError" class="text-red-600 text-sm">{{ saveError }}</p>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>
