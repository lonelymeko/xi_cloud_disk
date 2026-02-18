<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import LayoutHeader from './components/LayoutHeader.vue'
import SidebarLeft from './components/SidebarLeft.vue'
import MainArea from './components/MainArea.vue'
import UploadModal from './components/UploadModal.vue'
import LoginOverlay from './components/LoginOverlay.vue'
import ShareView from './components/ShareView.vue'
import { getToken, clearToken, getTokenPayload } from './lib/auth'
import { authProbe, getUserDetail } from './lib/api'

const showUpload = ref(false)
const loggedIn = ref(false)
const userName = ref('')
const userEmail = ref('')
const activeNav = ref('文件资源管理器')
const searchText = ref('')
const uploadParentId = ref(0)
const uploadSignal = ref(0)
const shareRoute = ref(false)

async function loadUserFromToken(token: string) {
  const payload = getTokenPayload(token)
  const identity = payload?.Identity || payload?.identity
  if (!identity) {
    userName.value = payload?.Name || payload?.name || ''
    userEmail.value = ''
    return
  }
  try {
    const data = await getUserDetail(identity)
    userName.value = data.name
    userEmail.value = data.email
  } catch {
    userName.value = payload?.Name || payload?.name || ''
    userEmail.value = ''
  }
}

async function onLoggedIn() {
  loggedIn.value = true
  const t = getToken()
  if (t) await loadUserFromToken(t)
}

function onLogout() {
  clearToken()
  loggedIn.value = false
  userName.value = ''
  userEmail.value = ''
  showUpload.value = false
  activeNav.value = '文件资源管理器'
  searchText.value = ''
}

function onSelectNav(value: string) {
  activeNav.value = value
}

function onSearch(value: string) {
  searchText.value = value
}

function onOpenUpload(parentId: number) {
  uploadParentId.value = parentId
  showUpload.value = true
}

function onUploaded() {
  uploadSignal.value += 1
}

function onOpenProfile() {
  activeNav.value = '个人中心'
}

function loadNavMemory() {
  const saved = localStorage.getItem('nav_active')
  if (saved) activeNav.value = saved
}

watch(activeNav, (value) => {
  localStorage.setItem('nav_active', value)
})

onMounted(async () => {
  loadNavMemory()
  shareRoute.value = (location.pathname || '').startsWith('/s/')
  const t = getToken()
  if (!t) {
    loggedIn.value = false
    return
  }
  const ok = await authProbe(t)
  if (!ok) clearToken()
  loggedIn.value = ok
  if (ok) await loadUserFromToken(t)
})
</script>

<template>
  <div class="bg-gray-50 font-sans text-gray-dark min-h-screen">
    <template v-if="shareRoute">
      <ShareView />
    </template>
    <template v-else>
      <LoginOverlay v-if="!loggedIn" @logged-in="onLoggedIn" />
      <template v-if="loggedIn">
        <LayoutHeader :user-name="userName" @search="onSearch" @logout="onLogout" @open-profile="onOpenProfile" />
        <SidebarLeft :active="activeNav" @select="onSelectNav" />
        <div class="pt-16 pl-64 h-[calc(100vh-4rem)]">
          <MainArea :key="`${activeNav}-${uploadSignal}`" :active="activeNav" :user-name="userName" :user-email="userEmail" :search="searchText" :refresh-key="uploadSignal" @open-upload="onOpenUpload" @refresh-user="onLoggedIn" @logout="onLogout" />
        </div>
        <UploadModal :visible="showUpload" :parent-id="uploadParentId" @close="showUpload=false" @uploaded="onUploaded" />
      </template>
    </template>
  </div>
</template>
