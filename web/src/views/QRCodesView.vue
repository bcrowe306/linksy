<template>
  <div>
    <PageHeader title="QR Codes" subtitle="Every QR code in this workspace. Dynamic codes can be re-pointed without reprinting.">
      <template #actions>
        <router-link v-if="canEdit" to="/qr-studio" class="btn-secondary"><Wand2 class="w-4 h-4" /> Open Studio</router-link>
        <button v-if="canEdit" class="btn-primary" @click="$emit('open-create', 'qr')"><Plus class="w-4 h-4" /> New QR code</button>
      </template>
    </PageHeader>

    <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center">
      <div class="relative flex-1 sm:max-w-xs">
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-subtle" />
        <input v-model="search" type="search" placeholder="Filter QR codes…" class="input pl-9" aria-label="Filter QR codes" />
      </div>
      <select v-model="type" class="input sm:w-48" aria-label="Filter by type">
        <option value="">All types</option>
        <option v-for="(meta, key) in typeMeta" :key="key" :value="key">{{ meta.label }}</option>
      </select>
    </div>

    <div v-if="store.loading && !store.qrCodes.length" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      <div v-for="i in 4" :key="i" class="skeleton h-72 rounded-2xl" />
    </div>

    <div v-else-if="!store.qrCodes.length" class="card">
      <EmptyState :icon="QrCode" title="No QR codes yet" description="Design a branded QR code with colors, logo and frame, then track every scan.">
        <router-link v-if="canEdit" to="/qr-studio" class="btn-primary"><Wand2 class="w-4 h-4" /> Design a QR code</router-link>
      </EmptyState>
    </div>

    <div v-else-if="!visible.length" class="card">
      <EmptyState :icon="SearchX" title="No QR codes match your filters" />
    </div>

    <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4" data-testid="qr-grid">
      <article v-for="qr in visible" :key="qr.id" class="card card-hover group flex flex-col overflow-hidden">
        <router-link :to="`/qr-studio/${qr.id}`" class="relative flex aspect-square items-center justify-center bg-[repeating-conic-gradient(rgb(var(--subtle))_0%_25%,transparent_0%_50%)] bg-[length:16px_16px] p-6">
          <img :src="previewUrl(qr)" :alt="`QR code for ${qr.title}`" loading="lazy" class="max-h-full max-w-full rounded-lg shadow-card transition-transform group-hover:scale-[1.02]" />
          <span class="absolute left-3 top-3 badge-neutral bg-surface/90 backdrop-blur">
            <component :is="typeMeta[qr.type]?.icon || QrCode" class="w-3 h-3" /> {{ typeMeta[qr.type]?.label || qr.type }}
          </span>
          <span v-if="qr.link_id" class="absolute right-3 top-3 badge-brand bg-surface/90 backdrop-blur">Dynamic</span>
        </router-link>

        <div class="flex flex-1 flex-col gap-3 p-4">
          <div class="min-w-0">
            <h3 class="truncate font-semibold text-fg" :title="qr.title">{{ qr.title || 'Untitled QR' }}</h3>
            <p class="truncate text-xs text-fg-muted" :title="qr.payload">{{ describe(qr) }}</p>
          </div>
          <div class="mt-auto flex items-center justify-between">
            <span class="inline-flex items-center gap-1.5 text-xs text-fg-muted">
              <ScanLine class="w-3.5 h-3.5" /><span class="font-semibold tabular-nums text-fg">{{ qr.scan_count.toLocaleString() }}</span> scans
            </span>
            <div class="flex items-center gap-0.5">
              <DropdownMenu>
                <template #trigger>
                  <Tip text="Download"><button class="btn-ghost btn-icon" aria-label="Download QR code"><Download class="w-4 h-4" /></button></Tip>
                </template>
                <MenuItem v-for="f in formats" :key="f.id" :icon="f.icon" @click="download(qr, f.id)">{{ f.label }}</MenuItem>
              </DropdownMenu>
              <Tip text="Analytics"><router-link :to="`/analytics?qr=${qr.id}`" class="btn-ghost btn-icon" aria-label="View analytics"><BarChart3 class="w-4 h-4" /></router-link></Tip>
              <Tip v-if="canEdit" text="Edit in Studio"><router-link :to="`/qr-studio/${qr.id}`" class="btn-ghost btn-icon" aria-label="Edit QR code"><Pencil class="w-4 h-4" /></router-link></Tip>
              <Tip v-if="canEdit" text="Delete"><button class="btn-ghost btn-icon hover:!text-rose-500" aria-label="Delete QR code" @click="remove(qr)"><Trash2 class="w-4 h-4" /></button></Tip>
            </div>
          </div>
        </div>
      </article>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import {
  Plus, QrCode, Wand2, Search, SearchX, Download, BarChart3, Pencil, Trash2, ScanLine, Globe, FileText, Contact, Wifi,
  Smartphone, Type, Mail, MessageSquare, Image, FileImage, FileType2
} from 'lucide-vue-next'
import { useLynqrStore } from '../stores/lynqr'
import { useAuthStore } from '../stores/auth'
import { api } from '../api/client'
import { useToast } from '../composables/useToast'
import { useConfirm } from '../composables/useConfirm'
import PageHeader from '../components/ui/PageHeader.vue'
import EmptyState from '../components/ui/EmptyState.vue'
import DropdownMenu from '../components/ui/DropdownMenu.vue'
import MenuItem from '../components/ui/MenuItem.vue'
import Tip from '../components/ui/Tip.vue'

