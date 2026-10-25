<template>
  <div>
    <div class="mb-3 flex items-center gap-2 text-xs">
      <span class="relative flex h-2 w-2">
        <span v-if="status === 'live'" class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-500 opacity-60" />
        <span class="relative inline-flex h-2 w-2 rounded-full" :class="status === 'live' ? 'bg-emerald-500' : 'bg-fg-subtle'" />
      </span>
      <span class="text-fg-muted" data-testid="live-status">{{ status === 'live' ? 'Live' : status === 'connecting' ? 'Connecting…' : 'Reconnecting…' }}</span>
    </div>
    <p v-if="!events.length" class="py-8 text-center text-sm text-fg-muted">Waiting for the next click or scan… <br /><span class="text-xs text-fg-subtle">Activity appears here instantly.</span></p>
    <TransitionGroup v-else tag="ul" class="space-y-1" enter-active-class="transition duration-300" enter-from-class="opacity-0 -translate-y-1" data-testid="live-feed">
      <li v-for="e in events" :key="e.key" class="flex items-center gap-3 rounded-lg px-2 py-1.5 hover:bg-subtle/60">
        <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg" :class="e.type === 'scan' ? 'bg-[rgb(var(--series-2)/0.12)]' : 'bg-[rgb(var(--series-1)/0.12)]'">
          <ScanLine v-if="e.type === 'scan'" class="h-3.5 w-3.5" :style="{ color: 'rgb(var(--series-2))' }" />
          <MousePointerClick v-else class="h-3.5 w-3.5" :style="{ color: 'rgb(var(--series-1))' }" />
        </span>
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm text-fg">{{ e.asset || 'Untitled' }}</p>
          <p class="truncate text-xs text-fg-subtle">{{ flag(e.country) }} {{ [e.city, e.country].filter(Boolean).join(', ') || 'Unknown location' }} · {{ e.device }} · {{ e.browser }}</p>
        </div>
        <span class="shrink-0 text-[11px] text-fg-subtle">{{ relativeTime(e.at) }}</span>
      </li>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { ScanLine, MousePointerClick } from 'lucide-vue-next'
import { flag } from '../../lib/countries'
import { relativeTime } from '../../lib/format'
import { useAuthStore } from '../../stores/auth'

const props = defineProps({ max: { type: Number, default: 8 } })
const emit = defineEmits(['event'])
const auth = useAuthStore()
const events = ref([])
const status = ref('connecting')
let ws = null
let retry = 0
let timer = null
let closed = false
let seq = 0

function connect() {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  ws = new WebSocket(`${proto}://${location.host}/api/v1/analytics/live`)
  ws.onmessage = (m) => {
    const data = JSON.parse(m.data)
    if (data.type === 'ready') {
      status.value = 'live'
      retry = 0
      return
    }
    if (data.type === 'click' || data.type === 'scan') {
      events.value = [{ ...data, key: ++seq }, ...events.value].slice(0, props.max)
      emit('event', data)
    }
  }
  ws.onclose = () => {
    if (closed) return
    status.value = 'reconnecting'
    timer = setTimeout(connect, Math.min(30000, 1000 * 2 ** retry++))
  }
}

function reset() {
  closed = true
  clearTimeout(timer)
  ws?.close()
  events.value = []
  closed = false
  retry = 0
  status.value = 'connecting'
  connect()
}

// Re-render relative times every 15s.
let tick
onMounted(() => {
  connect()
  tick = setInterval(() => (events.value = [...events.value]), 15000)
})
watch(() => auth.team?.id, reset)
onBeforeUnmount(() => {
  closed = true
  clearTimeout(timer)
  clearInterval(tick)
  ws?.close()
})
</script>
