<template>
  <section class="card overflow-hidden">
    <header class="flex flex-col gap-3 border-b border-line px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h2 class="text-sm font-semibold text-fg">Workspaces</h2>
        <p class="text-xs text-fg-muted">Search by name, owner email, workspace ID or Stripe customer ID.</p>
      </div>
      <div class="relative sm:w-80">
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-subtle" />
        <input v-model="q" type="search" placeholder="Search workspaces" class="input input-sm pl-9" data-testid="admin-ws-search" />
      </div>
    </header>

    <div v-if="loading && !rows.length" class="space-y-3 p-5"><div v-for="i in 6" :key="i" class="skeleton h-10" /></div>
    <EmptyState v-else-if="!rows.length" :icon="Building2" title="No workspaces found" />
    <div v-else class="overflow-x-auto">
      <table class="w-full text-sm" data-testid="admin-ws-table">
        <thead class="text-left text-xs text-fg-subtle">
          <tr class="border-b border-line">
            <th class="px-5 py-2.5 font-medium">Workspace</th>
            <th class="px-3 py-2.5 font-medium">Plan</th>
            <th class="px-3 py-2.5 text-right font-medium">Members</th>
            <th class="px-3 py-2.5 text-right font-medium">Links</th>
            <th class="px-3 py-2.5 text-right font-medium">QR</th>
            <th class="px-5 py-2.5 font-medium">Created</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-line">
          <tr v-for="w in rows" :key="w.id" class="cursor-pointer hover:bg-subtle/60" @click="open(w.id)">
            <td class="px-5 py-3">
              <p class="font-medium text-fg">{{ w.name }}</p>
              <p class="text-xs text-fg-muted">{{ w.owner_email }}</p>
            </td>
            <td class="px-3 py-3">
              <span :class="badge(w)[0]">{{ badge(w)[1] }}</span>
              <span v-if="w.suspended_reason" class="badge-danger ml-1">Suspended</span>
            </td>
            <td class="px-3 py-3 text-right tabular-nums">{{ w.members }}</td>
            <td class="px-3 py-3 text-right tabular-nums">{{ w.links }}</td>
            <td class="px-3 py-3 text-right tabular-nums">{{ w.qr_codes }}</td>
            <td class="px-5 py-3 text-fg-muted">{{ shortDate(w.created_at) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="hasMore" class="border-t border-line p-3 text-center"><button class="btn-ghost btn-sm" :disabled="loading" @click="more">Load more</button></div>

    <AppModal :open="!!detail" :title="detail?.team.name" :description="detail ? `${detail.team.id} · owner ${detail.team.owner_email}` : ''" size="xl" @close="detail = null">
      <div v-if="detail" class="space-y-6" data-testid="admin-ws-detail">
        <div v-if="detail.team.suspended_reason" class="rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-fg">
          <span class="font-semibold">Suspended:</span> {{ detail.team.suspended_reason }}
        </div>
        <dl class="grid gap-4 text-sm sm:grid-cols-4">
          <div><dt class="text-xs text-fg-subtle">Plan</dt><dd class="mt-1"><span :class="badge(detail.team)[0]">{{ badge(detail.team)[1] }}</span></dd></div>
          <div><dt class="text-xs text-fg-subtle">Trial ends</dt><dd class="mt-1 text-fg">{{ shortDate(detail.team.trial_ends_at) }}</dd></div>
          <div><dt class="text-xs text-fg-subtle">Billing</dt><dd class="mt-1 text-fg">{{ detail.team.interval ? `${detail.team.interval}ly · renews ${shortDate(detail.team.current_period_end)}` : '—' }}</dd></div>
          <div><dt class="text-xs text-fg-subtle">Discount</dt><dd class="mt-1 text-fg">{{ detail.team.discount_label || '—' }}</dd></div>
        </dl>
        <a v-if="detail.team.stripe_customer_id" :href="`https://dashboard.stripe.com/customers/${detail.team.stripe_customer_id}`" target="_blank" rel="noopener" class="inline-flex items-center gap-1 text-xs font-medium text-brand-500 hover:underline">
          Open in Stripe <ExternalLink class="h-3 w-3" />
        </a>

        <div class="flex flex-wrap gap-2">
          <form class="flex items-center gap-2" @submit.prevent="extend">
            <input v-model.number="extendDays" type="number" min="1" max="365" class="input input-sm w-20" aria-label="Days" />
            <button class="btn-secondary btn-sm" data-testid="admin-extend"><CalendarPlus class="h-3.5 w-3.5" /> Extend trial</button>
          </form>
          <button class="btn-secondary btn-sm" data-testid="admin-comp" @click="act('comp', { comped: !detail.team.comped })">
            <Gift class="h-3.5 w-3.5" /> {{ detail.team.comped ? 'Remove comp' : 'Comp workspace' }}
          </button>
          <button v-if="detail.team.suspended_reason" class="btn-secondary btn-sm" @click="act('unsuspend')"><ShieldCheck class="h-3.5 w-3.5" /> Lift suspension</button>
          <button v-else class="btn-danger-ghost btn-sm" data-testid="admin-suspend" @click="reasonOpen = true"><Ban class="h-3.5 w-3.5" /> Suspend</button>
          <router-link :to="{ path: '/admin/moderation', query: { q: detail.team.id } }" class="btn-ghost btn-sm"><Flag class="h-3.5 w-3.5" /> Review content</router-link>
        </div>

        <div class="grid gap-6 lg:grid-cols-2">
          <div>
            <h3 class="mb-2 text-xs font-semibold uppercase tracking-wide text-fg-subtle">Members ({{ detail.members.length }})</h3>
            <ul class="divide-y divide-line rounded-xl border border-line">
              <li v-for="mem in detail.members" :key="mem.user_id" class="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                <span class="truncate text-fg">{{ mem.email }}</span>
                <span class="text-xs capitalize text-fg-muted">{{ mem.role }}<span v-if="mem.status !== 'active'"> · {{ mem.status }}</span></span>
              </li>
            </ul>
          </div>
          <div>
            <h3 class="mb-2 text-xs font-semibold uppercase tracking-wide text-fg-subtle">Recent workspace activity</h3>
            <p v-if="!detail.audit.length" class="text-sm text-fg-muted">Nothing yet.</p>
            <ul v-else class="max-h-64 space-y-1.5 overflow-y-auto text-xs">
              <li v-for="e in detail.audit" :key="e.id" class="flex justify-between gap-3">
                <span class="truncate text-fg"><span class="text-fg-muted">{{ e.actor_email || 'system' }}</span> {{ e.action }}</span>
                <time class="shrink-0 text-fg-subtle">{{ shortDate(e.created_at) }}</time>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </AppModal>
    <ReasonModal :open="reasonOpen" title="Suspend workspace" description="Members can still sign in and view, but can't change anything. Its links and QR codes show a 'disabled' page." confirm-text="Suspend" @close="reasonOpen = false" @submit="suspend" />
  </section>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Search, Building2, CalendarPlus, Gift, Ban, ShieldCheck, Flag, ExternalLink } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useToast } from '../../composables/useToast'
