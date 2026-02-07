<script setup lang="ts">
const props = defineProps<{ view: 'detail' | 'medium' | 'large'; sortKey: 'name' | 'type' | 'size' | 'updated'; sortOrder: 'asc' | 'desc'; loading: boolean }>()
const emit = defineEmits<{
  (e: 'open-upload'): void
  (e: 'create-folder'): void
  (e: 'change-view', value: 'detail' | 'medium' | 'large'): void
  (e: 'change-sort-key', value: 'name' | 'type' | 'size' | 'updated'): void
  (e: 'change-sort-order', value: 'asc' | 'desc'): void
}>()

function setView(value: 'detail' | 'medium' | 'large') {
  emit('change-view', value)
}

function onSortChange(e: Event) {
  const target = e.target as HTMLSelectElement
  emit('change-sort-key', target.value as 'name' | 'type' | 'size' | 'updated')
}

function toggleSortOrder() {
  emit('change-sort-order', props.sortOrder === 'asc' ? 'desc' : 'asc')
}
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-4 mb-6">
    <div class="flex items-center gap-3">
      <button class="btn-primary" :disabled="props.loading" @click="emit('open-upload')">
        <i class="fa fa-upload"></i>
        <span>上传文件</span>
      </button>
      <button class="btn-secondary" :disabled="props.loading" @click="emit('create-folder')">
        <i class="fa fa-folder"></i>
        <span>新建文件夹</span>
      </button>
      <button class="btn-icon-secondary" :disabled="props.loading">
        <i class="fa fa-ellipsis-h"></i>
      </button>
    </div>
    <div class="flex items-center gap-3">
      <div class="flex items-center gap-2">
        <button class="btn-icon-secondary" :class="{ 'active-view': props.view === 'detail' }" @click="setView('detail')">
          <i class="fa fa-list"></i>
        </button>
        <button class="btn-icon-secondary" :class="{ 'active-view': props.view === 'medium' }" @click="setView('medium')">
          <i class="fa fa-th-large"></i>
        </button>
        <button class="btn-icon-secondary" :class="{ 'active-view': props.view === 'large' }" @click="setView('large')">
          <i class="fa fa-th"></i>
        </button>
      </div>
      <select class="border border-gray-light rounded-lg px-3 py-2 focus:outline-none focus:border-primary" :value="props.sortKey" @change="onSortChange">
        <option value="updated">修改日期</option>
        <option value="name">文件名</option>
        <option value="type">类型</option>
        <option value="size">大小</option>
      </select>
      <button class="btn-icon-secondary" @click="toggleSortOrder">
        <i class="fa" :class="props.sortOrder === 'asc' ? 'fa-sort-amount-asc' : 'fa-sort-amount-desc'"></i>
      </button>
    </div>
  </div>
</template>
