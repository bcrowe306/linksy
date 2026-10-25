<template>
  <label
    class="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-2xl border-2 border-dashed px-6 py-8 text-center transition-colors"
    :class="dragging ? 'border-brand-500 bg-brand-500/5' : 'border-line hover:border-line-strong hover:bg-subtle/40'"
    @dragover.prevent="dragging = true"
    @dragleave.prevent="dragging = false"
    @drop.prevent="onDrop"
  >
    <input type="file" class="sr-only" :accept="accept" data-testid="file-input" @change="onPick" />
    <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-500/10 text-brand-500">
      <FileCheck2 v-if="modelValue" class="h-5 w-5" />
      <UploadCloud v-else class="h-5 w-5" />
    </div>
    <template v-if="modelValue">
      <p class="text-sm font-medium text-fg">{{ modelValue.name }}</p>
      <p class="text-xs text-fg-muted">{{ formatBytes(modelValue.size) }} · click to choose a different file</p>
    </template>
    <template v-else>
      <p class="text-sm text-fg"><span class="font-semibold text-brand-500">Click to upload</span> or drag and drop</p>
      <p class="text-xs text-fg-muted">PDF, Office documents or images · up to {{ maxMb }} MB</p>
    </template>
    <p v-if="error" class="text-xs text-rose-500">{{ error }}</p>
  </label>
</template>

<script setup>
import { ref } from 'vue'
import { UploadCloud, FileCheck2 } from 'lucide-vue-next'

const props = defineProps({
  modelValue: { type: Object, default: null },
  accept: { type: String, default: '.pdf,.doc,.docx,.ppt,.pptx,.xls,.xlsx,.csv,.txt,.png,.jpg,.jpeg,.gif,.webp' },
  maxMb: { type: Number, default: 15 }
})
const emit = defineEmits(['update:modelValue'])
const dragging = ref(false)
const error = ref('')

function set(file) {
  error.value = ''
  if (!file) return
  if (file.size > props.maxMb * 1024 * 1024) {
    error.value = `File is larger than ${props.maxMb} MB`
    return
  }
  emit('update:modelValue', file)
}
function onPick(e) {
  set(e.target.files?.[0])
}
function onDrop(e) {
  dragging.value = false
  set(e.dataTransfer.files?.[0])
}
function formatBytes(n) {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}
</script>
