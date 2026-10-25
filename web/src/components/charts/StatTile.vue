<template>
  <div class="card p-5" :class="{ 'opacity-60 transition-opacity': loading }">
    <div class="flex items-start justify-between gap-3">
      <p class="flex items-center gap-1.5 text-sm text-fg-muted">
        {{ label }}
        <Tip v-if="tip" :text="tip" />
      </p>
      <component :is="icon" v-if="icon" class="h-4 w-4 text-fg-subtle" />
    </div>
    <div class="mt-2 flex items-end justify-between gap-3">
      <p class="text-3xl font-semibold tracking-tight text-fg" :data-testid="testid">{{ compact(value) }}</p>
      <Sparkline v-if="trend?.length > 1" :values="trend" :color="color" class="mb-1 shrink-0" />
    </div>
    <p v-if="previous !== undefined" class="mt-1.5 flex flex-wrap items-center gap-x-1 text-xs">
      <span class="inline-flex items-center gap-0.5 font-medium" :class="deltaClass">
        <TrendingUp v-if="d.direction === 'up'" class="h-3.5 w-3.5" />
        <TrendingDown v-else-if="d.direction === 'down'" class="h-3.5 w-3.5" />
        <Minus v-else class="h-3.5 w-3.5" />
        {{ d.label }}
      </span>
      <span class="text-fg-subtle">vs previous period</span>
    </p>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { TrendingUp, TrendingDown, Minus } from 'lucide-vue-next'
import { compact, delta } from '../../lib/format'
import Sparkline from './Sparkline.vue'
import Tip from '../ui/Tip.vue'

const props = defineProps({
  label: String,
  value: { type: Number, default: 0 },
  previous: { type: Number, default: undefined },
  trend: Array,
  color: { type: String, default: '--series-1' },
  icon: [Object, Function],
  tip: String,
  loading: Boolean,
  testid: String
})
const d = computed(() => delta(props.value, props.previous))
// More engagement is good: up = success ink, down = muted (not alarming red).
const deltaClass = computed(() => (d.value.direction === 'up' ? 'text-emerald-600 dark:text-emerald-400' : d.value.direction === 'down' ? 'text-rose-600 dark:text-rose-400' : 'text-fg-subtle'))
</script>