defineEmits(['open-create'])
const store = useLynqrStore()
const auth = useAuthStore()
const toast = useToast()
const confirm = useConfirm()
const canEdit = computed(() => auth.can('assets:edit'))

const typeMeta = {
  website: { label: 'Website', icon: Globe },
  document: { label: 'Document', icon: FileText },
  vcard: { label: 'Contact', icon: Contact },
  wifi: { label: 'Wi-Fi', icon: Wifi },
  app: { label: 'App store', icon: Smartphone },
  text: { label: 'Text', icon: Type },
  email: { label: 'Email', icon: Mail },
  sms: { label: 'SMS', icon: MessageSquare }
}
const formats = [
  { id: 'png', label: 'PNG · 2048px', icon: Image },
  { id: 'svg', label: 'SVG · vector', icon: FileType2 },
  { id: 'pdf', label: 'PDF · print ready', icon: FileText },
  { id: 'jpeg', label: 'JPEG · 2048px', icon: FileImage }
]

const search = ref('')
const type = ref('')
const visible = computed(() => {
  const q = search.value.trim().toLowerCase()
  return store.qrCodes.filter(
    (c) => (!type.value || c.type === type.value) && (!q || [c.title, c.payload].some((f) => (f || '').toLowerCase().includes(q)))
  )
})

// Cache-bust on update so edited styles show immediately.
const previewUrl = (qr) => `${api.getExportUrl(qr.id, 'png', 360, false)}&v=${new Date(qr.updated_at).getTime()}`

function describe(qr) {
  if (qr.type === 'wifi') return 'Wi-Fi credentials'
  if (qr.type === 'vcard') return 'Contact card'
  return qr.payload
}

function download(qr, fmt) {
  const a = document.createElement('a')
  a.href = api.getExportUrl(qr.id, fmt, 2048, true)
  a.download = ''
  document.body.appendChild(a)
  a.click()
  a.remove()
}

async function remove(qr) {
  const ok = await confirm({
    title: 'Delete QR code?',
    message: `"${qr.title}" will stop resolving. Anyone scanning a printed copy will see a "not found" page.`,
    confirmText: 'Delete QR code',
    danger: true
  })
  if (!ok) return
  try {
    await api.deleteQRCode(qr.id)
    toast.success('QR code deleted')
    await store.fetchQRCodes()
  } catch (err) {
    toast.error(err.message)
  }
}
</script>
