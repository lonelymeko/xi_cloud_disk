<script setup lang="ts">
type Crumb = { id: number; name: string }
const props = defineProps<{ path: Crumb[] }>()
const emit = defineEmits<{ (e: 'navigate', index: number): void }>()
function onNavigate(index: number) {
  emit('navigate', index)
}
</script>

<template>
  <div class="flex items-center gap-2 text-sm mb-6">
    <a href="#" class="text-primary" @click.prevent="onNavigate(0)">首页</a>
    <template v-for="(seg, idx) in props.path" :key="seg.id">
      <i class="fa fa-angle-right text-gray-medium text-xs"></i>
      <a
        v-if="idx < props.path.length - 1"
        href="#"
        class="text-primary"
        @click.prevent="onNavigate(idx)"
      >
        {{ seg.name }}
      </a>
      <span v-else class="text-gray-dark">{{ seg.name }}</span>
    </template>
  </div>
</template>
