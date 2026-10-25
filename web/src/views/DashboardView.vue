<template>
  <div>
    <div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h1 class="page-title">{{ greeting }}{{ firstName ? `, ${firstName}` : '' }}</h1>
        <p class="page-subtitle">Here's how {{ auth.team?.name || 'your workspace' }} is performing.</p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <DateRangePicker v-model="range" />
        <router-link :to="{ path: '/analytics', query: { range: range.preset } }" class="btn-secondary"><BarChart3 class="h-4 w-4" /> Full analytics</router-link>
      </div>
    </div>

    <!-- Getting started -->
    <section v-if="showOnboarding" class="card mb-6 overflow-hidden" data-testid="onboarding">
      <div class="flex flex-col gap-4 bg-gradient-to-r from-brand-500/10 via-violet-500/10 to-transparent p-5 sm:flex-row sm:items-center">
        <div class="flex-1">
          <h2 class="text-base font-semibold text-fg">Get started with Lynqr</h2>
          <p class="text-sm text-fg-muted">{{ doneCount }} of {{ steps.length }} done. Each step takes under a minute. <HelpLink guide="getting-started">Read the guide</HelpLink></p>
        </div>
        <div class="h-2 w-40 overflow-hidden rounded-full bg-subtle"><div class="h-full rounded-full bg-brand-500 transition-all" :style="{ width: `${(doneCount / steps.length) * 100}%` }" /></div>
      </div>
      <ul class="grid divide-y divide-line sm:grid-cols-2 sm:divide-y-0 lg:grid-cols-4">
        <li v-for="s in steps" :key="s.label" class="flex items-start gap-3 p-4">
          <CheckCircle2 v-if="s.done" class="mt-0.5 h-5 w-5 shrink-0 text-emerald-500" />
          <Circle v-else class="mt-0.5 h-5 w-5 shrink-0 text-fg-subtle" />
          <div>
            <p class="text-sm font-medium" :class="s.done ? 'text-fg-muted line-through' : 'text-fg'">{{ s.label }}</p>
            <button v-if="!s.done" class="mt-1 text-xs font-medium text-brand-500 hover:underline" @click="s.action">{{ s.cta }} →</button>
          </div>
        </li>
      </ul>
    </section>

    <!-- KPIs -->
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <StatTile label="Total engagements" :value="r?.totals.total || 0" :previous="r?.previous.total" :trend="trend('total')" :icon="Activity" :loading="loading" testid="dash-total" />
      <StatTile label="Link clicks" :value="r?.totals.clicks || 0" :previous="r?.previous.clicks" :trend="trend('clicks')" color="--series-1" :icon="MousePointerClick" :loading="loading" />
      <StatTile label="QR scans" :value="r?.totals.scans || 0" :previous="r?.previous.scans" :trend="trend('scans')" color="--series-2" :icon="ScanLine" :loading="loading" />
      <StatTile label="Unique visitors" :value="r?.totals.visitors || 0" :previous="r?.previous.visitors" :icon="Users" :loading="loading" tip="Distinct visitors, estimated from anonymized network addresses." />
    </div>

    <div class="mt-6 grid gap-6 xl:grid-cols-3">
      <ChartCard class="xl:col-span-2" title="Engagement over time" :table="seriesTable">
        <LineChart v-if="r" :points="r.series" :series="seriesDefs" :granularity="r.granularity" :loading="loading" :height="240" />
        <div v-else class="skeleton h-60" />
      </ChartCard>
      <ChartCard title="Live activity" tip="Clicks and scans from the last few moments, streamed in real time.">
        <LiveFeed @event="onLive" />
      </ChartCard>
    </div>

    <div class="mt-6 grid gap-6 lg:grid-cols-3">
      <ChartCard title="Top links">
        <BarList :rows="assetRows(r?.top_links)" :limit="5" :loading="loading" empty="No clicks yet in this period." />
      </ChartCard>
      <ChartCard title="Top QR codes">
        <BarList :rows="assetRows(r?.top_qr_codes)" :limit="5" color="--series-2" :loading="loading" empty="No scans yet in this period." />
      </ChartCard>
      <ChartCard title="Top countries">
        <BarList :rows="r?.countries || []" :limit="5" :labeler="countryName" :loading="loading">
          <template #icon="{ row }"><span aria-hidden="true">{{ flag(row.name) }}</span></template>
        </BarList>
      </ChartCard>
    </div>

    <!-- Recent assets -->
    <div class="mt-6 grid gap-6 lg:grid-cols-2">
      <section class="card overflow-hidden">
        <header class="flex items-center justify-between border-b border-line px-5 py-3.5">
          <h3 class="text-sm font-semibold text-fg">Recent links</h3>
          <router-link to="/links" class="text-xs font-medium text-brand-500 hover:underline">View all</router-link>
        </header>
        <p v-if="!store.links.length" class="px-5 py-8 text-center text-sm text-fg-muted">No links yet.</p>
        <ul v-else class="divide-y divide-line">
          <li v-for="l in store.links.slice(0, 5)" :key="l.id" class="flex items-center gap-3 px-5 py-3">
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-fg">{{ l.title || l.slug }}</p>
              <p class="truncate font-mono text-xs text-brand-500">/s/{{ l.slug }}</p>
            </div>
            <span class="text-xs text-fg-muted"><span class="font-semibold tabular-nums text-fg">{{ compact(l.click_count) }}</span> clicks</span>
            <CopyButton :value="`${origin}/s/${l.slug}`" label="Copy short link" />
          </li>
        </ul>
      </section>
      <section class="card overflow-hidden">
        <header class="flex items-center justify-between border-b border-line px-5 py-3.5">
          <h3 class="text-sm font-semibold text-fg">Recent QR codes</h3>
          <router-link to="/qr-codes" class="text-xs font-medium text-brand-500 hover:underline">View all</router-link>
        </header>
        <p v-if="!store.qrCodes.length" class="px-5 py-8 text-center text-sm text-fg-muted">No QR codes yet.</p>
        <ul v-else class="divide-y divide-line">
          <li v-for="q in store.qrCodes.slice(0, 5)" :key="q.id" class="flex items-center gap-3 px-5 py-2.5">
            <img :src="`${api.getExportUrl(q.id, 'png', 128, false)}&v=${new Date(q.updated_at).getTime()}`" alt="" class="h-10 w-10 rounded-md border border-line bg-white object-contain" loading="lazy" />
            <router-link :to="`/qr-studio/${q.id}`" class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-fg hover:underline">{{ q.title }}</p>
              <p class="text-xs capitalize text-fg-subtle">{{ q.type }}<span v-if="q.is_dynamic"> · dynamic</span></p>
            </router-link>
            <span class="text-xs text-fg-muted"><span class="font-semibold tabular-nums text-fg">{{ compact(q.scan_count) }}</span> scans</span>
          </li>
        </ul>
      </section>
    </div>
  </div>
