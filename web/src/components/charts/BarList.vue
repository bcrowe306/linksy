<template>
  <div :class="{ 'opacity-60 transition-opacity': loading }">
    <p v-if="!rows.length" class="py-8 text-center text-sm text-fg-muted">{{ empty }}</p>
    <ul v-else class="space-y-1">
      <li
        v-for="r in shown"
        :key="r.name"
        class="group relative flex items-center gap-3 rounded-lg px-2 py-1.5 hover:bg-subtle/60"
        tabindex="0"
        :title="`${label(r.name)}: ${r.count.toLocaleString()} (${pct(r.count)}%)`"
      >
        <span class="absolute inset-y-1 left-0 rounded-md transition-all" :style="{ width: `${(r.count / max) * 100}%`, background: `rgb(var(${color}) / 0.14)` }" />
        <span class="relative flex min-w-0 flex-1 items-center gap-2 text-sm text-fg">
          <slot name="icon" :row="r" />
          <span class="truncate">{{ label(r.name) }}</span>
        </span>
        <span class="relative text-xs text-fg-subtle tabular-nums">{{ pct(r.count) }}%</span>
        <span class="relative w-12 text-right text-sm font-semibold tabular-nums text-fg">{{ compact(r.count) }}</span>
      </li>
    </ul>
    <button v-if="rows.length > limit" class="btn-ghost btn-sm mt-1 w-full" @click="expanded = !expanded">
      {{ expanded ? 'Show less' : `Show all ${rows.length}` }}
    </button>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { compact } from '../../lib/format'

const props = defineProps({
  rows: { type: Array, default: () => [] }, // [{ name, count }]
  color: { type: String, default: '--series-1' },
  limit: { type: Number, default: 6 },
  labeler: { type: Function, default: (n) => n },
  empty: { type: String, default: 'No data for this period yet.' },
  loading: Boolean
})
const expanded = ref(false)
const max = computed(() => Math.max(1, ...props.rows.map((r) => r.count)))
const sum = computed(() => props.rows.reduce((a, r) => a + r.count, 0) || 1)
const shown = computed(() => (expanded.value ? props.rows : props.rows.slice(0, props.limit)))
const pct = (n) => Math.round((n / sum.value) * 100)
const label = (n) => props.labeler(n)
</script>
