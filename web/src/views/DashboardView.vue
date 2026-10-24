<template>
  <div class="space-y-8">
    <!-- Top Greeting & Quick Hero -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h1 class="text-2xl font-extrabold text-white tracking-tight">Overview Dashboard</h1>
        <p class="text-xs text-gray-400 mt-1">
          Monitor your dynamic QR codes, short links, and engagement analytics in real-time.
        </p>
      </div>

      <button
        @click="$emit('open-create')"
        class="bg-linksy-600 hover:bg-linksy-500 text-white px-4 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 self-start md:self-auto shadow-lg shadow-linksy-600/20"
      >
        <Plus class="w-4 h-4" />
        <span>Create QR or Link</span>
      </button>
    </div>

    <!-- Analytics Charts & Metrics (Requirement 3 & 4) -->
    <AnalyticsCharts :summary="store.analyticsSummary" />

    <!-- Two-column Recent Links & QR Codes -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Recent Short Links -->
      <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-6">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-2">
            <Link2 class="w-4 h-4 text-linksy-400" />
            <h2 class="text-sm font-bold text-white">Recent Short Links</h2>
          </div>
          <router-link to="/links" class="text-xs text-linksy-400 hover:underline">View All</router-link>
        </div>

        <div v-if="store.links.length" class="divide-y divide-gray-750">
          <div
            v-for="link in store.links.slice(0, 5)"
            :key="link.id"
            class="py-3 flex items-center justify-between text-xs"
          >
            <div class="truncate max-w-[280px]">
              <div class="font-semibold text-gray-200 truncate">{{ link.title }}</div>
              <div class="text-gray-400 font-mono text-[11px] truncate flex items-center gap-1.5 mt-0.5">
                <span class="text-linksy-400">/s/{{ link.slug }}</span>
                <span class="text-gray-600">•</span>
                <span class="text-gray-500 truncate">{{ link.target_url }}</span>
              </div>
            </div>

            <div class="flex items-center gap-3">
              <span class="text-gray-300 font-mono text-[11px] font-semibold">{{ link.click_count }} clicks</span>
              <button
                @click="copyUrl(`/s/${link.slug}`)"
                class="p-1.5 rounded-lg bg-gray-700 hover:bg-gray-600 text-gray-300"
                title="Copy short link"
              >
                <Copy class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        </div>
        <div v-else class="text-center py-8 text-xs text-gray-500">
          No short links created yet. Click "Create QR or Link" to begin.
        </div>
      </div>

      <!-- Recent QR Codes -->
      <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-6">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-2">
            <QrCode class="w-4 h-4 text-emerald-400" />
            <h2 class="text-sm font-bold text-white">Recent QR Codes</h2>
          </div>
          <router-link to="/qr-codes" class="text-xs text-linksy-400 hover:underline">View All</router-link>
        </div>

        <div v-if="store.qrCodes.length" class="divide-y divide-gray-750">
          <div
            v-for="qr in store.qrCodes.slice(0, 5)"
            :key="qr.id"
            class="py-3 flex items-center justify-between text-xs"
          >
            <div class="flex items-center gap-3">
              <img
                :src="`/api/v1/qr/${qr.id}/export?format=png&size=80`"
                class="w-9 h-9 rounded-lg bg-white p-0.5 border border-gray-700"
              />
              <div class="truncate max-w-[220px]">
                <div class="font-semibold text-gray-200 truncate">{{ qr.title }}</div>
                <div class="text-[11px] text-gray-400 capitalize">{{ qr.type }} QR Code</div>
              </div>
            </div>

            <div class="flex items-center gap-2">
              <span class="text-gray-300 font-mono text-[11px] font-semibold">{{ qr.scan_count }} scans</span>
              <a
                :href="`/api/v1/qr/${qr.id}/export?format=png&download=true`"
                class="p-1.5 rounded-lg bg-gray-700 hover:bg-gray-600 text-gray-300"
                title="Download PNG"
              >
                <Download class="w-3.5 h-3.5" />
              </a>
            </div>
          </div>
        </div>
        <div v-else class="text-center py-8 text-xs text-gray-500">
          No QR codes generated yet.
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { Plus, Link2, QrCode, Copy, Download } from 'lucide-vue-next'
import { useLinksyStore } from '../stores/linksy'
import AnalyticsCharts from '../components/AnalyticsCharts.vue'

defineEmits(['open-create'])
const store = useLinksyStore()

function copyUrl(path) {
  const full = window.location.origin + path
  navigator.clipboard.writeText(full)
}
</script>
