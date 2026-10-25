<template>
  <div>
    <PageHeader title="Analytics" subtitle="Clicks and scans across your links and QR codes. Bots and link previews are excluded.">
      <template #title-extra><HelpLink guide="analytics">How we measure</HelpLink></template>
    </PageHeader>

    <!-- Filters: one row above everything they scope -->
    <div class="mb-6 flex flex-wrap items-center gap-2">
      <DateRangePicker v-model="range" />
      <select v-model="asset" class="input w-auto max-w-xs" aria-label="Filter by link or QR code" data-testid="asset-filter">
        <option value="">All links & QR codes</option>
        <optgroup v-if="store.links.length" label="Short links">
          <option v-for="l in store.links" :key="l.id" :value="`link:${l.id}`">{{ l.title || l.slug }}</option>
        </optgroup>
        <optgroup v-if="store.qrCodes.length" label="QR codes">
          <option v-for="q in store.qrCodes" :key="q.id" :value="`qr:${q.id}`">{{ q.title }}</option>
        </optgroup>
      </select>
      <span class="text-xs text-fg-subtle">Times in {{ params.tz }}</span>
      <DropdownMenu class="ml-auto">
        <template #trigger><button class="btn-secondary" data-testid="export-report"><Download class="h-4 w-4" /> Export</button></template>
        <MenuItem :icon="FileSpreadsheet" @click="exportAs('csv')">Raw events (CSV)</MenuItem>
        <MenuItem :icon="Braces" @click="exportAs('json')">Report data (JSON)</MenuItem>
        <MenuItem :icon="FileText" @click="exportAs('pdf')">Printable report (PDF)</MenuItem>
      </DropdownMenu>
    </div>

    <div v-if="error" class="card mb-6 p-4 text-sm text-rose-500">{{ error }}</div>

    <!-- KPIs -->
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <StatTile label="Total engagements" :value="r?.totals.total || 0" :previous="r?.previous.total" :trend="trend('total')" :icon="Activity" :loading="loading" testid="kpi-total" />
      <StatTile label="Link clicks" :value="r?.totals.clicks || 0" :previous="r?.previous.clicks" :trend="trend('clicks')" color="--series-1" :icon="MousePointerClick" :loading="loading" testid="kpi-clicks" />
      <StatTile label="QR scans" :value="r?.totals.scans || 0" :previous="r?.previous.scans" :trend="trend('scans')" color="--series-2" :icon="ScanLine" :loading="loading" testid="kpi-scans" />
      <StatTile label="Unique visitors" :value="r?.totals.visitors || 0" :previous="r?.previous.visitors" :icon="Users" :loading="loading" tip="Distinct visitors, estimated from anonymized network addresses." />
    </div>

    <!-- Trend -->
    <ChartCard class="mt-6" title="Engagement over time" :subtitle="granularityLabel" :table="seriesTable">
      <LineChart v-if="r" :points="r.series" :series="seriesDefs" :granularity="r.granularity" :loading="loading" />
      <div v-else class="skeleton h-64" />
    </ChartCard>

    <!-- Geography -->
    <div class="mt-6 grid gap-6 xl:grid-cols-3">
      <ChartCard class="xl:col-span-2" title="Where people are" tip="From Cloudflare's visitor location data." :table="countryTable">
        <WorldMap :rows="r?.countries || []" :loading="loading" />
      </ChartCard>
      <ChartCard title="Top locations" :table="cityTable">
        <div class="mb-3 grid grid-cols-2 gap-1 rounded-xl bg-subtle p-1">
          <button v-for="t in ['countries', 'cities']" :key="t" class="rounded-lg py-1 text-xs font-medium capitalize" :class="geoTab === t ? 'bg-surface text-fg shadow-card' : 'text-fg-muted'" @click="geoTab = t">{{ t }}</button>
        </div>
        <BarList v-if="geoTab === 'countries'" :rows="r?.countries || []" :labeler="countryName" :loading="loading">
          <template #icon="{ row }"><span aria-hidden="true">{{ flag(row.name) }}</span></template>
        </BarList>
        <BarList v-else :rows="cityRows" :loading="loading" empty="City data appears as new visits arrive." />
      </ChartCard>
    </div>

    <!-- Tech & sources -->
    <div class="mt-6 grid gap-6 md:grid-cols-2 xl:grid-cols-3">
      <ChartCard title="Devices" :table="nameTable(r?.devices)"><BarList :rows="r?.devices || []" :loading="loading" /></ChartCard>
      <ChartCard title="Browsers" :table="nameTable(r?.browsers)"><BarList :rows="r?.browsers || []" :loading="loading" /></ChartCard>
      <ChartCard title="Operating systems" :table="nameTable(r?.os)"><BarList :rows="r?.os || []" :loading="loading" /></ChartCard>
      <ChartCard title="Channels" tip="Direct, social, email, search, other websites, or a QR scan." :table="nameTable(r?.channels)">
        <BarList :rows="r?.channels || []" :labeler="channelName" :loading="loading" />
      </ChartCard>
      <ChartCard title="Referring sites" :table="nameTable(r?.referrers)"><BarList :rows="r?.referrers || []" :loading="loading" /></ChartCard>
      <ChartCard title="When people engage" :subtitle="`Day of week × hour (${params.tz})`">
        <HourHeatmap v-if="r" :matrix="r.heatmap" :loading="loading" />
      </ChartCard>
    </div>

    <!-- Top assets -->
    <div class="mt-6 grid gap-6 lg:grid-cols-2">
      <ChartCard title="Top links" :table="assetTable(r?.top_links, 'Clicks')">
        <BarList :rows="assetRows(r?.top_links)" :loading="loading" empty="No link clicks in this period." />
      </ChartCard>
      <ChartCard title="Top QR codes" :table="assetTable(r?.top_qr_codes, 'Scans')">
        <BarList :rows="assetRows(r?.top_qr_codes)" color="--series-2" :loading="loading" empty="No QR scans in this period." />
      </ChartCard>
    </div>
  </div>
