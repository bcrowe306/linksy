<template>
  <div>
    <!-- Header -->
    <div class="mb-6 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex min-w-0 items-center gap-3">
        <router-link to="/qr-codes" class="btn-ghost btn-icon" aria-label="Back to QR codes"><ArrowLeft class="h-4 w-4" /></router-link>
        <div class="min-w-0 flex-1">
          <input
            v-model="title"
            class="w-full min-w-0 truncate bg-transparent text-2xl font-bold tracking-tight text-fg outline-none placeholder:text-fg-subtle focus:ring-0"
            placeholder="Name your QR code (e.g. Summer Campaign poster)"
            aria-label="QR code name"
            maxlength="120"
            :disabled="!canEdit"
            data-testid="qr-title-input"
          />
          <div class="mt-1 flex items-center gap-2 text-xs text-fg-muted">
            <span>{{ isEditing ? 'Editing' : 'New QR code' }}</span>
            <span v-if="isDirty" class="badge-warning" data-testid="unsaved-badge">Unsaved Changes</span>
            <span v-else-if="isEditing" class="badge-success"><Check class="h-3 w-3" /> Saved</span>
            <span v-if="existing?.scan_count !== undefined" class="text-fg-subtle">· {{ existing.scan_count }} scans</span>
          </div>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <router-link v-if="isEditing" :to="`/analytics?qr=${route.params.id}`" class="btn-secondary"><BarChart3 class="h-4 w-4" /> Analytics</router-link>
        <button v-if="isEditing && canEdit" class="btn-secondary" :disabled="saving" @click="duplicate"><Copy class="h-4 w-4" /> Duplicate</button>
        <button v-if="canEdit" class="btn-primary" :disabled="saving || !!contentError || !title.trim()" data-testid="qr-save" @click="save">
          <Loader2 v-if="saving" class="h-4 w-4 animate-spin" /><Save v-else class="h-4 w-4" />
          {{ isEditing ? 'Save changes' : 'Create QR code' }}
        </button>
      </div>
    </div>

    <div v-if="loadError" class="card p-6 text-sm text-rose-500">{{ loadError }}</div>

    <div v-else class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_380px] xl:grid-cols-[minmax(0,1fr)_420px]">
      <div class="space-y-6">
        <section class="card p-5 sm:p-6">
          <h2 class="mb-4 flex items-center gap-2 text-sm font-semibold text-fg"><span class="flex h-6 w-6 items-center justify-center rounded-full bg-brand-500/10 text-xs text-brand-500">1</span> Content</h2>
          <QRTypeFields v-model="content" :lock-type="isEditing && existing?.is_dynamic" />
          <div v-if="isEditing && existing && !existing.is_dynamic" class="mt-4 flex gap-2 rounded-xl bg-amber-500/10 px-3 py-2.5 text-xs text-amber-700 dark:text-amber-300">
            <AlertTriangle class="h-4 w-4 shrink-0" />
            This is a static code. Changing its content changes the printed pattern, so existing prints keep the old content.
          </div>
          <div v-if="existing?.scan_url" class="mt-4 flex items-center gap-2 rounded-xl border border-line px-3 py-2">
            <Zap class="h-4 w-4 text-brand-500" />
            <span class="text-xs text-fg-muted">Scan URL</span>
            <span class="truncate font-mono text-xs text-fg">{{ existing.scan_url }}</span>
            <CopyButton :value="existing.scan_url" label="Copy scan URL" class="ml-auto" />
          </div>
        </section>

        <section class="card p-5 sm:p-6">
          <h2 class="mb-4 flex items-center gap-2 text-sm font-semibold text-fg"><span class="flex h-6 w-6 items-center justify-center rounded-full bg-brand-500/10 text-xs text-brand-500">2</span> Design</h2>
          <fieldset :disabled="!canEdit"><QRDesignPanel v-model="style" /></fieldset>
        </section>
      </div>

      <!-- Preview & export -->
      <aside class="lg:sticky lg:top-24 lg:self-start">
        <section class="card p-5">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-sm font-semibold text-fg">Live preview</h2>
            <span v-if="contentError" class="text-[11px] text-fg-subtle">Sample content shown</span>
          </div>
          <QRLivePreview :content="previewContent" :style="style" :qr-id="isEditing ? route.params.id : ''" @result="lastPreview = $event" />
          <p v-if="contentError" class="mt-3 text-xs text-amber-600 dark:text-amber-400">{{ contentError }}</p>

          <div class="mt-5 border-t border-line pt-5">
            <h3 class="mb-3 text-xs font-semibold uppercase tracking-wider text-fg-subtle">Download</h3>
            <div class="grid grid-cols-2 gap-2">
              <button v-for="f in formats" :key="f.id" class="btn-secondary justify-start" :data-testid="`export-${f.id}`" @click="download(f)">
                <component :is="f.icon" class="h-4 w-4 text-fg-subtle" />
                <span class="text-left"><span class="block text-sm">{{ f.label }}</span><span class="block text-[10px] font-normal text-fg-subtle">{{ f.hint }}</span></span>
              </button>
            </div>
            <div class="mt-3 flex items-center gap-2">
              <label for="export-size" class="text-xs text-fg-muted">PNG/JPEG size</label>
              <select id="export-size" v-model.number="exportSize" class="input input-sm w-auto">
                <option :value="512">512 px</option><option :value="1024">1024 px</option><option :value="2048">2048 px</option><option :value="4096">4096 px (4K)</option>
              </select>
            </div>
            <p v-if="!isEditing || isDirty" class="mt-3 text-[11px] text-fg-subtle">Downloads use the saved version{{ !isEditing ? ', so create the code first' : '; save to include your latest changes' }}.</p>
          </div>
        </section>
      </aside>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import {
  ArrowLeft, Save, Loader2, Check, Copy, BarChart3, AlertTriangle, Zap, Image, FileImage, FileType2, FileText, CreditCard
} from 'lucide-vue-next'
import { api } from '../api/client'
import { useLynqrStore } from '../stores/lynqr'
import { useAuthStore } from '../stores/auth'
import { useToast } from '../composables/useToast'
import { useConfirm } from '../composables/useConfirm'
import { DEFAULT_STYLE, defaultsFor, parseRawData, rawDataFor, validateContent, QR_TYPES } from '../lib/qrContent'
import QRTypeFields from '../components/qr/QRTypeFields.vue'
import QRDesignPanel from '../components/qr/QRDesignPanel.vue'
import QRLivePreview from '../components/qr/QRLivePreview.vue'
import CopyButton from '../components/ui/CopyButton.vue'

