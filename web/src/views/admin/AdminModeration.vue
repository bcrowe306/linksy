<template>
  <div class="space-y-6">
    <!-- Reports -->
    <section class="card overflow-hidden">
      <header class="flex flex-col gap-3 border-b border-line px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-sm font-semibold text-fg">Abuse reports</h2>
          <p class="text-xs text-fg-muted">Sent by visitors from <a href="/report" target="_blank" class="text-brand-500 hover:underline">lynqr.app/report</a>.</p>
        </div>
        <select v-model="reportStatus" class="input input-sm w-36" aria-label="Report status" data-testid="report-status">
          <option value="open">Open</option>
          <option value="actioned">Actioned</option>
          <option value="dismissed">Dismissed</option>
        </select>
      </header>
      <div v-if="reports === null" class="space-y-3 p-5"><div v-for="i in 3" :key="i" class="skeleton h-10" /></div>
      <EmptyState v-else-if="!reports.length" :icon="Flag" :title="reportStatus === 'open' ? 'No open reports' : 'Nothing here'" />
      <ul v-else class="divide-y divide-line" data-testid="report-list">
        <li v-for="r in reports" :key="r.id" class="flex flex-col gap-2 px-5 py-3 sm:flex-row sm:items-start">
          <div class="min-w-0 flex-1">
            <p class="flex flex-wrap items-center gap-2 text-sm">
              <span class="badge-danger capitalize">{{ r.reason }}</span>
              <span class="truncate font-mono text-xs text-fg">{{ r.target_url }}</span>
            </p>
            <p v-if="r.details" class="mt-1 text-sm text-fg-muted">“{{ r.details }}”</p>
            <p class="mt-1 text-xs text-fg-subtle">{{ shortDate(r.created_at) }}<span v-if="r.reporter_email"> · {{ r.reporter_email }}</span><span v-if="r.resolved_by"> · {{ r.status }} by {{ r.resolved_by }}</span></p>
          </div>
          <div class="flex shrink-0 gap-2">
            <button class="btn-ghost btn-sm" @click="search(r.target_id)">Inspect</button>
            <template v-if="r.status === 'open'">
              <button class="btn-secondary btn-sm" @click="resolve(r, 'dismiss')">Dismiss</button>
              <button class="btn-danger btn-sm" :data-testid="`report-disable-${r.id}`" @click="resolve(r, 'disable')">Disable {{ r.target_type === 'qr' ? 'QR code' : 'link' }}</button>
            </template>
          </div>
        </li>
      </ul>
    </section>

    <!-- Search -->
    <section class="card overflow-hidden">
      <header class="border-b border-line px-5 py-4">
        <h2 class="text-sm font-semibold text-fg">Find links and QR codes</h2>
        <p class="mb-3 text-xs text-fg-muted">Search destinations (e.g. a domain), or paste a slug, scan code, link/QR ID or workspace ID.</p>
        <form class="flex gap-2" @submit.prevent="search(q)">
          <input v-model="q" type="search" placeholder="paypa1-login.com" class="input" data-testid="mod-search" />
          <button class="btn-secondary">Search</button>
        </form>
      </header>
      <p v-if="assets === null" class="px-5 py-6 text-sm text-fg-muted">Results appear here.</p>
      <EmptyState v-else-if="!assets.length" :icon="Search" title="No matches" />
      <ul v-else class="divide-y divide-line" data-testid="mod-results">
        <li v-for="a in assets" :key="a.type + a.id" class="flex flex-col gap-2 px-5 py-3 sm:flex-row sm:items-center">
          <div class="min-w-0 flex-1">
            <p class="flex flex-wrap items-center gap-2 text-sm">
              <span class="badge-neutral">{{ a.type === 'qr' ? 'QR code' : 'Link' }}</span>
              <span class="font-medium text-fg">{{ a.type === 'link' ? `/s/${a.label}` : a.label }}</span>
              <span v-if="a.moderation_reason" class="badge-danger" :title="a.moderation_reason">Disabled</span>
            </p>
            <p class="truncate text-xs text-fg-muted" :title="a.destination">→ {{ a.destination || '(no URL)' }}</p>
            <p class="text-xs text-fg-subtle">
              {{ a.team_name }} · {{ a.count }} {{ a.type === 'qr' ? 'scans' : 'clicks' }} · created {{ shortDate(a.created_at) }}
              <span v-if="a.moderation_reason"> · {{ a.moderation_reason }}</span>
            </p>
          </div>
          <div class="flex shrink-0 gap-2">
            <router-link :to="{ path: '/admin/workspaces', query: { q: a.team_id } }" class="btn-ghost btn-sm">Workspace</router-link>
            <button v-if="a.moderation_reason" class="btn-secondary btn-sm" @click="restore(a)">Restore</button>
            <button v-else class="btn-danger-ghost btn-sm" :data-testid="`mod-disable-${a.id}`" @click="target = a">Disable</button>
          </div>
        </li>
      </ul>
    </section>

    <!-- Blocklist -->
    <section class="card overflow-hidden">
      <header class="border-b border-line px-5 py-4">
        <h2 class="text-sm font-semibold text-fg">Blocked domains</h2>
        <p class="mb-3 text-xs text-fg-muted">New links and QR codes can't point at these domains or their subdomains, and existing ones stop redirecting.</p>
        <form class="flex flex-col gap-2 sm:flex-row" @submit.prevent="block">
          <input v-model="domain" required placeholder="example.com" class="input sm:w-64" data-testid="block-domain" />
          <input v-model="domainReason" placeholder="Reason (optional)" maxlength="300" class="input flex-1" />
          <button class="btn-danger" data-testid="block-submit"><Ban class="h-4 w-4" /> Block</button>
        </form>
      </header>
      <EmptyState v-if="blocked && !blocked.length" :icon="ShieldCheck" title="No blocked domains" />
      <ul v-else class="divide-y divide-line" data-testid="blocklist">
        <li v-for="d in blocked || []" :key="d.domain" class="flex items-center gap-3 px-5 py-2.5 text-sm">
          <span class="font-mono text-fg">{{ d.domain }}</span>
          <span class="flex-1 truncate text-xs text-fg-muted">{{ d.reason }}<span v-if="d.created_by"> · {{ d.created_by }}, {{ shortDate(d.created_at) }}</span></span>
          <button class="btn-ghost btn-sm" @click="unblock(d)">Unblock</button>
        </li>
      </ul>
    </section>

    <ReasonModal :open="!!target" :title="`Disable ${target?.type === 'qr' ? 'QR code' : 'link'}`" description="Visitors see a 'disabled' page. The owner sees it's disabled but can't turn it back on." confirm-text="Disable" @close="target = null" @submit="disable" />
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Flag, Search, Ban, ShieldCheck } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useToast } from '../../composables/useToast'
import { useAdminCall, shortDate } from '../../composables/useAdmin'
import EmptyState from '../../components/ui/EmptyState.vue'
import ReasonModal from '../../components/admin/ReasonModal.vue'