import { useAdminCall, PLAN_BADGES, shortDate } from '../../composables/useAdmin'
import AppModal from '../../components/ui/AppModal.vue'
import EmptyState from '../../components/ui/EmptyState.vue'
import ReasonModal from '../../components/admin/ReasonModal.vue'

const call = useAdminCall()
const toast = useToast()
const route = useRoute()
const q = ref(String(route.query.q || ''))
const rows = ref([])
const loading = ref(false)
const hasMore = ref(false)
const detail = ref(null)
const extendDays = ref(7)
const reasonOpen = ref(false)

const badge = (w) => PLAN_BADGES[w.state] || ['badge-neutral', w.state]

async function load(append = false) {
  loading.value = true
  try {
    const { workspaces } = await call(() => api.admin.workspaces(q.value.trim(), append ? rows.value.length : 0))
    rows.value = append ? [...rows.value, ...workspaces] : workspaces
    hasMore.value = workspaces.length === 50
  } finally {
    loading.value = false
  }
}
const more = () => load(true)
let timer
watch(q, () => {
  clearTimeout(timer)
  timer = setTimeout(() => load(), 250)
})
onMounted(() => load())

async function open(id) {
  detail.value = await call(() => api.admin.workspace(id))
}

async function act(action, body = {}) {
  const { workspace } = await call(() => api.admin.workspaceAction(detail.value.team.id, action, body))
  detail.value = { ...detail.value, team: workspace }
  const i = rows.value.findIndex((r) => r.id === workspace.id)
  if (i >= 0) rows.value[i] = workspace
  toast.success('Workspace updated')
}
const extend = () => act('trial', { days: extendDays.value })
async function suspend(reason) {
  reasonOpen.value = false
  await act('suspend', { reason })
}
</script>