const route = useRoute()
const router = useRouter()
const store = useLynqrStore()
const auth = useAuthStore()
const toast = useToast()
const confirm = useConfirm()

const canEdit = computed(() => auth.can('assets:edit'))
const isEditing = computed(() => !!route.params.id)
const existing = ref(null)
const loadError = ref('')
const saving = ref(false)
const lastPreview = ref(null)
const exportSize = ref(2048)

const initialType = QR_TYPES.some((t) => t.id === route.query.type) ? route.query.type : 'website'
const title = ref('')
const content = ref({ type: initialType, target_url: '', is_dynamic: true, document_id: '', data: defaultsFor(initialType) })
const style = ref({ ...DEFAULT_STYLE })

const formats = [
  { id: 'png', label: 'PNG', hint: 'Web & print', icon: Image },
  { id: 'svg', label: 'SVG', hint: 'Infinite vector', icon: FileType2 },
  { id: 'pdf', label: 'PDF sheet', hint: 'A4, print ready', icon: FileText, layout: 'sheet' },
  { id: 'pdf-card', label: 'PDF card', hint: 'Sticker / card', icon: CreditCard, format: 'pdf', layout: 'card' },
  { id: 'jpeg', label: 'JPEG', hint: 'Solid background', icon: FileImage }
]

// ----- Content helpers -----
const apiContent = computed(() => ({
  type: content.value.type,
  target_url: content.value.target_url,
  raw_data: rawDataFor(content.value.type, content.value.data),
  is_dynamic: content.value.is_dynamic,
  document_id: content.value.document_id
}))
const contentError = computed(() => validateContent(content.value))

