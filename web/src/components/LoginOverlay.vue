<script setup lang="ts">
import { ref, onUnmounted } from 'vue'
import { login, register, resetPassword, sendVerificationCode } from '../lib/api'
import { setToken } from '../lib/auth'

const emit = defineEmits<{ (e: 'logged-in'): void }>()
const mode = ref<'login' | 'register' | 'forgot'>('login')
const name = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const code = ref('')
const loading = ref(false)
const error = ref('')
const success = ref('')
const cooldown = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

function setMode(next: 'login' | 'register' | 'forgot') {
  mode.value = next
  error.value = ''
  success.value = ''
}

function startCooldown() {
  cooldown.value = 60
  if (timer) clearInterval(timer)
  timer = setInterval(() => {
    if (cooldown.value <= 1) {
      cooldown.value = 0
      if (timer) clearInterval(timer)
      timer = null
      return
    }
    cooldown.value -= 1
  }, 1000)
}

async function onSendCode() {
  error.value = ''
  success.value = ''
  if (!email.value.trim()) {
    error.value = '请输入邮箱'
    return
  }
  loading.value = true
  try {
    await sendVerificationCode(email.value.trim())
    startCooldown()
  } catch (e: any) {
    const msg = e?.message || '发送失败'
    if (msg.includes('Failed to fetch') || msg.includes('NetworkError')) {
      error.value = '网络异常或跨域配置问题'
    } else if (msg.startsWith('HTTP ')) {
      error.value = '请求失败，请检查服务状态'
    } else {
      error.value = msg
    }
  } finally {
    loading.value = false
  }
}

async function submit() {
  error.value = ''
  success.value = ''
  loading.value = true
  try {
    if (mode.value === 'login') {
      const data = await login(name.value.trim(), password.value)
      setToken(data.token)
      emit('logged-in')
    } else if (mode.value === 'register') {
      const data = await register(name.value.trim(), email.value.trim(), password.value, code.value.trim())
      setToken(data.token)
      emit('logged-in')
    } else {
      if (!email.value.trim() || !code.value.trim() || !password.value) {
        error.value = '请完整填写信息'
        return
      }
      if (password.value !== confirmPassword.value) {
        error.value = '两次密码不一致'
        return
      }
      await resetPassword(email.value.trim(), code.value.trim(), password.value)
      setMode('login')
      success.value = '密码已重置，请登录'
    }
  } catch (e: any) {
    const msg = e?.message || '操作失败'
    if (msg.includes('Failed to fetch') || msg.includes('NetworkError')) {
      error.value = '网络异常或跨域配置问题'
    } else if (msg.startsWith('HTTP ')) {
      error.value = '请求失败，请检查服务状态'
    } else {
      error.value = msg
    }
  } finally {
    loading.value = false
  }
}

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="fixed inset-0 z-50 bg-white">
    <div class="h-full w-full flex items-center justify-center">
      <div class="w-full max-w-sm p-6">
        <div class="flex items-center gap-4 mb-6">
          <template v-if="mode !== 'forgot'">
            <button class="text-lg font-semibold" :class="mode === 'login' ? 'text-gray-800' : 'text-gray-medium'" @click="setMode('login')">登录</button>
            <button class="text-lg font-semibold" :class="mode === 'register' ? 'text-gray-800' : 'text-gray-medium'" @click="setMode('register')">注册</button>
          </template>
          <template v-else>
            <button class="text-lg font-semibold text-gray-800" @click="setMode('forgot')">忘记密码</button>
            <button class="text-sm text-gray-medium" @click="setMode('login')">返回登录</button>
          </template>
        </div>
        <form class="space-y-3" @submit.prevent="submit" :autocomplete="mode === 'login' ? 'on' : 'off'">
          <label v-if="mode !== 'forgot'" class="block">
            <span class="sr-only">用户名</span>
            <input
              v-model="name"
              name="username"
              class="w-full border border-gray-300 rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="用户名"
              autocomplete="username"
              inputmode="email"
            />
          </label>
          <label v-if="mode !== 'login'" class="block">
            <span class="sr-only">邮箱</span>
            <input
              v-model="email"
              name="email"
              class="w-full border border-gray-300 rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="邮箱"
              autocomplete="email"
              inputmode="email"
            />
          </label>
          <label class="block">
            <span class="sr-only">密码</span>
            <input
              v-model="password"
              name="password"
              class="w-full border border-gray-300 rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              type="password"
              :placeholder="mode === 'forgot' ? '新密码' : '密码'"
              :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
            />
          </label>
          <label v-if="mode === 'forgot'" class="block">
            <span class="sr-only">确认新密码</span>
            <input
              v-model="confirmPassword"
              name="confirmPassword"
              class="w-full border border-gray-300 rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              type="password"
              placeholder="确认新密码"
              autocomplete="new-password"
            />
          </label>
          <div v-if="mode !== 'login'" class="space-y-2">
            <div class="flex gap-2">
              <label class="flex-1">
                <span class="sr-only">验证码</span>
                <input
                  v-model="code"
                  name="one-time-code"
                  class="w-full border border-gray-300 rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="验证码"
                  autocomplete="one-time-code"
                  inputmode="numeric"
                />
              </label>
              <button
                type="button"
                class="px-3 py-2 rounded border border-gray-300 text-gray-700 hover:bg-gray-50 disabled:opacity-60"
                :disabled="loading || cooldown > 0"
                @click="onSendCode"
              >
                {{ cooldown > 0 ? `${cooldown}s` : '获取验证码' }}
              </button>
            </div>
            <p class="text-xs text-gray-medium">本地开发请查看后端控制台验证码</p>
          </div>
          <button
            type="submit"
            class="w-full bg-blue-600 text-white rounded px-3 py-2 hover:bg-blue-700 disabled:opacity-60"
            :disabled="loading || (mode === 'login' && (!name || !password)) || (mode === 'register' && (!name || !email || !password || !code)) || (mode === 'forgot' && (!email || !password || !code || !confirmPassword))"
          >
            {{
              loading
                ? (mode === 'login' ? '登录中...' : mode === 'register' ? '注册中...' : '提交中...')
                : (mode === 'login' ? '登录' : mode === 'register' ? '注册' : '确认重置')
            }}
          </button>
          <button v-if="mode === 'login'" type="button" class="text-sm text-gray-medium hover:text-gray-700" @click="setMode('forgot')">忘记密码？</button>
          <p v-if="error" class="text-red-600 text-sm">{{ error }}</p>
          <p v-if="success" class="text-green-600 text-sm">{{ success }}</p>
        </form>
      </div>
    </div>
  </div>
</template>
