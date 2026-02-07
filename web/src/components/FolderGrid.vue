<script setup lang="ts">
import type { UserFile } from '../lib/api'

const props = defineProps<{ folders: UserFile[] }>()
const emit = defineEmits<{
  (e: 'open', item: UserFile): void
  (e: 'rename', item: UserFile): void
  (e: 'delete', item: UserFile): void
}>()

function openFolder(item: UserFile) {
  emit('open', item)
}
</script>

<template>
  <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
    <div v-for="folder in props.folders" :key="folder.identity" class="bg-white rounded-xl shadow-card p-4 file-hover cursor-pointer">
      <div class="flex items-center justify-between mb-3">
        <div class="w-12 h-12 rounded-lg bg-blue-50 text-primary flex items-center justify-center" @click="openFolder(folder)">
          <i class="fa fa-folder text-2xl"></i>
        </div>
        <div class="relative group">
          <button class="btn-icon-secondary w-8 h-8">
            <i class="fa fa-ellipsis-v"></i>
          </button>
          <div class="absolute right-0 top-full mt-2 w-44 bg-white rounded-lg shadow-lg border border-gray-light opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200 z-10">
            <div class="py-1">
              <button class="w-full flex items-center gap-3 px-4 py-2 text-sm hover:bg-gray-50" @click="emit('rename', folder)">
                <i class="fa fa-pencil w-5 text-gray-medium"></i>
                <span>重命名</span>
              </button>
              <button class="w-full flex items-center gap-3 px-4 py-2 text-sm hover:bg-gray-50 text-red-500" @click="emit('delete', folder)">
                <i class="fa fa-trash w-5"></i>
                <span>删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
      <h3 class="font-medium truncate" @click="openFolder(folder)">{{ folder.name }}</h3>
      <p class="text-xs text-gray-medium mt-1">文件夹</p>
    </div>
  </div>
</template>
