<template>
  <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-6 flex flex-col items-center">
    <div class="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-4 flex items-center gap-2">
      <Eye class="w-4 h-4 text-lynqr-400" />
      <span>Live Preview & Export</span>
    </div>

    <!-- QR Code Canvas / Container -->
    <div
      class="relative flex flex-col items-center justify-center p-4 rounded-xl shadow-2xl transition-all"
      :style="{
        backgroundColor: style.bg_color || '#ffffff',
        border: style.border_width ? `${style.border_width}px solid ${style.border_color || '#3b82f6'}` : 'none'
      }"
    >
      <!-- Simulated or API rendered QR image -->
      <div class="w-48 h-48 relative flex items-center justify-center">
        <img
          v-if="qrImageSrc"
          :src="qrImageSrc"
          alt="QR Code Preview"
          class="w-full h-full object-contain"
        />
        <div v-else class="w-full h-full flex flex-col items-center justify-center text-gray-400 text-xs">
          <QrCode class="w-12 h-12 mb-2 opacity-50" />
          <span>Enter payload to preview</span>
        </div>

        <!-- Center Logo Overlay -->
        <div
          v-if="style.logo_base64"
          class="absolute w-10 h-10 rounded-full p-1 shadow-md flex items-center justify-center overflow-hidden"
          :style="{ backgroundColor: style.bg_color || '#ffffff' }"
        >
          <img :src="style.logo_base64" class="w-full h-full object-contain rounded-full" />
        </div>
      </div>

      <!-- Frame Label -->
      <div
        v-if="style.label_text"
        class="mt-2 text-xs font-bold tracking-wider uppercase text-center"
        :style="{ color: style.label_color || style.fg_color || '#000000' }"
      >
        {{ style.label_text }}
      </div>
    </div>

    <!-- Export Buttons (Requirement 5) -->
    <div v-if="qrId" class="mt-6 w-full space-y-2">
      <div class="text-xs font-semibold text-gray-400 text-center mb-2">Export High Resolution Formats</div>
      <div class="grid grid-cols-2 gap-2">
        <a
          :href="`/api/v1/qr/${qrId}/export?format=png&download=true`"
          class="py-2 px-3 bg-gray-700 hover:bg-gray-600 text-white rounded-lg text-xs font-medium text-center transition-colors flex items-center justify-center gap-1.5"
        >
          <Download class="w-3.5 h-3.5 text-lynqr-400" />
          <span>PNG (HD)</span>
        </a>
        <a
          :href="`/api/v1/qr/${qrId}/export?format=jpeg&download=true`"
          class="py-2 px-3 bg-gray-700 hover:bg-gray-600 text-white rounded-lg text-xs font-medium text-center transition-colors flex items-center justify-center gap-1.5"
        >
          <Download class="w-3.5 h-3.5 text-amber-400" />
          <span>JPEG</span>
        </a>
        <a
          :href="`/api/v1/qr/${qrId}/export?format=pdf&download=true`"
          class="py-2 px-3 bg-gray-700 hover:bg-gray-600 text-white rounded-lg text-xs font-medium text-center transition-colors flex items-center justify-center gap-1.5"
        >
          <FileText class="w-3.5 h-3.5 text-rose-400" />
          <span>Print PDF</span>
        </a>
        <a
          :href="`/api/v1/qr/${qrId}/export?format=svg&download=true`"
          class="py-2 px-3 bg-gray-700 hover:bg-gray-600 text-white rounded-lg text-xs font-medium text-center transition-colors flex items-center justify-center gap-1.5"
        >
          <Download class="w-3.5 h-3.5 text-emerald-400" />
          <span>SVG Vector</span>
        </a>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Eye, QrCode, Download, FileText } from 'lucide-vue-next'

const props = defineProps({
  qrId: { type: String, default: null },
  payload: { type: String, default: '' },
  style: {
    type: Object,
    default: () => ({
      fg_color: '#000000',
      bg_color: '#ffffff',
      border_width: 0,
      border_color: '#3b82f6',
      label_text: '',
      logo_base64: ''
    })
  }
})

const qrImageSrc = computed(() => {
  if (props.qrId) {
    return `/api/v1/qr/${props.qrId}/export?format=png&size=300`
  }
  if (!props.payload) return null
  // Quick fallback online preview generator for live typing
  const clean = encodeURIComponent(props.payload)
  const fg = props.style.fg_color?.replace('#', '') || '000000'
  const bg = props.style.bg_color?.replace('#', '') || 'ffffff'
  return `https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=${clean}&color=${fg}&bgcolor=${bg}&format=png`
})
</script>
