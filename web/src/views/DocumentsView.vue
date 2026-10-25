<template>
  <div>
    <PageHeader title="Documents" subtitle="Host menus, brochures and reports behind a QR code. Replace the file any time; the QR code never changes.">
      <template #title-extra><HelpLink guide="documents" /></template>
      <template #actions>
        <button v-if="canEdit && selected.length" class="btn-danger-ghost" data-testid="delete-selected" @click="askDelete(selected)">
          <Trash2 class="w-4 h-4" /> Delete selected <span class="badge-danger">{{ selected.length }}</span>
        </button>
        <button v-if="canEdit" class="btn-primary" data-testid="upload-document" @click="uploadInput.click()">
          <UploadCloud class="w-4 h-4" /> Upload
        </button>
      </template>
    </PageHeader>

    <input ref="uploadInput" type="file" class="hidden" :accept="accept" @change="onUpload" />
    <input ref="replaceInput" type="file" class="hidden" :accept="accept" @change="onReplace" />

    <div class="card overflow-hidden">
      <div v-if="store.loading && !store.documents.length" class="space-y-3 p-5"><div v-for="i in 3" :key="i" class="skeleton h-12" /></div>

      <EmptyState v-else-if="!store.documents.length" :icon="FileText" title="No documents yet" description="Upload a PDF, image menu or presentation and Lynqr generates a dynamic QR code for it.">
        <button v-if="canEdit" class="btn-primary" @click="uploadInput.click()"><UploadCloud class="w-4 h-4" /> Upload a document</button>
      </EmptyState>

      <table v-else class="table-base" data-testid="documents-table">
        <thead>
          <tr>
            <th v-if="canEdit" class="w-10">
              <input type="checkbox" class="h-4 w-4 rounded border-line accent-brand-500" :checked="allSelected" aria-label="Select all documents" @change="toggleAll" />
            </th>
            <th>Document</th>
            <th class="hidden sm:table-cell">Version</th>
            <th class="hidden md:table-cell">Size</th>
            <th class="hidden md:table-cell">QR codes</th>
            <th class="hidden lg:table-cell">Updated</th>
            <th class="w-px"><span class="sr-only">Actions</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="doc in store.documents" :key="doc.id">
            <td v-if="canEdit">
              <input v-model="selected" type="checkbox" :value="doc.id" class="h-4 w-4 rounded border-line accent-brand-500" :aria-label="`Select ${doc.filename}`" />
            </td>
            <td class="max-w-[280px]">
              <div class="flex items-center gap-3">
                <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg" :class="iconFor(doc).cls">
                  <component :is="iconFor(doc).icon" class="h-4 w-4" />
                </div>
                <div class="min-w-0">
                  <p class="truncate font-medium text-fg" :title="doc.filename">{{ doc.filename }}</p>
                  <p class="truncate text-xs text-fg-subtle">{{ doc.mime_type }}</p>
                </div>
              </div>
            </td>
            <td class="hidden sm:table-cell"><span class="badge-brand">v{{ doc.version }}</span></td>
            <td class="hidden md:table-cell text-fg-muted tabular-nums">{{ formatBytes(doc.file_size) }}</td>
            <td class="hidden md:table-cell text-fg-muted">{{ qrCount(doc) }}</td>
            <td class="hidden lg:table-cell whitespace-nowrap text-fg-muted">{{ formatDate(doc.updated_at) }}</td>
            <td>
              <div class="flex items-center justify-end gap-0.5">
                <Tip text="Open file"><a :href="fileUrl(doc)" target="_blank" rel="noopener" class="btn-ghost btn-icon" aria-label="Open file"><ExternalLink class="w-4 h-4" /></a></Tip>
                <template v-if="canEdit">
                  <Tip text="Upload a new version (QR codes keep working)">
                    <button class="btn-ghost btn-icon" aria-label="Replace file" data-testid="replace-document" @click="startReplace(doc)"><RefreshCw class="w-4 h-4" /></button>
                  </Tip>
                  <Tip text="Create another QR code for this file">
                    <button class="btn-ghost btn-icon" aria-label="Create QR code" @click="createQR(doc)"><QrCode class="w-4 h-4" /></button>
                  </Tip>
                  <Tip text="Delete"><button class="btn-ghost btn-icon hover:!text-rose-500" aria-label="Delete document" @click="askDelete([doc.id])"><Trash2 class="w-4 h-4" /></button></Tip>
                </template>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Delete modal -->
    <AppModal :open="deleteIds.length > 0" size="sm" :title="deleteIds.length > 1 ? `Delete ${deleteIds.length} documents?` : 'Delete document?'" @close="deleteIds = []">
      <p class="text-sm text-fg-muted">The file is removed from storage immediately and its download link stops working.</p>
      <label class="mt-4 flex items-start justify-between gap-4 rounded-xl border border-line px-3.5 py-3">
        <span class="text-sm">
          <span class="font-medium text-fg">Also delete linked QR codes</span>
          <span class="block text-xs text-fg-muted">Otherwise they remain but point at a removed file.</span>
        </span>
        <Toggle v-model="deleteQR" />
      </label>
      <template #footer>
        <button class="btn-secondary" @click="deleteIds = []">Cancel</button>
        <button class="btn-danger" :disabled="deleting" data-testid="confirm-delete-documents" @click="executeDelete">
          <Loader2 v-if="deleting" class="w-4 h-4 animate-spin" /> Delete
        </button>
      </template>
    </AppModal>
  </div>