</template>

<script setup>
import HelpLink from '../components/ui/HelpLink.vue'
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Activity, MousePointerClick, ScanLine, Users, Download, FileSpreadsheet, FileText, Braces } from 'lucide-vue-next'
import { api } from '../api/client'
import { useLynqrStore } from '../stores/lynqr'
import { useReport } from '../composables/useReport'
import { rangeParams, rangeFromQuery } from '../lib/range'
import { loadCountries, flag } from '../lib/countries'
import PageHeader from '../components/ui/PageHeader.vue'
import DropdownMenu from '../components/ui/DropdownMenu.vue'
import MenuItem from '../components/ui/MenuItem.vue'
import DateRangePicker from '../components/charts/DateRangePicker.vue'
import StatTile from '../components/charts/StatTile.vue'
import ChartCard from '../components/charts/ChartCard.vue'
import LineChart from '../components/charts/LineChart.vue'
import BarList from '../components/charts/BarList.vue'
import WorldMap from '../components/charts/WorldMap.vue'
import HourHeatmap from '../components/charts/HourHeatmap.vue'

const route = useRoute()
const router = useRouter()
const store = useLynqrStore()

const range = ref(rangeFromQuery(route.query))
const asset = ref(route.query.link ? `link:${route.query.link}` : route.query.qr ? `qr:${route.query.qr}` : '')
const geoTab = ref('countries')

const params = computed(() => {
  const p = rangeParams(range.value)
  const [kind, id] = asset.value.split(':')
  if (kind === 'link') p.link_id = id
  if (kind === 'qr') p.qr_id = id
  return p
})
const { report: r, loading, error } = useReport(params)

// Keep the URL shareable
watch([range, asset], () => {
  const [kind, id] = asset.value.split(':')
  const q = { range: range.value.preset }
  if (range.value.preset === 'custom') Object.assign(q, { from: range.value.from, to: range.value.to })
  if (kind) q[kind] = id
  router.replace({ query: q })
})

const seriesDefs = computed(() => {
  const [kind] = asset.value.split(':')
  if (kind === 'link') return [{ key: 'clicks', label: 'Link clicks', color: '--series-1' }]
  if (kind === 'qr') return [{ key: 'scans', label: 'QR scans', color: '--series-2' }]
  return [
    { key: 'clicks', label: 'Link clicks', color: '--series-1' },
    { key: 'scans', label: 'QR scans', color: '--series-2' }
  ]
})
const granularityLabel = computed(() => r.value && `${{ hour: 'Hourly', day: 'Daily', week: 'Weekly' }[r.value.granularity]} totals`)
const trend = (k) => r.value?.series.map((p) => (k === 'total' ? p.clicks + p.scans : p[k])) || []

const names = ref(null)
onMounted(async () => (names.value = await loadCountries()))
const countryName = (c) => names.value?.name(c) || c
const channelName = (c) => ({ direct: 'Direct', social: 'Social', email: 'Email', search: 'Search', web: 'Other websites', qr: 'QR scan' })[c] || c
const cityRows = computed(() => (r.value?.cities || []).map((c) => ({ name: `${flag(c.country)} ${c.city}${c.region ? `, ${c.region}` : ''}`, count: c.count })))
const assetRows = (list) => (list || []).map((a) => ({ name: a.title || a.label, count: a.count }))

// Accessible table twins
const nameTable = (list) => list && { columns: ['Name', 'Engagements'], rows: list.map((n) => [n.name, n.count.toLocaleString()]) }
const assetTable = (list, label) => list && { columns: ['Name', 'Detail', label], rows: list.map((a) => [a.title, a.label, a.count.toLocaleString()]) }
const seriesTable = computed(() => r.value && {
  columns: ['Period', ...seriesDefs.value.map((s) => s.label)],
  rows: r.value.series.map((p) => [new Date(p.t).toLocaleString(), ...seriesDefs.value.map((s) => p[s.key].toLocaleString())])
})
const countryTable = computed(() => r.value && { columns: ['Country', 'Engagements'], rows: r.value.countries.map((c) => [countryName(c.name), c.count.toLocaleString()]) })
const cityTable = computed(() => r.value && { columns: ['City', 'Region', 'Country', 'Engagements'], rows: r.value.cities.map((c) => [c.city, c.region, countryName(c.country), c.count.toLocaleString()]) })

function exportAs(format) {
  const a = document.createElement('a')
  a.href = api.analyticsExportUrl({ ...params.value, format })
  document.body.appendChild(a)
  a.click()
  a.remove()
}
</script>
