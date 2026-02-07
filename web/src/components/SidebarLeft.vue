<script setup lang="ts">
const props = defineProps<{ active: string }>()
const emit = defineEmits<{ (e: 'select', value: string): void }>()
function isActive(name: string) { return props.active === name }
function setActive(name: string) { emit('select', name) }
</script>

<template>
  <aside class="w-64 bg-white border-r border-gray-light p-4 flex flex-col fixed top-16 left-0 bottom-0 overflow-y-auto">
    <nav class="flex-1 space-y-1">
      <a href="#" class="nav-item" :class="{ active: isActive('首页') }" @click.prevent="setActive('首页')">
        <i class="fa fa-home w-5"></i>
        <span>首页</span>
      </a>
      <a href="#" class="nav-item" :class="{ active: isActive('文件资源管理器') }" @click.prevent="setActive('文件资源管理器')">
        <i class="fa fa-folder-open w-5"></i>
        <span>文件资源管理器</span>
      </a>
      <a href="#" class="nav-item" :class="{ active: isActive('个人中心') }" @click.prevent="setActive('个人中心')">
        <i class="fa fa-user w-5"></i>
        <span>个人中心</span>
      </a>
      <a href="#" class="nav-item" :class="{ active: isActive('收藏') }" @click.prevent="setActive('收藏')">
        <i class="fa fa-star w-5"></i>
        <span>收藏</span>
      </a>
      <a href="#" class="nav-item" :class="{ active: isActive('最近访问') }" @click.prevent="setActive('最近访问')">
        <i class="fa fa-clock-o w-5"></i>
        <span>最近访问</span>
      </a>
      <a href="#" class="nav-item" :class="{ active: isActive('已分享') }" @click.prevent="setActive('已分享')">
        <i class="fa fa-share-alt w-5"></i>
        <span>已分享</span>
      </a>
      <a href="#" class="nav-item" :class="{ active: isActive('回收站') }" @click.prevent="setActive('回收站')">
        <i class="fa fa-trash w-5"></i>
        <span>回收站</span>
      </a>
      <div class="pt-4 mt-4 border-t border-gray-light">
        <div class="px-4 py-2 text-xs font-medium text-gray-medium uppercase">文件分类</div>
        <a href="#" class="nav-item" :class="{ active: isActive('图片') }" @click.prevent="setActive('图片')">
          <i class="fa fa-file-image-o w-5"></i>
          <span>图片</span>
        </a>
        <a href="#" class="nav-item" :class="{ active: isActive('视频') }" @click.prevent="setActive('视频')">
          <i class="fa fa-file-video-o w-5"></i>
          <span>视频</span>
        </a>
        <a href="#" class="nav-item" :class="{ active: isActive('音频') }" @click.prevent="setActive('音频')">
          <i class="fa fa-file-audio-o w-5"></i>
          <span>音频</span>
        </a>
        <a href="#" class="nav-item" :class="{ active: isActive('文档') }" @click.prevent="setActive('文档')">
          <i class="fa fa-file-text-o w-5"></i>
          <span>文档</span>
        </a>
        <a href="#" class="nav-item" :class="{ active: isActive('压缩包') }" @click.prevent="setActive('压缩包')">
          <i class="fa fa-file-archive-o w-5"></i>
          <span>压缩包</span>
        </a>
      </div>
    </nav>
  </aside>
</template>