</template>

<script setup>
import HelpLink from '../components/ui/HelpLink.vue'
import { ref, computed } from 'vue'
import {
  FileText, UploadCloud, RefreshCw, Trash2, QrCode, ExternalLink, Loader2, FileImage, FileSpreadsheet, Presentation
} from 'lucide-vue-next'
import { useLynqrStore } from '../stores/lynqr'
import { useAuthStore } from '../stores/auth'
import { api } from '../api/client'
import { useToast } from '../composables/useToast'
import PageHeader from '../components/ui/PageHeader.vue'
import EmptyState from '../components/ui/EmptyState.vue'
import AppModal from '../components/ui/AppModal.vue'
import Tip from '../components/ui/Tip.vue'
import Toggle from '../components/ui/Toggle.vue'

const store = useLynqrStore()
const auth = useAuthStore()
const toast = useToast()
const canEdit = computed(() => auth.can('assets:edit'))
const accept = '.pdf,.doc,.docx,.ppt,.pptx,.xls,.xlsx,.csv,.txt,.png,.jpg,.jpeg,.gif,.webp'
const MAX_BYTES = 15 * 1024 * 1024

const uploadInput = ref(null)
const replaceInput = ref(null)
const replacing = ref(null)
const selected = ref([])
const deleteIds = ref([])
const deleteQR = ref(true)
const deleting = ref(false)

const allSelected = computed(() => store.documents.length > 0 && selected.value.length === store.documents.length)
function toggleAll() {
  selected.value = allSelected.value ? [] : store.documents.map((d) => d.id)
}

const fileUrl = (doc) => `/api/v1/documents/files/${doc.stored_path}`
const qrCount = (doc) => store.qrCodes.filter((q) => q.document_id === doc.id).length

function iconFor(doc) {
  const m = doc.mime_type || ''
  if (m.startsWith('image/')) return { icon: FileImage, cls: 'bg-sky-500/10 text-sky-500' }
  if (m.includes('sheet') || m.includes('csv') || m.includes('excel')) return { icon: FileSpreadsheet, cls: 'bg-emerald-500/10 text-emerald-500' }
  if (m.includes('presentation') || m.includes('powerpoint')) return { icon: Presentation, cls: 'bg-amber-500/10 text-amber-500' }
  return { icon: FileText, cls: 'bg-rose-500/10 text-rose-500' }
}

function formatBytes(n) {
  if (n < 1024) return `${n} B`
  if (n < 1048576) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1048576).toFixed(1)} MB`
}
function formatDate(d) {
  return new Date(d).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' })
}

function tooBig(file) {
  if (file.size > MAX_BYTES) {
    toast.error('Files must be 15 MB or smaller')
    return true
  }
  return false
}

async function onUpload(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file || tooBig(file)) return
  try {
    await api.uploadDocument(file)
    toast.success(`${file.name} uploaded with a dynamic QR code`, { title: 'Document uploaded' })
    await Promise.all([store.fetchDocuments(), store.fetchQRCodes()])
  } catch (err) {
    toast.error(err.message)
  }
}

function startReplace(doc) {
  replacing.value = doc
  replaceInput.value.click()
}

async function onReplace(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file || !replacing.value || tooBig(file)) return
  try {
    const res = await api.updateDocument(replacing.value.id, file)
    toast.success(`Now serving version ${res.document.version}. Existing QR codes already point at it.`, { title: 'Document replaced' })
    await Promise.all([store.fetchDocuments(), store.fetchQRCodes()])
  } catch (err) {
    toast.error(err.message)
  } finally {
    replacing.value = null
  }
}

async function createQR(doc) {
  try {
    await api.createQRCode({
      type: 'document',
      title: `Doc: ${doc.filename}`,
      document_id: doc.id
    })
    toast.success(`QR code created for ${doc.filename}`)
    await store.fetchQRCodes()
  } catch (err) {
    toast.error(err.message)
  }
}

function askDelete(ids) {
  deleteIds.value = [...ids]
}

async function executeDelete() {
  deleting.value = true
  try {
    const res = await api.bulkDeleteDocuments(deleteIds.value, deleteQR.value)
    const qrNote = res.deleted_qr_count ? ` and ${res.deleted_qr_count} QR code(s)` : ''
    toast.success(`Deleted ${res.deleted_docs_count} document(s)${qrNote}`)
    selected.value = selected.value.filter((id) => !deleteIds.value.includes(id))
    deleteIds.value = []
    await Promise.all([store.fetchDocuments(), store.fetchQRCodes()])
  } catch (err) {
    toast.error(err.message)
  } finally {
    deleting.value = false
  }
}
</script>