// Until content is valid, preview the design with representative sample data.
const samples = {
  website: { target_url: 'https://example.com' },
  document: { is_dynamic: true },
  vcard: { raw_data: JSON.stringify({ first_name: 'Ada', last_name: 'Lovelace', phone: '+15550100' }) },
  wifi: { raw_data: JSON.stringify({ ssid: 'Guest', password: 'welcome123' }) },
  app: { is_dynamic: true },
  text: { raw_data: 'Hello from Lynqr' },
  email: { raw_data: JSON.stringify({ email: 'hello@example.com' }) },
  sms: { raw_data: JSON.stringify({ phone: '+15550100' }) },
  phone: { raw_data: JSON.stringify({ phone: '+15550100' }) }
}
const previewContent = computed(() => (contentError.value ? { ...apiContent.value, ...samples[content.value.type] } : apiContent.value))

// ----- Dirty tracking -----
const baseline = ref('')
const snapshot = () => JSON.stringify({ t: title.value, c: content.value, s: style.value })
const isDirty = computed(() => !!baseline.value && snapshot() !== baseline.value)
function markClean() {
  baseline.value = snapshot()
}

function onBeforeUnload(e) {
  if (isDirty.value) {
    e.preventDefault()
    e.returnValue = ''
  }
}

onBeforeRouteLeave(async () => {
  if (!isDirty.value) return true
  return confirm({
    title: 'Discard unsaved changes?',
    message: 'You have unsaved changes to this QR code. Leave without saving?',
    confirmText: 'Discard changes',
    danger: true
  })
})

// ----- Load / save -----
async function load(id) {
  loadError.value = ''
  try {
    const q = await api.getQRCode(id)
    existing.value = q
    title.value = q.title
    content.value = {
      type: q.type,
      target_url: q.target_url || '',
      is_dynamic: q.is_dynamic,
      document_id: q.document_id || '',
      data: parseRawData(q.type, q.raw_data)
    }
    style.value = { ...DEFAULT_STYLE, ...q.style }
    markClean()
  } catch (err) {
    loadError.value = err.status === 404 ? 'This QR code does not exist in the current workspace.' : err.message
  }
}

async function save() {
  if (contentError.value) return toast.error(contentError.value)
  if (!title.value.trim()) return toast.error('Give the QR code a name')
  saving.value = true
  try {
    const payload = { ...apiContent.value, title: title.value, style: style.value }
    if (isEditing.value) {
      existing.value = await api.updateQRCode(route.params.id, payload)
      markClean()
      toast.success('QR code saved')
    } else {
      const created = await api.createQRCode(payload)
      existing.value = created
      markClean()
      toast.success(created.is_dynamic ? 'Your dynamic QR code is live' : 'QR code created', { title: created.title })
      await router.replace(`/qr-studio/${created.id}`)
    }
    store.fetchQRCodes()
    store.fetchLinks()
  } catch (err) {
    toast.error(err.message)
  } finally {
    saving.value = false
  }
}

async function duplicate() {
  saving.value = true
  try {
    const copy = await api.createQRCode({ ...apiContent.value, title: `${title.value} (copy)`, style: style.value })
    markClean()
    toast.success('Duplicated')
    store.fetchQRCodes()
    await router.push(`/qr-studio/${copy.id}`)
  } catch (err) {
    toast.error(err.message)
  } finally {
    saving.value = false
  }
}

async function download(f) {
  if (!isEditing.value) {
    toast.info('Create the QR code first, then download it')
    return
  }
  if (isDirty.value) {
    const ok = await confirm({ title: 'Save before downloading?', message: 'Downloads use the saved design. Save your changes now?', confirmText: 'Save & download' })
    if (!ok) return
    await save()
    if (isDirty.value) return
  }
  const fmt = f.format || f.id
  const a = document.createElement('a')
  a.href = api.getExportUrl(route.params.id, fmt, exportSize.value, true, f.layout || '')
  document.body.appendChild(a)
  a.click()
  a.remove()
}

watch(
  () => route.params.id,
  (id) => {
    if (id) load(id)
    else {
      existing.value = null
      markClean()
    }
  }
)

onMounted(async () => {
  window.addEventListener('beforeunload', onBeforeUnload)
  store.fetchDocuments()
  if (route.params.id) await load(route.params.id)
  else markClean()
})
onBeforeUnmount(() => window.removeEventListener('beforeunload', onBeforeUnload))
</script>
