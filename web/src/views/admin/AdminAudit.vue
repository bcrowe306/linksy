<template>
  <section class="card overflow-hidden">
    <header class="flex flex-col gap-3 border-b border-line px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h2 class="text-sm font-semibold text-fg">Platform audit log</h2>
        <p class="text-xs text-fg-muted">Every action taken in this console. Entries can't be edited or deleted.</p>
      </div>
      <div class="relative sm:w-72">
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-subtle" />
        <input v-model="q" type="search" placeholder="Filter by action, admin or target" class="input input-sm pl-9" data-testid="admin-audit-search" />
      </div>
    </header>
    <div v-if="loading && !events.length" class="space-y-3 p-5"><div v-for="i in 6" :key="i" class="skeleton h-10" /></div>
    <EmptyState v-else-if="!events.length" :icon="ScrollText" title="No entries" />
    <ol v-else class="divide-y divide-line" data-testid="admin-audit-list">
      <li v-for="e in events" :key="e.id" class="flex gap-3 px-5 py-3">
        <span class="mt-0.5 shrink-0 rounded-md bg-subtle px-2 py-0.5 font-mono text-[11px] text-fg-muted">{{ e.action }}</span>
        <div class="min-w-0 flex-1">
          <p class="text-sm text-fg">
            <span class="font-medium">{{ e.actor_email || e.actor_id }}</span>
            <span v-if="e.target_label || e.target_id" class="text-fg-muted"> → {{ e.target_label || e.target_id }}</span>
          </p>
          <p v-if="details(e)" class="truncate text-xs text-fg-muted" :title="details(e)">{{ details(e) }}</p>
          <p class="text-xs text-fg-subtle"><time :datetime="e.created_at">{{ new Date(e.created_at).toLocaleString() }}</time><span v-if="e.ip"> · {{ e.ip }}</span></p>
        </div>
      </li>
    </ol>
    <div v-if="hasMore" class="border-t border-line p-3 text-center"><button class="btn-ghost btn-sm" :disabled="loading" @click="load(true)">Load more</button></div>
  </section>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { Search, ScrollText } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useAdminCall } from '../../composables/useAdmin'
import EmptyState from '../../components/ui/EmptyState.vue'

const call = useAdminCall()
const q = ref('')
const events = ref([])
const loading = ref(false)
const hasMore = ref(false)

async function load(append = false) {
  loading.value = true
  try {
    const res = await call(() => api.admin.audit(q.value.trim(), append ? events.value.length : 0))
    events.value = append ? [...events.value, ...res.events] : res.events
    hasMore.value = res.events.length === 50
  } finally {
    loading.value = false
  }
}
let timer
watch(q, () => {
  clearTimeout(timer)
  timer = setTimeout(() => load(), 250)
})
onMounted(() => load())

function details(e) {
  try {
    const m = JSON.parse(e.metadata || '{}')
    return Object.entries(m)
      .filter(([, v]) => v !== null && v !== '' && v !== undefined)
      .map(([k, v]) => `${k.replace(/_/g, ' ')}: ${typeof v === 'object' ? JSON.stringify(v) : v}`)
      .join(' · ')
  } catch {
    return ''
  }
}
</script>
