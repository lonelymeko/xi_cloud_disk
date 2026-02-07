<script setup lang="ts">
import { onMounted } from 'vue'
import Chart from 'chart.js/auto'

onMounted(() => {
  const canvas = document.getElementById('storageChart') as HTMLCanvasElement | null
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  new Chart(ctx, {
    type: 'doughnut',
    data: {
      labels: ['视频', '图片', '压缩包', '文档'],
      datasets: [{
        data: [3.5, 1.2, 1.4, 0.85],
        backgroundColor: ['#0066cc', '#3b82f6', '#8b5cf6', '#22c55e'],
        borderWidth: 0,
        hoverOffset: 4,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      cutout: '70%',
      plugins: { legend: { display: false } },
    },
  })
})
</script>

<template>
  <aside class="w-80 bg-white border-l border-gray-light p-6 hidden lg:block">
    <div class="mb-6">
      <h2 class="text-lg font-medium mb-4">存储分析</h2>
      <div class="bg-gray-50 rounded-xl p-4 mb-4">
        <canvas id="storageChart" height="200"></canvas>
      </div>
      <div class="space-y-2">
        <div class="flex items-center justify-between text-sm">
          <div class="flex items-center gap-2">
            <span class="w-3 h-3 rounded-full bg-primary"></span>
            <span>视频</span>
          </div>
          <span class="font-medium">3.5GB</span>
        </div>
        <div class="flex items-center justify-between text-sm">
          <div class="flex items-center gap-2">
            <span class="w-3 h-3 rounded-full bg-blue-500"></span>
            <span>图片</span>
          </div>
          <span class="font-medium">1.2GB</span>
        </div>
        <div class="flex items-center justify-between text-sm">
          <div class="flex items-center gap-2">
            <span class="w-3 h-3 rounded-full bg-purple-500"></span>
            <span>压缩包</span>
          </div>
          <span class="font-medium">1.4GB</span>
        </div>
        <div class="flex items-center justify-between text-sm">
          <div class="flex items-center gap-2">
            <span class="w-3 h-3 rounded-full bg-green-500"></span>
            <span>文档</span>
          </div>
          <span class="font-medium">850MB</span>
        </div>
      </div>
    </div>
    <div class="mb-6">
      <h2 class="text-lg font-medium mb-4">快速访问</h2>
      <div class="space-y-3">
        <a href="#" class="flex items-center gap-3 p-3 rounded-lg hover:bg-gray-50">
          <div class="w-10 h-10 rounded-lg bg-primary bg-opacity-10 text-primary flex items-center justify-center">
            <i class="fa fa-star"></i>
          </div>
          <div>
            <h3 class="font-medium">收藏文件</h3>
            <p class="text-xs text-gray-medium">查看所有收藏的文件</p>
          </div>
        </a>
        <a href="#" class="flex items-center gap-3 p-3 rounded-lg hover:bg-gray-50">
          <div class="w-10 h-10 rounded-lg bg-green-500 bg-opacity-10 text-green-500 flex items-center justify-center">
            <i class="fa fa-clock-o"></i>
          </div>
          <div>
            <h3 class="font-medium">最近文件</h3>
            <p class="text-xs text-gray-medium">查看最近访问的文件</p>
          </div>
        </a>
        <a href="#" class="flex items-center gap-3 p-3 rounded-lg hover:bg-gray-50">
          <div class="w-10 h-10 rounded-lg bg-red-500 bg-opacity-10 text-red-500 flex items-center justify-center">
            <i class="fa fa-share-alt"></i>
          </div>
          <div>
            <h3 class="font-medium">已分享文件</h3>
            <p class="text-xs text-gray-medium">管理所有分享的文件</p>
          </div>
        </a>
      </div>
    </div>
    <div>
      <h2 class="text-lg font-medium mb-4">智能建议</h2>
      <div class="bg-blue-50 rounded-xl p-4 border border-blue-100">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-lg bg-primary bg-opacity-10 text-primary flex items-center justify-center">
            <i class="fa fa-lightbulb-o"></i>
          </div>
          <h3 class="font-medium">释放存储空间</h3>
        </div>
        <p class="text-sm text-gray-dark mb-3">您可以清理不必要的文件来释放存储空间。</p>
        <button class="btn-primary w-full text-sm">
          <i class="fa fa-trash"></i>
          <span>清理空间</span>
        </button>
      </div>
    </div>
  </aside>
</template>
