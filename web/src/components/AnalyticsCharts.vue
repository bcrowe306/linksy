<template>
  <div class="space-y-6">
    <!-- Summary Top Stat Cards -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <div class="bg-gray-800/80 border border-gray-700/80 rounded-2xl p-5">
        <div class="flex items-center justify-between text-gray-400 text-xs font-semibold mb-2">
          <span>Total Link Clicks</span>
          <MousePointerClick class="w-4 h-4 text-linksy-400" />
        </div>
        <div class="text-3xl font-extrabold text-white tracking-tight">
          {{ formatNumber(summary?.total_clicks || 0) }}
        </div>
        <div class="text-[11px] text-gray-500 mt-1 flex items-center gap-1">
          <TrendingUp class="w-3 h-3 text-emerald-400" />
          <span class="text-emerald-400 font-medium">+12%</span> vs last period
        </div>
      </div>

      <div class="bg-gray-800/80 border border-gray-700/80 rounded-2xl p-5">
        <div class="flex items-center justify-between text-gray-400 text-xs font-semibold mb-2">
          <span>Total QR Scans</span>
          <QrCode class="w-4 h-4 text-emerald-400" />
        </div>
        <div class="text-3xl font-extrabold text-white tracking-tight">
          {{ formatNumber(summary?.total_scans || 0) }}
        </div>
        <div class="text-[11px] text-gray-500 mt-1 flex items-center gap-1">
          <Smartphone class="w-3 h-3 text-emerald-400" />
          <span>Camera & App Scans</span>
        </div>
      </div>

      <div class="bg-gray-800/80 border border-gray-700/80 rounded-2xl p-5">
        <div class="flex items-center justify-between text-gray-400 text-xs font-semibold mb-2">
          <span>Top Traffic Source</span>
          <Globe2 class="w-4 h-4 text-amber-400" />
        </div>
        <div class="text-xl font-bold text-white tracking-tight truncate">
          {{ summary?.by_referrer?.[0]?.name || 'Direct / None' }}
        </div>
        <div class="text-[11px] text-gray-500 mt-1">
          {{ summary?.by_referrer?.[0]?.count || 0 }} total engagements
        </div>
      </div>

      <div class="bg-gray-800/80 border border-gray-700/80 rounded-2xl p-5">
        <div class="flex items-center justify-between text-gray-400 text-xs font-semibold mb-2">
          <span>Primary Device</span>
          <Monitor class="w-4 h-4 text-purple-400" />
        </div>
        <div class="text-xl font-bold text-white tracking-tight">
          {{ summary?.by_device?.[0]?.name || 'Mobile' }}
        </div>
        <div class="text-[11px] text-gray-500 mt-1">
          {{ summary?.by_device?.[0]?.count || 0 }} interactions
        </div>
      </div>
    </div>

    <!-- Detailed Breakdowns -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Device Distribution -->
      <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-5">
        <h3 class="text-sm font-semibold text-gray-200 mb-4 flex items-center gap-2">
          <Smartphone class="w-4 h-4 text-linksy-400" />
          <span>Device Distribution</span>
        </h3>
        <div v-if="summary?.by_device?.length" class="space-y-3">
          <div v-for="d in summary.by_device" :key="d.name" class="space-y-1">
            <div class="flex justify-between text-xs">
              <span class="text-gray-300 font-medium">{{ d.name }}</span>
              <span class="text-gray-400 font-mono">{{ d.count }} ({{ calcPercent(d.count, totalEngagements) }}%)</span>
            </div>
            <div class="h-2 w-full bg-gray-700/50 rounded-full overflow-hidden">
              <div
                class="h-full bg-linksy-500 rounded-full transition-all duration-500"
                :style="{ width: `${calcPercent(d.count, totalEngagements)}%` }"
              ></div>
            </div>
          </div>
        </div>
        <div v-else class="text-xs text-gray-500 py-6 text-center">
          No device data recorded yet.
        </div>
      </div>

      <!-- Browser Distribution -->
      <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-5">
        <h3 class="text-sm font-semibold text-gray-200 mb-4 flex items-center gap-2">
          <Compass class="w-4 h-4 text-emerald-400" />
          <span>Browser Breakdown</span>
        </h3>
        <div v-if="summary?.by_browser?.length" class="space-y-3">
          <div v-for="b in summary.by_browser" :key="b.name" class="space-y-1">
            <div class="flex justify-between text-xs">
              <span class="text-gray-300 font-medium">{{ b.name }}</span>
              <span class="text-gray-400 font-mono">{{ b.count }} ({{ calcPercent(b.count, totalEngagements) }}%)</span>
            </div>
            <div class="h-2 w-full bg-gray-700/50 rounded-full overflow-hidden">
              <div
                class="h-full bg-emerald-500 rounded-full transition-all duration-500"
                :style="{ width: `${calcPercent(b.count, totalEngagements)}%` }"
              ></div>
            </div>
          </div>
        </div>
        <div v-else class="text-xs text-gray-500 py-6 text-center">
          No browser data recorded yet.
        </div>
      </div>

      <!-- Top Referrers -->
      <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-5">
        <h3 class="text-sm font-semibold text-gray-200 mb-4 flex items-center gap-2">
          <Share2 class="w-4 h-4 text-amber-400" />
          <span>Top Referrers</span>
        </h3>
        <div v-if="summary?.by_referrer?.length" class="space-y-2.5">
          <div
            v-for="r in summary.by_referrer"
            :key="r.name"
            class="flex items-center justify-between p-2 rounded-lg bg-gray-800/40 border border-gray-750 text-xs"
          >
            <span class="font-medium text-gray-300 truncate max-w-[180px]">{{ r.name }}</span>
            <span class="px-2 py-0.5 rounded bg-gray-700 text-gray-300 font-mono text-[11px]">{{ r.count }}</span>
          </div>
        </div>
        <div v-else class="text-xs text-gray-500 py-6 text-center">
          No referrer channels tracked yet.
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import {
  MousePointerClick,
  QrCode,
  TrendingUp,
  Smartphone,
  Globe2,
  Monitor,
  Compass,
  Share2
} from 'lucide-vue-next'

const props = defineProps({
  summary: {
    type: Object,
    default: null
  }
})

const totalEngagements = computed(() => {
  if (!props.summary) return 1
  return (props.summary.total_clicks || 0) + (props.summary.total_scans || 0) || 1
})

function calcPercent(count, total) {
  if (!total) return 0
  return Math.round((count / total) * 100)
}

function formatNumber(num) {
  return new Intl.NumberFormat().format(num)
}
</script>
