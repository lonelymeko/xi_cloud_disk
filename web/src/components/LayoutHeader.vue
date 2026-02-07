<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{ userName: string }>()
const emit = defineEmits<{ (e: 'search', value: string): void; (e: 'logout'): void; (e: 'open-profile'): void }>()
function onSearch(e: Event) {
  const target = e.target as HTMLInputElement
  emit('search', target.value)
}

const userShort = computed(() => {
  const name = props.userName || ''
  if (!name) return '--'
  return name.length > 2 ? name.slice(0, 2) : name
})

const isLoggedIn = computed(() => !!props.userName)

const menuOpen = ref(false)
const confirmOpen = ref(false)

function toggleMenu() {
  if (!isLoggedIn.value) return
  menuOpen.value = !menuOpen.value
}

function requestLogout() {
  if (!isLoggedIn.value) return
  menuOpen.value = false
  confirmOpen.value = true
}

function openProfile() {
  if (!isLoggedIn.value) return
  menuOpen.value = false
  emit('open-profile')
}

function confirmLogout() {
  if (!isLoggedIn.value) return
  confirmOpen.value = false
  emit('logout')
}

function cancelLogout() {
  confirmOpen.value = false
}
</script>

<template>
  <header class="bg-white shadow-sm fixed top-0 left-0 right-0 z-50 h-16">
    <div class="container mx-auto px-4 h-full flex items-center justify-between">
      <div class="flex items-center gap-2">
        <div class="text-primary text-2xl">
          <i class="fa fa-cloud"></i>
        </div>
        <h1 class="text-xl font-bold">CloudDrive</h1>
      </div>
      <div v-if="isLoggedIn" class="flex items-center gap-4">
        <div class="relative">
          <input type="text" placeholder="搜索文件..." class="pl-10 pr-4 py-2 rounded-lg border border-gray-light focus:outline-none focus:border-primary w-64" @input="onSearch">
          <i class="fa fa-search absolute left-3 top-3 text-gray-medium"></i>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn-icon-secondary relative">
            <i class="fa fa-bell"></i>
            <span class="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full"></span>
          </button>
          <div class="flex items-center gap-2">
            <div class="w-8 h-8 rounded-full bg-primary text-white flex items-center justify-center">
              <span>{{ userShort }}</span>
            </div>
            <span class="font-medium hidden md:inline">{{ userName || '未登录' }}</span>
            <button class="text-gray-medium" @click="toggleMenu">
              <i class="fa fa-angle-down"></i>
            </button>
            <div v-if="menuOpen" class="absolute right-0 top-12 w-40 bg-white border border-gray-light rounded-lg shadow-card p-2">
              <button class="w-full text-left px-3 py-2 rounded hover:bg-gray-50 text-gray-dark" @click="openProfile">个人中心</button>
              <button class="w-full text-left px-3 py-2 rounded hover:bg-gray-50 text-gray-dark" @click="requestLogout">退出登录</button>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="flex items-center gap-2 text-gray-medium">
        <div class="w-8 h-8 rounded-full bg-gray-light flex items-center justify-center">
          <i class="fa fa-user"></i>
        </div>
        <span class="text-sm">未登录</span>
      </div>
    </div>
    <div v-if="confirmOpen && isLoggedIn" class="fixed inset-0 z-50 bg-black bg-opacity-40 flex items-center justify-center">
      <div class="bg-white rounded-lg shadow-card w-full max-w-sm p-6">
        <div class="text-lg font-semibold text-gray-800 mb-2">确认退出</div>
        <div class="text-sm text-gray-medium mb-4">确定要退出当前账号吗？</div>
        <div class="flex justify-end gap-2">
          <button class="btn-secondary" @click="cancelLogout">取消</button>
          <button class="btn-primary" @click="confirmLogout">退出</button>
        </div>
      </div>
    </div>
  </header>
</template>
