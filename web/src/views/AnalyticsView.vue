<template>
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold text-gray-50">Tracking & Analytics Reports</h1>
        <p class="text-xs text-gray-400 mt-0.5">Filter by asset and generate on-demand performance reports.</p>
      </div>

      <!-- Asset Filter & Export -->
      <div class="flex items-center gap-3">
        <select
          v-model="selectedFilter"
          @change="onFilterChange"
          class="bg-gray-800 border border-gray-700 rounded-xl px-3 py-1.5 text-xs text-gray-200 focus:outline-none focus:ring-2 focus:ring-lynqr-500"
        >
          <option value="">All Assets (Global)</option>
          <optgroup label="Short Links">
            <option v-for="l in store.links" :key="l.id" :value="'link:' + l.id">{{ l.title }} (/s/{{ l.slug }})</option>
          </optgroup>
          <optgroup label="QR Codes">
            <option v-for="q in store.qrCodes" :key="q.id" :value="'qr:' + q.id">{{ q.title }} (QR)</option>
          </optgroup>
        </select>

        <button
          @click="exportCSV"
          class="bg-gray-800 hover:bg-gray-750 border border-gray-700 text-gray-200 px-3 py-1.5 rounded-xl text-xs font-medium flex items-center gap-1.5 transition-colors"
        >
          <Download class="w-3.5 h-3.5" />
          <span>Export CSV</span>
        </button>
      </div>
    </div>

    <!-- Charts and distributions -->
    <AnalyticsCharts :summary="store.analyticsSummary" />

    <!-- Country & City Breakdown Table -->
    <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-6">
      <h3 class="text-sm font-semibold text-gray-200 mb-4 flex items-center gap-2">
        <Globe2 class="w-4 h-4 text-lynqr-400" />
        <span>Geographic Distribution (Top Countries)</span>
      </h3>

      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs text-gray-300">
          <thead class="bg-gray-800/80 text-[11px] uppercase tracking-wider text-gray-400 border-b border-gray-700">
            <tr>
              <th class="px-4 py-2.5">Country / Region</th>
              <th class="px-4 py-2.5">Interactions</th>
              <th class="px-4 py-2.5">Engagement Share</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-750">
            <tr v-for="c in store.analyticsSummary?.by_country || []" :key="c.name" class="hover:bg-gray-800/40">
              <td class="px-4 py-3 font-semibold text-gray-200">{{ c.name }}</td>
              <td class="px-4 py-3 font-mono">{{ c.count }}</td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-3 max-w-xs">
                  <div class="h-2 w-32 bg-gray-700 rounded-full overflow-hidden">
                    <div
                      class="h-full bg-lynqr-500 rounded-full"
                      :style="{ width: `${Math.round((c.count / (store.analyticsSummary?.total_clicks + store.analyticsSummary?.total_scans || 1)) * 100)}%` }"
                    ></div>
                  </div>
                  <span class="text-gray-400 font-mono text-[11px]">
                    {{ Math.round((c.count / (store.analyticsSummary?.total_clicks + store.analyticsSummary?.total_scans || 1)) * 100) }}%
                  </span>
                </div>
              </td>
            </tr>
            <tr v-if="!store.analyticsSummary?.by_country?.length">
              <td colspan="3" class="px-4 py-8 text-center text-gray-500">
                No geographic data recorded yet.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Download, Globe2 } from 'lucide-vue-next'
import { useLynqrStore } from '../stores/lynqr'
import AnalyticsCharts from '../components/AnalyticsCharts.vue'

const store = useLynqrStore()
const selectedFilter = ref('')

async function onFilterChange() {
  if (!selectedFilter.value) {
    await store.fetchAnalytics(null, null)
    return
  }
  const [type, id] = selectedFilter.value.split(':')
  if (type === 'link') {
    await store.fetchAnalytics(id, null)
  } else if (type === 'qr') {
    await store.fetchAnalytics(null, id)
  }
}

function exportCSV() {
  const sum = store.analyticsSummary
  if (!sum) return
  let csv = 'Metric,Value\n'
  csv += `Total Clicks,${sum.total_clicks}\n`
  csv += `Total Scans,${sum.total_scans}\n\n`
  csv += 'Country,Count\n'
  sum.by_country?.forEach(c => {
    csv += `"${c.name}",${c.count}\n`
  })
  csv += '\nDevice,Count\n'
  sum.by_device?.forEach(d => {
    csv += `"${d.name}",${d.count}\n`
  })

  const blob = new Blob([csv], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `lynqr-analytics-report-${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
}
</script>
