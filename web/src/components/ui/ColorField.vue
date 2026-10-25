<template>
  <div>
    <label v-if="label" class="label">{{ label }} <Tip v-if="tip" :text="tip" /></label>
    <div class="flex items-center gap-2 rounded-xl border border-line bg-surface p-1.5 focus-within:border-brand-500 focus-within:ring-4 focus-within:ring-brand-500/15">
      <label class="relative h-7 w-7 shrink-0 cursor-pointer overflow-hidden rounded-lg border border-line" :style="{ background: modelValue || 'transparent' }">
        <input type="color" class="absolute inset-0 h-full w-full cursor-pointer opacity-0" :value="modelValue || fallback" :aria-label="label" @input="$emit('update:modelValue', $event.target.value)" />
        <span v-if="!modelValue" class="absolute inset-0 bg-[linear-gradient(135deg,transparent_45%,#ef4444_45%,#ef4444_55%,transparent_55%)]" />
      </label>
      <input
        :value="modelValue"
        class="min-w-0 flex-1 bg-transparent font-mono text-xs uppercase text-fg outline-none placeholder:normal-case placeholder:font-sans"
        :placeholder="placeholder"
        maxlength="7"
        @change="onText"
      />
      <button v-if="clearable && modelValue" type="button" class="text-fg-subtle hover:text-fg" aria-label="Reset color" @click="$emit('update:modelValue', '')">
        <X class="h-3.5 w-3.5" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { X } from 'lucide-vue-next'
import Tip from './Tip.vue'

defineProps({
  modelValue: { type: String, default: '' },
  label: String,
  tip: String,
  clearable: Boolean,
  placeholder: { type: String, default: '#000000' },
  fallback: { type: String, default: '#000000' }
})
const emit = defineEmits(['update:modelValue'])
function onText(e) {
  let v = e.target.value.trim()
  if (v && !v.startsWith('#')) v = '#' + v
  if (v === '' || /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.test(v)) emit('update:modelValue', v.toLowerCase())
  else e.target.value = ''
}
</script>
