<template>
  <div class="flex items-center gap-2.5">
    <svg :class="sizeClass" viewBox="0 0 32 32" aria-hidden="true">
      <defs>
        <linearGradient id="lynqr-logo-g" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stop-color="#6366f1" /><stop offset="1" stop-color="#a855f7" />
        </linearGradient>
      </defs>
      <rect width="32" height="32" rx="8" fill="url(#lynqr-logo-g)" />
      <path fill="#fff" d="M8 8h6v6H8zM18 8h6v6h-6zM8 18h6v6H8z" opacity=".95" />
      <path fill="#fff" d="M18 18h2.5v2.5H18zM21.5 21.5H24V24h-2.5zM18 21.5h2.5V24H18zM21.5 18H24v2.5h-2.5z" opacity=".75" />
    </svg>
    <span v-if="!iconOnly" class="font-extrabold tracking-tight text-fg" :class="textClass">Lynqr</span>
  </div>
</template>

<script setup>
import { computed } from 'vue'
const props = defineProps({ size: { type: String, default: 'md' }, iconOnly: Boolean })
const sizeClass = computed(() => ({ sm: 'w-6 h-6', md: 'w-8 h-8', lg: 'w-10 h-10' })[props.size])
const textClass = computed(() => ({ sm: 'text-base', md: 'text-lg', lg: 'text-2xl' })[props.size])
</script>
