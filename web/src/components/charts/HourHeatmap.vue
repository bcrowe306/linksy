<template>
  <div :class="{ 'opacity-60 transition-opacity': loading }">
    <div class="overflow-x-auto">
      <div class="grid w-full gap-[2px]" style="grid-template-columns: 30px repeat(24, minmax(6px, 1fr))" role="grid" aria-label="Engagement by day of week and hour">
        <span />
        <span v-for="h in 24" :key="`h${h}`" class="text-center text-[10px] text-fg-subtle">{{ (h - 1) % 6 === 0 ? hourLabel(h - 1) : '' }}</span>
        <template v-for="d in order" :key="d">
          <span class="pr-1 text-right text-[11px] leading-[18px] text-fg-subtle">{{ days[d] }}</span>
          <span
            v-for="h in 24"
            :key="`${d}-${h}`"
            class="h-[18px] rounded-[3px] outline-none ring-brand-500 hover:ring-1 focus:ring-2"
            :style="{ background: cell(matrix[d][h - 1]) }"
            :title="`${daysLong[d]} ${hourLabel(h - 1)}: ${matrix[d][h - 1].toLocaleString()}`"
            tabindex="0"
            role="gridcell"
          />
        </template>
      </div>
    </div>
    <div class="mt-3 flex items-center gap-2 text-[11px] text-fg-subtle">
      <span>Less</span>
      <span v-for="i in 7" :key="i" class="h-2.5 w-4 rounded-sm" :style="{ background: `rgb(var(--seq-${i - 1}))` }" />
      <span>More</span>
      <span v-if="peak" class="ml-auto">Peak: <span class="font-medium text-fg">{{ peak }}</span></span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
const props = defineProps({ matrix: { type: Array, default: () => Array.from({ length: 7 }, () => Array(24).fill(0)) }, loading: Boolean })
const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
const daysLong = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
const order = [1, 2, 3, 4, 5, 6, 0] // Monday first
const max = computed(() => Math.max(1, ...props.matrix.flat()))
const hourLabel = (h) => new Date(2000, 0, 1, h).toLocaleTimeString(undefined, { hour: 'numeric' })
function cell(n) {
  if (!n) return 'rgb(var(--seq-0))'
  return `rgb(var(--seq-${Math.min(6, 1 + Math.floor((n / max.value) * 5.999))}))`
}
const peak = computed(() => {
  let best = [0, 0, 0]
  props.matrix.forEach((row, d) => row.forEach((n, h) => n > best[0] && (best = [n, d, h])))
  return best[0] ? `${daysLong[best[1]]} ${hourLabel(best[2])}` : ''
})
</script>
