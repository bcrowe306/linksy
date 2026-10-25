<template>
  <div class="w-full" :class="{ 'opacity-60 transition-opacity': loading }">
    <!-- Legend (always present for 2+ series) -->
    <div v-if="series.length > 1" class="mb-3 flex flex-wrap gap-4 text-xs text-fg-muted" role="list">
      <span v-for="s in series" :key="s.key" class="flex items-center gap-1.5" role="listitem">
        <span class="h-0.5 w-4 rounded-full" :style="{ background: `rgb(var(${s.color}))` }" />{{ s.label }}
        <span class="font-semibold text-fg tabular-nums">{{ compact(total(s.key)) }}</span>
      </span>
    </div>

    <div ref="wrap" class="relative" :style="{ height: `${height}px` }" @pointermove="onMove" @pointerleave="hover = -1">
      <svg :width="width" :height="height" class="block" role="img" :aria-label="ariaLabel">
        <!-- grid + y ticks -->
        <g v-for="t in ticks" :key="t">
          <line :x1="padL" :x2="width - padR" :y1="y(t)" :y2="y(t)" stroke="rgb(var(--chart-grid))" stroke-width="1" />
          <text :x="padL - 8" :y="y(t) + 4" text-anchor="end" class="fill-fg-subtle text-[11px] tabular-nums">{{ compact(t) }}</text>
        </g>
        <!-- x labels -->
        <text v-for="i in xLabelIdx" :key="`x${i}`" :x="x(i)" :y="height - 6" :text-anchor="i === 0 ? 'start' : i === points.length - 1 ? 'end' : 'middle'" class="fill-fg-subtle text-[11px]">{{ bucketLabel(points[i].t, granularity) }}</text>
        <!-- series -->
        <g v-for="s in series" :key="s.key">
          <path v-if="series.length === 1" :d="areaPath(s.key)" :fill="`rgb(var(${s.color}) / 0.1)`" />
          <path :d="linePath(s.key, completeCount)" fill="none" :stroke="`rgb(var(${s.color}))`" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />
          <!-- The current period is still in progress: draw it faded so a partial day doesn't read as a drop -->
          <path v-if="partial" :d="partialPath(s.key)" fill="none" :stroke="`rgb(var(${s.color}) / 0.35)`" stroke-width="2" stroke-linecap="round" />
        </g>
        <!-- crosshair -->
        <g v-if="hover >= 0">
          <line :x1="x(hover)" :x2="x(hover)" :y1="padT" :y2="height - padB" stroke="rgb(var(--chart-axis))" stroke-width="1" />
          <circle v-for="s in series" :key="s.key" :cx="x(hover)" :cy="y(points[hover][s.key])" r="4" :fill="`rgb(var(${s.color}))`" stroke="rgb(var(--surface))" stroke-width="2" />
        </g>
      </svg>

      <div
        v-if="hover >= 0"
        class="pointer-events-none absolute top-2 z-10 min-w-[150px] rounded-xl border border-line bg-elevated px-3 py-2 text-xs shadow-pop"
        :style="{ left: `${Math.min(Math.max(x(hover) + 12, 0), width - 170)}px` }"
        role="tooltip"
      >
        <p class="mb-1.5 text-fg-muted">{{ tooltipDate(points[hover].t, granularity) }}<span v-if="partial && hover === points.length - 1"> (so far)</span></p>
        <p v-for="s in series" :key="s.key" class="flex items-center justify-between gap-4">
          <span class="flex items-center gap-1.5 text-fg-muted"><span class="h-0.5 w-3 rounded-full" :style="{ background: `rgb(var(${s.color}))` }" />{{ s.label }}</span>
          <span class="font-semibold tabular-nums text-fg">{{ points[hover][s.key].toLocaleString() }}</span>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { compact, niceTicks, bucketLabel, tooltipDate } from '../../lib/format'

const props = defineProps({
  points: { type: Array, default: () => [] }, // [{ t, clicks, scans }]
  series: { type: Array, required: true }, // [{ key, label, color }]
  granularity: { type: String, default: 'day' },
  height: { type: Number, default: 260 },
  loading: Boolean
})

const wrap = ref(null)
const width = ref(600)
const hover = ref(-1)
const padL = 44, padR = 12, padT = 8, padB = 26

let ro
onMounted(() => {
  ro = new ResizeObserver(([e]) => (width.value = Math.max(240, e.contentRect.width)))
  ro.observe(wrap.value)
})
onBeforeUnmount(() => ro?.disconnect())

const maxV = computed(() => Math.max(1, ...props.points.flatMap((p) => props.series.map((s) => p[s.key]))))
const ticks = computed(() => niceTicks(maxV.value))
const top = computed(() => ticks.value[ticks.value.length - 1] || 1)
const x = (i) => padL + (props.points.length <= 1 ? 0 : (i / (props.points.length - 1)) * (width.value - padL - padR))
const y = (v) => padT + (1 - v / top.value) * (props.height - padT - padB)

const linePath = (key, n = props.points.length) =>
  props.points.slice(0, n).map((p, i) => `${i ? 'L' : 'M'}${x(i).toFixed(1)},${y(p[key]).toFixed(1)}`).join('')

// The last bucket is partial when "now" falls inside it.
const bucketMs = { hour: 3600e3, day: 86400e3, week: 7 * 86400e3 }
const partial = computed(() => {
  const n = props.points.length
  if (n < 2) return false
  const last = new Date(props.points[n - 1].t).getTime()
  return Date.now() < last + bucketMs[props.granularity]
})
const completeCount = computed(() => (partial.value ? props.points.length - 1 : props.points.length))
function partialPath(key) {
  const n = props.points.length
  return `M${x(n - 2).toFixed(1)},${y(props.points[n - 2][key]).toFixed(1)}L${x(n - 1).toFixed(1)},${y(props.points[n - 1][key]).toFixed(1)}`
}
const areaPath = (key) => (props.points.length ? `${linePath(key)}L${x(props.points.length - 1)},${y(0)}L${x(0)},${y(0)}Z` : '')
const total = (key) => props.points.reduce((a, p) => a + p[key], 0)

const xLabelIdx = computed(() => {
  const n = props.points.length
  if (!n) return []
  const max = Math.max(2, Math.floor(width.value / 90))
  const step = Math.max(1, Math.ceil(n / max))
  const idx = []
  for (let i = 0; i < n; i += step) idx.push(i)
  // Always label the last bucket, dropping a neighbor that would collide with it.
  if (idx[idx.length - 1] !== n - 1) {
    if (n - 1 - idx[idx.length - 1] < step / 2) idx.pop()
    idx.push(n - 1)
  }
  return idx
})

function onMove(e) {
  if (!props.points.length) return
  const rect = wrap.value.getBoundingClientRect()
  const rel = (e.clientX - rect.left - padL) / (width.value - padL - padR)
  hover.value = Math.min(props.points.length - 1, Math.max(0, Math.round(rel * (props.points.length - 1))))
}

const ariaLabel = computed(() => `${props.series.map((s) => `${s.label}: ${total(s.key)}`).join(', ')} over ${props.points.length} periods`)
</script>
