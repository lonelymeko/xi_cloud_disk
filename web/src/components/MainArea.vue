<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import Chart from 'chart.js/auto'
import { changePassword, getUserFileList, type UserFile } from '../lib/api'
import { getToken } from '../lib/auth'
import FileWorkspace from './FileWorkspace.vue'
const props = defineProps<{ active: string; userName: string; userEmail: string; search: string; refreshKey: number }>()
const emit = defineEmits<{ (e: 'open-upload', parentId: number): void; (e: 'refresh-user'): void; (e: 'logout'): void }>()
const passwordOpen = ref(false)
const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordLoading = ref(false)
const passwordError = ref('')

const homeSource = ref<'mock' | 'api'>('mock')
const homeLoading = ref(false)
const homeError = ref('')
const homeList = ref<UserFile[]>([])
const homeTotal = ref(0)
const doughnutRef = ref<HTMLCanvasElement | null>(null)
const barRef = ref<HTMLCanvasElement | null>(null)
const doughnutContainerRef = ref<HTMLDivElement | null>(null)
const barContainerRef = ref<HTMLDivElement | null>(null)
const doughnutChart = ref<Chart | null>(null)
const barChart = ref<Chart | null>(null)
const homeReady = ref(false)
const resizeObserver = ref<ResizeObserver | null>(null)

function observeContainers() {
  if (!resizeObserver.value) return
  if (doughnutContainerRef.value) resizeObserver.value.observe(doughnutContainerRef.value)
  if (barContainerRef.value) resizeObserver.value.observe(barContainerRef.value)
}

const mockFiles: UserFile[] = [
  { id: 1, identity: 'mock-1', name: '产品手册.pdf', ext: '.pdf', size: 1862400, repository_identity: 'repo-1', updated_at: '2026-02-01 10:24:00' },
  { id: 2, identity: 'mock-2', name: '年终总结.pptx', ext: '.pptx', size: 8243200, repository_identity: 'repo-2', updated_at: '2026-02-02 09:18:00' },
  { id: 3, identity: 'mock-3', name: '产品视频.mp4', ext: '.mp4', size: 582432000, repository_identity: 'repo-3', updated_at: '2026-02-03 19:45:00' },
  { id: 4, identity: 'mock-4', name: '会议录音.mp3', ext: '.mp3', size: 28432000, repository_identity: 'repo-4', updated_at: '2026-01-28 08:12:00' },
  { id: 5, identity: 'mock-5', name: '项目截图.png', ext: '.png', size: 1240000, repository_identity: 'repo-5', updated_at: '2026-01-26 16:33:00' },
  { id: 6, identity: 'mock-6', name: '设计稿.psd', ext: '.psd', size: 82432000, repository_identity: 'repo-6', updated_at: '2026-01-31 14:20:00' },
  { id: 7, identity: 'mock-7', name: '预算表.xlsx', ext: '.xlsx', size: 1520000, repository_identity: 'repo-7', updated_at: '2026-01-29 11:02:00' },
  { id: 8, identity: 'mock-8', name: '照片合集.zip', ext: '.zip', size: 84243200, repository_identity: 'repo-8', updated_at: '2026-02-01 21:30:00' },
  { id: 9, identity: 'mock-9', name: '发布说明.txt', ext: '.txt', size: 124000, repository_identity: 'repo-9', updated_at: '2026-01-25 09:40:00' },
  { id: 10, identity: 'mock-10', name: '销售数据.csv', ext: '.csv', size: 342000, repository_identity: 'repo-10', updated_at: '2026-02-04 13:05:00' },
]


function openPassword() {
  passwordOpen.value = true
  passwordError.value = ''
  oldPassword.value = ''
  newPassword.value = ''
  confirmPassword.value = ''
}

function closePassword() {
  if (passwordLoading.value) return
  passwordOpen.value = false
}

function validatePassword() {
  if (!oldPassword.value || !newPassword.value || !confirmPassword.value) {
    passwordError.value = '请完整填写密码信息'
    return false
  }
  if (newPassword.value !== confirmPassword.value) {
    passwordError.value = '两次新密码不一致'
    return false
  }
  if (oldPassword.value === newPassword.value) {
    passwordError.value = '新密码不能与旧密码相同'
    return false
  }
  return true
}

