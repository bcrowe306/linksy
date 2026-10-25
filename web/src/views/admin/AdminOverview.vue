<template>
  <div class="space-y-6">
    <div class="flex justify-end">
      <select v-model.number="days" class="input input-sm w-36" aria-label="Time range" data-testid="admin-range">
        <option :value="7">Last 7 days</option>
        <option :value="30">Last 30 days</option>
        <option :value="90">Last 90 days</option>
        <option :value="365">Last 12 months</option>
      </select>
    </div>

    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <div class="card p-5" data-testid="admin-mrr">
        <p class="flex items-center gap-1.5 text-sm text-fg-muted">MRR <Tip text="Monthly recurring revenue at list price: $8.99 per monthly plan and $5 per annual plan. Discounts aren't subtracted." /></p>
        <p class="mt-2 text-3xl font-semibold tracking-tight text-fg tabular-nums">{{ m ? money(m.mrr_cents) : '—' }}</p>
        <p class="mt-1.5 text-xs text-fg-subtle">{{ paying }} paying workspace{{ paying === 1 ? '' : 's' }}</p>
      </div>
      <div class="card p-5">
        <p class="flex items-center gap-1.5 text-sm text-fg-muted">Trial conversion <Tip text="Of all workspaces whose free trial has ended, the share that now pays." /></p>
        <p class="mt-2 text-3xl font-semibold tracking-tight text-fg tabular-nums">{{ m ? `${m.conversion_pct.toFixed(1)}%` : '—' }}</p>
        <p class="mt-1.5 text-xs text-fg-subtle">{{ m?.trials_ended || 0 }} trials ended</p>
      </div>
      <StatTile label="Users" :value="m?.users || 0" :trend="trend('signups')" :icon="Users" :loading="loading" testid="admin-users" />
      <StatTile label="Workspaces" :value="m?.workspaces || 0" :trend="trend('workspaces')" color="--series-2" :icon="Building2" :loading="loading" />
    </div>

    <div class="grid gap-6 xl:grid-cols-3">
      <ChartCard class="xl:col-span-2" title="Signups" :table="table(['signups', 'workspaces'], ['New users', 'New workspaces'])">
        <LineChart :points="points(['signups', 'workspaces'])" :series="[{ key: 'signups', label: 'New users', color: '--series-1' }, { key: 'workspaces', label: 'New workspaces', color: '--series-2' }]" :loading="loading" :height="220" />
      </ChartCard>
      <ChartCard title="Workspace plans" tip="Every workspace by its current plan state.">
        <BarList :rows="planRows" :limit="7" :loading="loading" empty="No workspaces yet." />
      </ChartCard>
    </div>

    <div class="grid gap-6 lg:grid-cols-2">
      <ChartCard title="Engagement" :table="table(['clicks', 'scans'], ['Link clicks', 'QR scans'])">
        <LineChart :points="points(['clicks', 'scans'])" :series="[{ key: 'clicks', label: 'Link clicks', color: '--series-1' }, { key: 'scans', label: 'QR scans', color: '--series-2' }]" :loading="loading" :height="200" />
      </ChartCard>
      <ChartCard title="Content created" :subtitle="`${m?.links || 0} links and ${m?.qr_codes || 0} QR codes in total`" :table="table(['links', 'qr_codes'], ['Short links', 'QR codes'])">
        <LineChart :points="points(['links', 'qr_codes'])" :series="[{ key: 'links', label: 'Short links', color: '--series-1' }, { key: 'qr_codes', label: 'QR codes', color: '--series-2' }]" :loading="loading" :height="200" />
      </ChartCard>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { Users, Building2 } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useAdminCall, PLAN_BADGES, money } from '../../composables/useAdmin'
import StatTile from '../../components/charts/StatTile.vue'
import ChartCard from '../../components/charts/ChartCard.vue'
import LineChart from '../../components/charts/LineChart.vue'
import BarList from '../../components/charts/BarList.vue'
import Tip from '../../components/ui/Tip.vue'

const call = useAdminCall()
const days = ref(30)
const m = ref(null)
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    m.value = await call(() => api.admin.metrics(days.value))
  } finally {
    loading.value = false
  }
}
watch(days, load, { immediate: true })

const paying = computed(() => (m.value ? (m.value.plans.active || 0) + (m.value.plans.past_due || 0) : 0))
const planRows = computed(() =>
  Object.entries(m.value?.plans || {})
    .map(([k, count]) => ({ name: PLAN_BADGES[k]?.[1] || k, count }))
    .sort((a, b) => b.count - a.count)
)

// Daily buckets (UTC), with empty days filled in.
const dayKeys = computed(() => {
  const out = []
  const end = new Date()
  for (let i = days.value - 1; i >= 0; i--) {
    const d = new Date(Date.UTC(end.getUTCFullYear(), end.getUTCMonth(), end.getUTCDate() - i))
    out.push(d.toISOString().slice(0, 10))
  }
  return out
})
function lookup(key) {
  return Object.fromEntries((m.value?.series[key] || []).map((d) => [d.day, d.count]))
}
function points(keys) {
  const maps = Object.fromEntries(keys.map((k) => [k, lookup(k)]))
  return dayKeys.value.map((day) => ({ t: `${day}T00:00:00Z`, ...Object.fromEntries(keys.map((k) => [k, maps[k][day] || 0])) }))
}
const trend = (key) => points([key]).map((p) => p[key])
function table(keys, labels) {
  return { columns: ['Day', ...labels], rows: points(keys).map((p) => [p.t.slice(0, 10), ...keys.map((k) => p[k])]) }
}
</script>
