<template>
  <section class="card overflow-hidden">
    <header class="flex flex-col gap-3 border-b border-line px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h2 class="text-sm font-semibold text-fg">Audit log</h2>
        <p class="text-xs text-fg-muted">A permanent record of changes in this workspace.</p>
      </div>
      <div class="flex gap-2">
        <select v-model="category" class="input input-sm w-40" aria-label="Filter by category">
          <option value="">All activity</option>
          <option value="member">Members</option>
          <option value="link">Links</option>
          <option value="qr">QR codes</option>
          <option value="document">Documents</option>
          <option value="api_key">API keys</option>
          <option value="team">Workspace</option>
        </select>
        <button class="btn-secondary btn-sm" @click="exportCSV"><Download class="h-3.5 w-3.5" /> CSV</button>
      </div>
    </header>

    <div v-if="loading && !events.length" class="space-y-3 p-5"><div v-for="i in 5" :key="i" class="skeleton h-10" /></div>
    <EmptyState v-else-if="!visible.length" :icon="ScrollText" title="No activity yet" />
    <ol v-else class="divide-y divide-line" data-testid="audit-list">
      <li v-for="e in visible" :key="e.id" class="flex gap-3 px-5 py-3">
        <div class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-subtle text-fg-muted">
          <component :is="iconFor(e.action)" class="h-4 w-4" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm text-fg">
            <span class="font-medium">{{ e.actor_name || e.actor_email || 'Someone' }}</span>
            {{ AUDIT_ACTIONS[e.action] || e.action }}
            <span v-if="detail(e)" class="text-fg-muted">· {{ detail(e) }}</span>
          </p>
          <p class="text-xs text-fg-subtle">
            <time :datetime="e.created_at" :title="new Date(e.created_at).toLocaleString()">{{ relative(e.created_at) }}</time>
            <span v-if="e.ip"> · {{ e.ip }}</span>
          </p>
        </div>
      </li>
    </ol>
    <div v-if="hasMore" class="border-t border-line p-3 text-center">
      <button class="btn-ghost btn-sm" :disabled="loading" @click="loadMore">Load more</button>
    </div>
  </section>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { Download, ScrollText, Link2, QrCode, FileText, KeyRound, Users, Building2, Shield } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useToast } from '../../composables/useToast'
import { AUDIT_ACTIONS } from '../../lib/roles'
import EmptyState from '../ui/EmptyState.vue'

const PAGE = 50
const toast = useToast()
const events = ref([])
const loading = ref(false)
const hasMore = ref(false)
const category = ref('')

const visible = computed(() => (category.value ? events.value.filter((e) => e.action.startsWith(category.value + '.')) : events.value))

function iconFor(action) {
  const k = action.split('.')[0]
  return { link: Link2, qr: QrCode, document: FileText, api_key: KeyRound, member: Users, team: Building2, user: Shield }[k] || ScrollText
}

function detail(e) {
  try {
    const m = JSON.parse(e.metadata || '{}')
    if (m.email) return m.role ? `${m.email} as ${m.role}` : m.email
    if (m.from && m.to) return [e.target_label, `${m.from} → ${m.to}`].filter(Boolean).join(' · ')
    if (e.target_label) return e.target_label
    if (m.previous_slug) return `/s/${m.previous_slug} → /s/${m.slug}`
    return m.title || m.filename || (m.slug && `/s/${m.slug}`) || m.name || ''
  } catch {
    return ''
  }
}

function relative(d) {
  const s = (Date.now() - new Date(d)) / 1000
  if (s < 60) return 'just now'
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`
  if (s < 604800) return `${Math.floor(s / 86400)}d ago`
  return new Date(d).toLocaleDateString()
}

async function fetchPage(offset) {
  loading.value = true
  try {
    const res = await api.getAuditLog(PAGE, offset)
    events.value = offset ? [...events.value, ...res.data] : res.data
    hasMore.value = res.data.length === PAGE
  } catch (err) {
    toast.error(err.message)
  } finally {
    loading.value = false
  }
}
const loadMore = () => fetchPage(events.value.length)

function exportCSV() {
  const esc = (v) => `"${String(v ?? '').replace(/"/g, '""')}"`
  const rows = [['time', 'actor', 'email', 'action', 'target_type', 'target_id', 'details', 'ip']]
  visible.value.forEach((e) => rows.push([e.created_at, e.actor_name, e.actor_email, e.action, e.target_type, e.target_id, e.metadata, e.ip]))
  const blob = new Blob([rows.map((r) => r.map(esc).join(',')).join('\n')], { type: 'text/csv' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `lynqr-audit-${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(a.href)
}

onMounted(() => fetchPage(0))
</script>
