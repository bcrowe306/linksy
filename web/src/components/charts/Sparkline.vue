<template>
  <svg :viewBox="`0 0 ${w} ${h}`" :width="w" :height="h" class="overflow-visible" aria-hidden="true">
    <path v-if="path" :d="area" :fill="`rgb(var(${color}) / 0.1)`" />
    <path v-if="path" :d="path" fill="none" :stroke="`rgb(var(${color}))`" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
    <circle v-if="last" :cx="last[0]" :cy="last[1]" r="3" :fill="`rgb(var(${color}))`" stroke="rgb(var(--surface))" stroke-width="2" />
  </svg>
</template>

<script setup>
import { computed } from 'vue'
const props = defineProps({ values: { type: Array, default: () => [] }, color: { type: String, default: '--series-1' }, w: { type: Number, default: 96 }, h: { type: Number, default: 28 } })
const pts = computed(() => {
  const v = props.values
  if (v.length < 2) return []
  const max = Math.max(...v, 1)
  return v.map((y, i) => [(i / (v.length - 1)) * props.w, props.h - 3 - (y / max) * (props.h - 6)])
})
const path = computed(() => pts.value.map((p, i) => `${i ? 'L' : 'M'}${p[0].toFixed(1)},${p[1].toFixed(1)}`).join(''))
const area = computed(() => (pts.value.length ? `${path.value}L${props.w},${props.h}L0,${props.h}Z` : ''))
const last = computed(() => pts.value[pts.value.length - 1])
</script>
