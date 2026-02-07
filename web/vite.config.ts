import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const target = env.VITE_API_PROXY_TARGET || 'http://172.26.175.210:8888'
  return {
    plugins: [vue()],
    server: {
      proxy: {
        '/api': {
          target,
          changeOrigin: true,
        },
      },
    },
  }
})