</template>

<script setup>
import HelpLink from '../components/ui/HelpLink.vue'
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Activity, MousePointerClick, ScanLine, Users, BarChart3, CheckCircle2, Circle } from 'lucide-vue-next'
import { api } from '../api/client'
import { useLynqrStore } from '../stores/lynqr'
import { useAuthStore } from '../stores/auth'
import { useReport } from '../composables/useReport'
import { rangeParams } from '../lib/range'
import { compact } from '../lib/format'
import { loadCountries, flag } from '../lib/countries'
import DateRangePicker from '../components/charts/DateRangePicker.vue'
import StatTile from '../components/charts/StatTile.vue'
import ChartCard from '../components/charts/ChartCard.vue'
import LineChart from '../components/charts/LineChart.vue'
import BarList from '../components/charts/BarList.vue'
import LiveFeed from '../components/charts/LiveFeed.vue'
import CopyButton from '../components/ui/CopyButton.vue'

const emit = defineEmits(['open-create'])
const store = useLynqrStore()
const auth = useAuthStore()
const router = useRouter()
const origin = window.location.origin

const range = ref({ preset: '7d' })
const params = computed(() => rangeParams(range.value))
const { report: r, loading, reload } = useReport(params)

const firstName = computed(() => auth.user?.first_name || '')
const greeting = computed(() => {
  const h = new Date().getHours()
  return h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening'
})

const seriesDefs = [
  { key: 'clicks', label: 'Link clicks', color: '--series-1' },
  { key: 'scans', label: 'QR scans', color: '--series-2' }
]
const trend = (k) => r.value?.series.map((p) => (k === 'total' ? p.clicks + p.scans : p[k])) || []
const seriesTable = computed(() => r.value && {
  columns: ['Period', 'Link clicks', 'QR scans'],
  rows: r.value.series.map((p) => [new Date(p.t).toLocaleString(), p.clicks, p.scans])
})
const assetRows = (list) => (list || []).map((a) => ({ name: a.title || a.label, count: a.count }))

const names = ref(null)
onMounted(async () => (names.value = await loadCountries()))
const countryName = (c) => names.value?.name(c) || c

// Live events refresh the numbers shortly after they arrive (debounced).
let refreshTimer
function onLive() {
  clearTimeout(refreshTimer)
  refreshTimer = setTimeout(() => {
    reload()
    store.fetchLinks()
    store.fetchQRCodes()
  }, 1500)
}

const steps = computed(() => [
  { label: 'Create a short link', done: store.links.some((l) => !store.qrCodes.some((q) => q.link_id === l.id && q.owns_link)), cta: 'Create link', action: () => emit('open-create', 'link') },
  { label: 'Design a QR code', done: store.qrCodes.length > 0, cta: 'Open QR Studio', action: () => router.push('/qr-studio') },
  { label: 'Get your first click or scan', done: (r.value?.totals.total || 0) > 0 || store.links.some((l) => l.click_count) || store.qrCodes.some((q) => q.scan_count), cta: 'Copy a link to share', action: () => router.push('/links') },
  { label: 'Invite a teammate', done: (auth.team?.member_count || 1) > 1, cta: 'Invite', action: () => router.push('/settings?tab=members') }
])
const doneCount = computed(() => steps.value.filter((s) => s.done).length)
const showOnboarding = computed(() => auth.can('assets:edit') && doneCount.value < steps.value.length)
</script>