const call = useAdminCall()
const toast = useToast()
const route = useRoute()
const reportStatus = ref('open')
const reports = ref(null)
const q = ref(String(route.query.q || ''))
const assets = ref(null)
const blocked = ref(null)
const domain = ref('')
const domainReason = ref('')
const target = ref(null)

const loadReports = async () => (reports.value = (await call(() => api.admin.reports(reportStatus.value))).reports)
const loadBlocked = async () => (blocked.value = (await call(() => api.admin.blocklist())).domains)
watch(reportStatus, loadReports)
onMounted(() => {
  loadReports()
  loadBlocked()
  if (q.value) search(q.value)
})

async function search(term) {
  q.value = term
  if (term.trim().length < 2) return
  assets.value = (await call(() => api.admin.search(term.trim()))).assets
}

async function disable(reason) {
  const a = target.value
  target.value = null
  await call(() => api.admin.moderate(a.type, a.id, 'disable', reason))
  toast.success('Disabled')
  await search(q.value)
}
async function restore(a) {
  await call(() => api.admin.moderate(a.type, a.id, 'restore'))
  toast.success('Restored')
  await search(q.value)
}
async function resolve(r, action) {
  await call(() => api.admin.resolveReport(r.id, action))
  toast.success(action === 'disable' ? 'Disabled and resolved' : 'Report dismissed')
  await loadReports()
}
async function block() {
  await call(() => api.admin.block(domain.value, domainReason.value))
  toast.success(`Blocked ${domain.value}`)
  domain.value = domainReason.value = ''
  await loadBlocked()
}
async function unblock(d) {
  await call(() => api.admin.unblock(d.domain))
  await loadBlocked()
}
</script>