async function submitPassword() {
  passwordError.value = ''
  if (!validatePassword()) return
  const token = getToken()
  if (!token) {
    passwordError.value = '登录已失效，请重新登录'
    return
  }
  passwordLoading.value = true
  try {
    await changePassword('', oldPassword.value, newPassword.value, token)
    emit('logout')
    closePassword()
  } catch (e: any) {
    const msg = e?.message || '修改密码失败'
    if (msg.includes('Failed to fetch') || msg.includes('NetworkError')) {
      passwordError.value = '网络异常或跨域配置问题'
    } else if (msg.startsWith('HTTP ')) {
      passwordError.value = '请求失败，请检查服务状态'
    } else {
      passwordError.value = msg
    }
  } finally {
    passwordLoading.value = false
  }
}

function isFolder(item: UserFile) {
  return !item.repository_identity
}

function formatSize(size: number) {
  if (!size || size <= 0) return '0B'
  if (size < 1024) return `${size}B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)}KB`
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)}MB`
  return `${(size / 1024 / 1024 / 1024).toFixed(1)}GB`
}

const fileTypes = computed(() => {
  const buckets = [
    { key: 'image', label: '图片', exts: ['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp'], color: 'bg-blue-500', hex: '#3b82f6' },
    { key: 'video', label: '视频', exts: ['.mp4', '.avi', '.mov', '.mkv', '.flv', '.wmv', '.webm', '.m4v'], color: 'bg-red-500', hex: '#ef4444' },
    { key: 'audio', label: '音频', exts: ['.mp3', '.wav', '.aac', '.flac', '.ogg', '.m4a'], color: 'bg-indigo-500', hex: '#6366f1' },
    { key: 'doc', label: '文档', exts: ['.pdf', '.doc', '.docx', '.xls', '.xlsx', '.ppt', '.pptx', '.txt', '.md'], color: 'bg-green-500', hex: '#22c55e' },
    { key: 'zip', label: '压缩包', exts: ['.zip', '.rar', '.7z', '.tar', '.gz'], color: 'bg-purple-500', hex: '#a855f7' },
    { key: 'other', label: '其他', exts: [], color: 'bg-gray-400', hex: '#9ca3af' },
  ]
  const rows = buckets.map((bucket) => ({ ...bucket, size: 0, count: 0 }))
  for (const item of homeList.value) {
    if (isFolder(item)) continue
    const ext = item.ext?.toLowerCase() || ''
    const target = rows.find((row) => row.exts.includes(ext)) || rows.find((row) => row.key === 'other')
    if (!target) continue
    target.count += 1
    target.size += item.size || 0
  }
  return rows
})

const totalSize = computed(() => fileTypes.value.reduce((sum, item) => sum + item.size, 0))

function buildCharts() {
  if (props.active !== '首页') return
  if (!homeReady.value) return
  if (doughnutChart.value) {
    doughnutChart.value.stop()
    doughnutChart.value.destroy()
    doughnutChart.value = null
  }
  if (barChart.value) {
    barChart.value.stop()
    barChart.value.destroy()
    barChart.value = null
  }
  const doughnutCanvas = doughnutRef.value
  const barCanvas = barRef.value
  if (!doughnutCanvas || !barCanvas) return
  if (doughnutCanvas) {
    const ctx = doughnutCanvas.getContext('2d')
    if (ctx) {
      doughnutChart.value = new Chart(ctx, {
        type: 'doughnut',
        data: {
          labels: fileTypes.value.map((item) => item.label),
          datasets: [{
            data: fileTypes.value.map((item) => item.size),
            backgroundColor: fileTypes.value.map((item) => item.hex),
            borderWidth: 0,
            hoverOffset: 4,
          }],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          cutout: '70%',
          animation: false,
          plugins: { legend: { display: false } },
        },
      })
    }
  }
  if (barCanvas) {
    const ctx = barCanvas.getContext('2d')
    if (ctx) {
      barChart.value = new Chart(ctx, {
        type: 'bar',
        data: {
          labels: fileTypes.value.map((item) => item.label),
          datasets: [{
            data: fileTypes.value.map((item) => item.count),
            backgroundColor: fileTypes.value.map((item) => item.hex),
            borderRadius: 8,
          }],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          animation: false,
          scales: {
            y: { beginAtZero: true, ticks: { precision: 0 } },
            x: { grid: { display: false } },
          },
          plugins: { legend: { display: false } },
        },
      })
    }
  }
}

async function loadHomeData() {
  if (props.active !== '首页') return
  homeError.value = ''
  homeLoading.value = true
  try {
    if (homeSource.value === 'mock') {
      homeList.value = mockFiles
      homeTotal.value = mockFiles.length
    } else {
      const token = getToken()
      if (!token) {
        homeError.value = '登录已失效，请重新登录'
        return
      }
      const data = await getUserFileList(0, 1, 200, token)
      homeList.value = data.list || []
      homeTotal.value = data.count || 0
    }
    await nextTick()
    homeReady.value = true
    buildCharts()
  } catch (e: any) {
    homeError.value = e?.message || '加载失败'
  } finally {
    homeLoading.value = false
  }
}

watch(() => props.active, async () => {
  if (props.active === '首页') {
    await nextTick()
    observeContainers()
    loadHomeData()
  } else {
    homeReady.value = false
    if (doughnutChart.value) {
      doughnutChart.value.stop()
      doughnutChart.value.destroy()
      doughnutChart.value = null
    }
    if (barChart.value) {
      barChart.value.stop()
      barChart.value.destroy()
      barChart.value = null
    }
  }
})

watch(homeSource, () => {
  if (props.active === '首页') loadHomeData()
})

watch(() => fileTypes.value.map((item) => `${item.key}:${item.count}:${item.size}`).join('|'), async () => {
  if (props.active !== '首页') return
  await nextTick()
  homeReady.value = true
  buildCharts()
})

onMounted(() => {
  if (props.active === '首页') loadHomeData()
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver.value = new ResizeObserver(() => {
      if (!homeReady.value) return
      doughnutChart.value?.resize()
      barChart.value?.resize()
    })
    observeContainers()
  }
})

onUnmounted(() => {
  if (doughnutChart.value) doughnutChart.value.destroy()
  if (barChart.value) barChart.value.destroy()
  if (resizeObserver.value) resizeObserver.value.disconnect()
})
</script>

<template>
  <main class="h-full overflow-y-auto p-6">
    <div v-if="props.active === '个人中心'">
      <div class="flex items-center gap-2 text-sm mb-6">
        <a href="#" class="text-primary">首页</a>
        <i class="fa fa-angle-right text-gray-medium text-xs"></i>
        <span class="text-gray-dark">个人中心</span>
      </div>
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div class="lg:col-span-2 space-y-6">
          <div class="bg-white rounded-xl shadow-card p-6">
            <div class="flex items-center justify-between mb-4">
              <h2 class="text-lg font-medium">基础信息</h2>
              <button class="btn-secondary" @click="emit('refresh-user')">刷新</button>
            </div>
            <div class="flex items-center gap-4">
              <div class="w-14 h-14 rounded-full bg-primary text-white flex items-center justify-center text-lg">
                <span>{{ props.userName ? props.userName.slice(0, 2) : '--' }}</span>
              </div>
              <div>
                <div class="text-lg font-semibold">{{ props.userName || '未登录' }}</div>
                <div class="text-sm text-gray-medium">{{ props.userEmail || '未绑定邮箱' }}</div>
              </div>
            </div>
          </div>
          <div class="bg-white rounded-xl shadow-card p-6">
            <h2 class="text-lg font-medium mb-4">账号安全</h2>
            <div class="flex items-center justify-between">
              <div>
                <div class="font-medium">密码</div>
                <div class="text-sm text-gray-medium">建议定期更新密码</div>
              </div>
              <button class="btn-secondary" @click="openPassword">修改密码</button>
            </div>
          </div>
        </div>
        <div class="space-y-6">
          <div class="bg-white rounded-xl shadow-card p-6">
            <h2 class="text-lg font-medium mb-4">注册邮箱</h2>
            <div class="text-sm text-gray-medium mb-3">用于找回密码和安全验证</div>
            <div class="font-medium">{{ props.userEmail || '未绑定邮箱' }}</div>
          </div>
          <div class="bg-white rounded-xl shadow-card p-6">
            <h2 class="text-lg font-medium mb-4">账号操作</h2>
            <div class="text-sm text-gray-medium mb-3">退出登录请使用右上角用户菜单</div>
          </div>
        </div>
      </div>
      <div v-if="passwordOpen" class="fixed inset-0 z-50 bg-black bg-opacity-40 flex items-center justify-center">
        <div class="bg-white rounded-lg shadow-card w-full max-w-md p-6">
          <div class="text-lg font-semibold text-gray-800 mb-4">修改密码</div>
          <div class="space-y-3">
            <label class="block">
              <span class="sr-only">旧密码</span>
              <input v-model="oldPassword" type="password" placeholder="旧密码" class="w-full border border-gray-300 rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500" autocomplete="current-password" />
            </label>
            <label class="block">
              <span class="sr-only">新密码</span>
              <input v-model="newPassword" type="password" placeholder="新密码" class="w-full border border-gray-300 rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500" autocomplete="new-password" />
            </label>
            <label class="block">
              <span class="sr-only">确认新密码</span>
              <input v-model="confirmPassword" type="password" placeholder="确认新密码" class="w-full border border-gray-300 rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500" autocomplete="new-password" />
            </label>
            <p v-if="passwordError" class="text-red-600 text-sm">{{ passwordError }}</p>
          </div>
          <div class="flex justify-end gap-2 mt-6">
            <button class="btn-secondary" :disabled="passwordLoading" @click="closePassword">取消</button>
            <button class="btn-primary" :disabled="passwordLoading" @click="submitPassword">{{ passwordLoading ? '提交中...' : '确认修改' }}</button>
          </div>
        </div>
      </div>
    </div>
    <div v-else-if="props.active === '首页'">
      <div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between mb-6">
        <div>
          <h2 class="text-xl font-semibold">数据看板</h2>
          <p class="text-sm text-gray-medium">基于文件列表的类型统计与占比分析</p>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn-secondary" :class="{ 'active-view': homeSource === 'mock' }" @click="homeSource = 'mock'">模拟数据</button>
          <button class="btn-secondary" :class="{ 'active-view': homeSource === 'api' }" @click="homeSource = 'api'">真实数据</button>
        </div>
      </div>
      <div v-if="homeError" class="mb-4 text-sm text-red-500">{{ homeError }}</div>
      <div v-else-if="homeLoading" class="mb-4 text-sm text-gray-medium">加载中...</div>
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div class="lg:col-span-2 space-y-6">
          <div class="bg-white rounded-xl shadow-card p-6">
            <div class="flex items-center justify-between mb-4">
              <h3 class="text-lg font-medium">文件类型占比</h3>
              <span class="text-sm text-gray-medium">总容量 {{ formatSize(totalSize) }}</span>
            </div>
            <div ref="doughnutContainerRef" class="h-64 w-full relative">
              <canvas ref="doughnutRef" class="w-full h-full"></canvas>
            </div>
            <div class="grid grid-cols-2 md:grid-cols-3 gap-3 mt-4">
              <div v-for="item in fileTypes" :key="item.key" class="flex items-center justify-between text-sm bg-gray-50 rounded-lg px-3 py-2">
                <div class="flex items-center gap-2">
                  <span class="w-3 h-3 rounded-full" :class="item.color"></span>
                  <span>{{ item.label }}</span>
                </div>
                <span class="font-medium">{{ formatSize(item.size) }}</span>
              </div>
            </div>
          </div>
          <div class="bg-white rounded-xl shadow-card p-6">
            <div class="flex items-center justify-between mb-4">
              <h3 class="text-lg font-medium">文件类型数量统计</h3>
              <span class="text-sm text-gray-medium">{{ homeTotal }} 项</span>
            </div>
            <div ref="barContainerRef" class="h-64 w-full relative">
              <canvas ref="barRef" class="w-full h-full"></canvas>
            </div>
          </div>
        </div>
        <div class="space-y-6">
          <div class="bg-white rounded-xl shadow-card p-6">
            <h3 class="text-lg font-medium mb-4">数据概览</h3>
            <div class="space-y-3 text-sm">
              <div class="flex items-center justify-between">
                <span class="text-gray-medium">文件总数</span>
                <span class="font-medium">{{ homeTotal }}</span>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-gray-medium">总容量</span>
                <span class="font-medium">{{ formatSize(totalSize) }}</span>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-gray-medium">数据源</span>
                <span class="font-medium">{{ homeSource === 'mock' ? '模拟数据' : '真实接口' }}</span>
              </div>
            </div>
          </div>
          <div class="bg-white rounded-xl shadow-card p-6">
            <h3 class="text-lg font-medium mb-4">文件类型分布</h3>
            <div class="space-y-3">
              <div v-for="item in fileTypes" :key="item.key" class="flex items-center justify-between text-sm">
                <div class="flex items-center gap-2">
                  <span class="w-3 h-3 rounded-full" :class="item.color"></span>
                  <span>{{ item.label }}</span>
                </div>
                <span class="font-medium">{{ item.count }} 个</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <template v-else>
      <FileWorkspace :active="props.active" :search="props.search" :refresh-key="props.refreshKey" @open-upload="emit('open-upload', $event)" />
    </template>
  </main>
</template>
