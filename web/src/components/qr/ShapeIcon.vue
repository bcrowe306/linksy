<template>
  <!-- Miniature previews of module, eye and frame shapes -->
  <svg viewBox="0 0 24 24" class="h-7 w-7" aria-hidden="true">
    <template v-if="kind === 'dot'">
      <g fill="currentColor">
        <template v-if="shape === 'dots'"><circle v-for="p in grid" :key="p.join()" :cx="p[0] + 2.5" :cy="p[1] + 2.5" r="2.2" /></template>
        <template v-else-if="shape === 'rounded'"><rect v-for="p in grid" :key="p.join()" :x="p[0] + 0.3" :y="p[1] + 0.3" width="4.4" height="4.4" rx="1.4" /></template>
        <template v-else-if="shape === 'extra-rounded'"><path d="M4 2h6a2 2 0 0 1 0 4H8v6a2 2 0 0 1-4 0V4a2 2 0 0 1 0-2zM14 14h4a2 2 0 0 1 2 2v4a2 2 0 0 1-4 0v-2h-2a2 2 0 0 1 0-4z" /></template>
        <template v-else-if="shape === 'classy'"><path d="M6 2h4v4H4V4a2 2 0 0 1 2-2zM4 6h4v6H6a2 2 0 0 1-2-2zM14 14h6v4a2 2 0 0 1-2 2h-4z" /></template>
        <template v-else><rect v-for="p in grid" :key="p.join()" :x="p[0]" :y="p[1]" width="5" height="5" /></template>
      </g>
    </template>
    <template v-else-if="kind === 'eye'">
      <path :d="eyePath" fill="currentColor" fill-rule="evenodd" />
    </template>
    <template v-else-if="kind === 'inner'">
      <circle v-if="shape === 'circle'" cx="12" cy="12" r="7" fill="currentColor" />
      <rect v-else-if="shape === 'rounded'" x="5" y="5" width="14" height="14" rx="3.5" fill="currentColor" />
      <path v-else-if="shape === 'leaf'" d="M12 5h7v7a7 7 0 0 1-7 7H5v-7a7 7 0 0 1 7-7z" fill="currentColor" />
      <rect v-else x="5" y="5" width="14" height="14" fill="currentColor" />
    </template>
    <template v-else-if="kind === 'frame'">
      <g fill="none" stroke="currentColor" stroke-width="1.6">
        <rect v-if="shape !== 'none'" x="3" :y="shape === 'banner-top' ? 7 : 3" width="18" height="14" rx="2.5" />
        <rect x="7" :y="shape === 'banner-top' ? 10 : 6" width="10" height="8" rx="1" stroke-dasharray="2 1.5" />
      </g>
      <rect v-if="shape === 'banner-bottom'" x="3" y="17" width="18" height="4" rx="1.5" fill="currentColor" />
      <rect v-if="shape === 'banner-top'" x="3" y="3" width="18" height="4" rx="1.5" fill="currentColor" />
      <rect v-if="shape === 'pill'" x="7" y="18.5" width="10" height="4" rx="2" fill="currentColor" />
      <rect v-if="shape === 'card'" x="7" y="18" width="10" height="1.6" rx=".8" fill="currentColor" />
      <path v-if="shape === 'none'" d="M4 4l16 16" stroke="currentColor" stroke-width="1.6" />
    </template>
  </svg>
</template>

<script setup>
import { computed } from 'vue'
const props = defineProps({ kind: String, shape: String })
const grid = [[2, 2], [7, 2], [2, 7], [12, 12], [17, 12], [17, 17], [12, 17]]
const eyePath = computed(() => {
  const ring = {
    square: 'M3 3h18v18H3zM6 6v12h12V6z',
    rounded: 'M8 3h8a5 5 0 0 1 5 5v8a5 5 0 0 1-5 5H8a5 5 0 0 1-5-5V8a5 5 0 0 1 5-5zM8 6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2z',
    'extra-rounded': 'M10 3h4a7 7 0 0 1 7 7v4a7 7 0 0 1-7 7h-4a7 7 0 0 1-7-7v-4a7 7 0 0 1 7-7zM10 6a4 4 0 0 0-4 4v4a4 4 0 0 0 4 4h4a4 4 0 0 0 4-4v-4a4 4 0 0 0-4-4z',
    circle: 'M12 3a9 9 0 1 1 0 18 9 9 0 0 1 0-18zM12 6a6 6 0 1 0 0 12 6 6 0 0 0 0-12z',
    leaf: 'M12 3h9v9a9 9 0 0 1-9 9H3v-9a9 9 0 0 1 9-9zM12 6a6 6 0 0 0-6 6v6h6a6 6 0 0 0 6-6V6z'
  }
  return ring[props.shape] || ring.square
})
</script>
