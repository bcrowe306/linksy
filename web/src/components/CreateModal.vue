<template>
  <div v-if="isOpen" class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <!-- Backdrop -->
    <div @click="$emit('close')" class="absolute inset-0 bg-black/75 backdrop-blur-sm transition-opacity"></div>

    <!-- Modal Content -->
    <div class="relative bg-gray-900 border border-gray-800 rounded-2xl w-full max-w-2xl max-h-[90vh] overflow-y-auto shadow-2xl z-10 flex flex-col">
      <!-- Modal Header -->
      <div class="px-6 py-4 border-b border-gray-800 flex items-center justify-between sticky top-0 bg-gray-900/95 backdrop-blur z-20">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-lg bg-linksy-600/20 text-linksy-400 flex items-center justify-center">
            <Plus class="w-4 h-4" />
          </div>
          <h2 class="text-lg font-bold text-white">Create New Asset</h2>
        </div>
        <button @click="$emit('close')" class="text-gray-400 hover:text-white p-1 rounded-lg hover:bg-gray-800">
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Mode Selector Tabs -->
      <div class="p-6 pb-2">
        <div class="flex rounded-xl bg-gray-800/80 p-1 border border-gray-750">
          <button
            @click="activeTab = 'link'"
            class="flex-1 py-2 text-xs font-semibold rounded-lg flex items-center justify-center gap-2 transition-all"
            :class="activeTab === 'link' ? 'bg-linksy-600 text-white shadow-md' : 'text-gray-400 hover:text-gray-200'"
          >
            <Link2 class="w-4 h-4" />
            <span>Short Link</span>
          </button>
          <button
            @click="activeTab = 'qr'"
            class="flex-1 py-2 text-xs font-semibold rounded-lg flex items-center justify-center gap-2 transition-all"
            :class="activeTab === 'qr' ? 'bg-linksy-600 text-white shadow-md' : 'text-gray-400 hover:text-gray-200'"
          >
            <QrCode class="w-4 h-4" />
            <span>Custom QR Code</span>
          </button>
          <button
            @click="activeTab = 'doc'"
            class="flex-1 py-2 text-xs font-semibold rounded-lg flex items-center justify-center gap-2 transition-all"
            :class="activeTab === 'doc' ? 'bg-linksy-600 text-white shadow-md' : 'text-gray-400 hover:text-gray-200'"
          >
            <FileUp class="w-4 h-4" />
            <span>Document QR</span>
          </button>
        </div>
      </div>

      <!-- Form Body -->
      <div class="p-6 pt-3 space-y-4 flex-1">
        <!-- Error Alert -->
        <div v-if="errorMessage" class="p-3 rounded-lg bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs flex items-center gap-2">
          <AlertCircle class="w-4 h-4 shrink-0" />
          <span>{{ errorMessage }}</span>
        </div>

        <!-- 1. SHORT LINK FORM -->
        <div v-if="activeTab === 'link'" class="space-y-4">
          <div>
            <label class="block text-xs font-semibold text-gray-300 mb-1">Destination URL *</label>
            <input
              v-model="linkForm.target_url"
              type="url"
              placeholder="https://example.com/very-long-url-to-shorten"
              class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3.5 py-2 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-linksy-500"
            />
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-1">Title (Optional)</label>
              <input
                v-model="linkForm.title"
                type="text"
                placeholder="Marketing Promo Link"
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3.5 py-2 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-linksy-500"
              />
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-1">Custom Slug (Optional)</label>
              <div class="flex items-center">
                <span class="text-xs text-gray-500 bg-gray-800 px-2 py-2 border border-r-0 border-gray-700 rounded-l-lg font-mono">/s/</span>
                <input
                  v-model="linkForm.custom_slug"
                  type="text"
                  placeholder="launch2026"
                  class="w-full bg-gray-800 border border-gray-700 rounded-r-lg px-3 py-2 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-linksy-500 font-mono"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- 2. CUSTOM QR CODE FORM -->
        <div v-else-if="activeTab === 'qr'" class="space-y-4">
          <!-- QR Type Selection -->
          <div>
            <label class="block text-xs font-semibold text-gray-300 mb-1">QR Code Type</label>
            <select
              v-model="qrForm.type"
              @change="onQRTypeChange"
              class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-linksy-500"
            >
              <option value="website">Website / URL (Dynamic)</option>
              <option value="document">Document (PDF / Menu / Report) (Dynamic)</option>
              <option value="vcard">Contact Card (vCard)</option>
              <option value="wifi">Wi-Fi Network</option>
              <option value="email">Email Pre-fill</option>
              <option value="sms">SMS Text Message</option>
              <option value="text">Plain Text</option>
            </select>
          </div>

          <div>
            <label class="block text-xs font-semibold text-gray-300 mb-1">QR Title *</label>
            <input
              v-model="qrForm.title"
              type="text"
              placeholder="e.g. Table 5 Menu or Guest Wi-Fi"
              class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-linksy-500"
            />
          </div>

          <!-- Dynamic inputs based on QR type -->
          <div v-if="qrForm.type === 'website'">
            <label class="block text-xs font-semibold text-gray-300 mb-1">Target URL *</label>
            <input
              v-model="qrForm.target_url"
              type="url"
              placeholder="https://mywebsite.com"
              class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-linksy-500"
            />
          </div>

          <!-- Document QR Code: Select from existing uploaded documents or upload new -->
          <div v-else-if="qrForm.type === 'document'" class="space-y-3">
            <div class="flex items-center justify-between">
              <label class="block text-xs font-semibold text-gray-300">Document Source</label>
              <div class="flex rounded-lg bg-gray-800 p-0.5 border border-gray-700 text-[11px]">
                <button
                  type="button"
                  @click="docSource = 'existing'"
                  :class="docSource === 'existing' ? 'bg-linksy-600 text-white font-semibold' : 'text-gray-400 hover:text-white'"
                  class="px-2.5 py-1 rounded transition-colors"
                >
                  Choose Existing ({{ store.documents.length }})
                </button>
                <button
                  type="button"
                  @click="docSource = 'new'"
                  :class="docSource === 'new' ? 'bg-linksy-600 text-white font-semibold' : 'text-gray-400 hover:text-white'"
                  class="px-2.5 py-1 rounded transition-colors"
                >
                  Upload New
                </button>
              </div>
            </div>

            <!-- Existing Documents Picker -->
            <div v-if="docSource === 'existing'">
              <div v-if="store.documents.length > 0" class="space-y-2">
                <div class="text-[11px] text-gray-400">Select from your uploaded documents:</div>
                <div class="max-h-48 overflow-y-auto space-y-1.5 pr-1 border border-gray-750 rounded-xl p-2 bg-gray-800/40">
                  <div
                    v-for="doc in store.documents"
                    :key="doc.id"
                    @click="selectDocument(doc)"
                    class="p-2.5 rounded-lg border cursor-pointer transition-all flex items-center justify-between"
                    :class="selectedExistingDocId === doc.id
                      ? 'bg-linksy-600/20 border-linksy-500 text-white'
                      : 'bg-gray-800/70 border-gray-700 hover:border-gray-600 text-gray-300'"
                  >
                    <div class="flex items-center gap-2.5 truncate max-w-[340px]">
                      <FileText class="w-4 h-4 text-rose-400 shrink-0" />
                      <div class="truncate">
                        <div class="font-medium text-xs truncate">{{ doc.filename }}</div>
                        <div class="text-[10px] text-gray-500 font-mono">
                          v{{ doc.version }} • {{ (doc.file_size / 1024 / 1024).toFixed(2) }} MB • {{ new Date(doc.updated_at).toLocaleDateString() }}
                        </div>
                      </div>
                    </div>
                    <div class="shrink-0 pl-2">
                      <span
                        v-if="selectedExistingDocId === doc.id"
                        class="w-5 h-5 rounded-full bg-linksy-500 text-white flex items-center justify-center"
                      >
                        <Check class="w-3 h-3 stroke-[3]" />
                      </span>
                      <span v-else class="text-xs text-gray-500 hover:text-gray-300">Select</span>
                    </div>
                  </div>
                </div>
                <div v-if="selectedDocumentObj" class="p-2.5 rounded-lg bg-linksy-950/40 border border-linksy-500/30 text-xs text-linksy-300 flex items-center gap-2">
                  <CheckCircle2 class="w-4 h-4 text-linksy-400 shrink-0" />
                  <span class="truncate">Linked to <strong>{{ selectedDocumentObj.filename }}</strong> (v{{ selectedDocumentObj.version }})</span>
                </div>
              </div>
              <div v-else class="p-4 rounded-xl bg-gray-800/40 border border-dashed border-gray-700 text-center space-y-2">
                <FileText class="w-8 h-8 text-gray-500 mx-auto" />
                <div class="text-xs text-gray-300 font-medium">No documents uploaded yet</div>
                <p class="text-[11px] text-gray-500">Upload your first document to reuse across QR codes.</p>
                <button
                  type="button"
                  @click="docSource = 'new'"
                  class="px-3 py-1.5 rounded-lg bg-linksy-600 hover:bg-linksy-500 text-white text-xs font-semibold inline-flex items-center gap-1.5"
                >
                  <FileUp class="w-3.5 h-3.5" />
                  <span>Upload Document Now</span>
                </button>
              </div>
            </div>

            <!-- Upload New Document in QR Form -->
            <div v-else class="space-y-2">
              <div
                @dragover.prevent
                @drop.prevent="handleDocFileDrop"
                class="border-2 border-dashed border-gray-700 hover:border-linksy-500 rounded-xl p-6 flex flex-col items-center justify-center text-center cursor-pointer transition-colors"
                @click="$refs.qrDocFileInput.click()"
              >
                <input ref="qrDocFileInput" type="file" @change="handleDocFileSelect" class="hidden" accept=".pdf,.doc,.docx,.png,.jpg,.jpeg" />
                <FileUp class="w-8 h-8 text-linksy-400 mb-1.5" />
                <span class="text-xs font-semibold text-gray-200">
                  {{ qrDocFile ? qrDocFile.name : 'Choose document file to link' }}
                </span>
                <span class="text-[10px] text-gray-500 mt-0.5">PDF, Menus, Documents up to 15MB</span>
              </div>
              <div v-if="qrDocFile" class="text-xs text-emerald-400 flex items-center gap-1.5 justify-center">
                <Check class="w-3.5 h-3.5" />
                <span>{{ qrDocFile.name }} ({{ (qrDocFile.size / 1024 / 1024).toFixed(2) }} MB)</span>
              </div>
            </div>
          </div>

          <div v-else-if="qrForm.type === 'wifi'" class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-1">SSID Network Name</label>
              <input v-model="wifiData.ssid" type="text" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-1">Password</label>
              <input v-model="wifiData.password" type="text" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
            </div>
          </div>

          <div v-else-if="qrForm.type === 'vcard'" class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-1">First Name</label>
              <input v-model="vcardData.first_name" type="text" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-1">Last Name</label>
              <input v-model="vcardData.last_name" type="text" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-1">Phone</label>
              <input v-model="vcardData.phone" type="tel" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-300 mb-1">Email</label>
              <input v-model="vcardData.email" type="email" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
            </div>
          </div>

          <!-- Styling customization controls -->
          <div class="border-t border-gray-800 pt-3">
            <span class="text-xs font-semibold text-gray-400 block mb-2">Style & Colors</span>
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <div>
                <label class="block text-[11px] text-gray-400 mb-1">Foreground</label>
                <div class="flex items-center gap-2">
                  <input v-model="qrForm.style.fg_color" type="color" class="w-8 h-8 rounded border-0 bg-transparent cursor-pointer" />
                  <span class="text-xs font-mono text-gray-300">{{ qrForm.style.fg_color }}</span>
                </div>
              </div>
              <div>
                <label class="block text-[11px] text-gray-400 mb-1">Background</label>
                <div class="flex items-center gap-2">
                  <input v-model="qrForm.style.bg_color" type="color" class="w-8 h-8 rounded border-0 bg-transparent cursor-pointer" />
                  <span class="text-xs font-mono text-gray-300">{{ qrForm.style.bg_color }}</span>
                </div>
              </div>
              <div>
                <label class="block text-[11px] text-gray-400 mb-1">Border (px)</label>
                <input v-model.number="qrForm.style.border_width" type="number" min="0" max="10" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-2 py-1 text-xs text-gray-100" />
              </div>
              <div>
                <label class="block text-[11px] text-gray-400 mb-1">Label Tag</label>
                <input v-model="qrForm.style.label_text" type="text" placeholder="SCAN ME" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-2 py-1 text-xs text-gray-100" />
              </div>
            </div>
          </div>
        </div>

        <!-- 3. DOCUMENT UPLOAD TAB -->
        <div v-else-if="activeTab === 'doc'" class="space-y-4">
          <div
            @dragover.prevent
            @drop.prevent="handleFileDrop"
            class="border-2 border-dashed border-gray-700 hover:border-linksy-500 rounded-2xl p-8 flex flex-col items-center justify-center text-center cursor-pointer transition-colors"
            @click="$refs.fileInput.click()"
          >
            <input ref="fileInput" type="file" @change="handleFileSelect" class="hidden" accept=".pdf,.doc,.docx,.png,.jpg,.jpeg" />
            <FileUp class="w-10 h-10 text-linksy-400 mb-2" />
            <span class="text-sm font-semibold text-gray-200">
              {{ selectedFile ? selectedFile.name : 'Choose a file or drag & drop here' }}
            </span>
            <span class="text-xs text-gray-500 mt-1">PDF, Docs, Menus, Presentations up to 15MB</span>
          </div>
          <div v-if="selectedFile" class="text-xs text-emerald-400 flex items-center gap-1.5 justify-center">
            <Check class="w-4 h-4" />
            <span>Ready to upload ({{ (selectedFile.size / 1024 / 1024).toFixed(2) }} MB)</span>
          </div>
        </div>
      </div>

      <!-- Footer Buttons -->
      <div class="px-6 py-4 bg-gray-900 border-t border-gray-800 flex items-center justify-end gap-3 sticky bottom-0">
        <button
          @click="$emit('close')"
          class="px-4 py-2 rounded-lg text-sm font-medium text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
        >
          Cancel
        </button>
        <button
          @click="submit"
          :disabled="isSubmitting"
          class="px-5 py-2 rounded-lg text-sm font-semibold bg-linksy-600 hover:bg-linksy-500 text-white shadow-lg shadow-linksy-600/25 transition-all flex items-center gap-2"
        >
          <Loader2 v-if="isSubmitting" class="w-4 h-4 animate-spin" />
          <span>{{ isSubmitting ? 'Creating...' : 'Create Asset' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import {
  Plus,
  X,
  Link2,
  QrCode,
  FileUp,
  FileText,
  AlertCircle,
  Check,
  CheckCircle2,
  Loader2
} from 'lucide-vue-next'
import { api } from '../api/client'
import { useLinksyStore } from '../stores/linksy'

const props = defineProps({
  isOpen: { type: Boolean, default: false }
})
const emit = defineEmits(['close', 'created'])
const store = useLinksyStore()

const activeTab = ref('link')
const isSubmitting = ref(false)
const errorMessage = ref('')

const linkForm = ref({
  target_url: '',
  title: '',
  custom_slug: ''
})

const qrForm = ref({
  type: 'website',
  title: '',
  target_url: '',
  is_dynamic: true,
  style: {
    fg_color: '#000000',
    bg_color: '#ffffff',
    border_width: 0,
    border_color: '#3b82f6',
    label_text: ''
  }
})

// Document QR selection state
const docSource = ref('existing') // 'existing' | 'new'
const selectedExistingDocId = ref('')
const qrDocFile = ref(null)

const wifiData = ref({ ssid: '', password: '', encryption: 'WPA' })
const vcardData = ref({ first_name: '', last_name: '', phone: '', email: '' })
const selectedFile = ref(null)

const selectedDocumentObj = computed(() => {
  return store.documents.find(d => d.id === selectedExistingDocId.value)
})

watch(() => props.isOpen, (newVal) => {
  if (newVal) {
    store.fetchDocuments()
    // Default to existing if documents exist
    if (store.documents.length > 0 && !selectedExistingDocId.value) {
      selectDocument(store.documents[0])
    }
  }
})

function onQRTypeChange() {
  if (qrForm.value.type === 'document') {
    docSource.value = store.documents.length > 0 ? 'existing' : 'new'
    if (store.documents.length > 0 && !selectedExistingDocId.value) {
      selectDocument(store.documents[0])
    }
  }
}

function selectDocument(doc) {
  selectedExistingDocId.value = doc.id
  qrForm.value.target_url = `${window.location.origin}/api/v1/documents/files/${doc.stored_path}`
  if (!qrForm.value.title || qrForm.value.title.startsWith('Doc: ') || qrForm.value.title.startsWith('QR: ')) {
    qrForm.value.title = `Doc: ${doc.filename}`
  }
}

function handleDocFileSelect(e) {
  if (e.target.files?.[0]) {
    qrDocFile.value = e.target.files[0]
    if (!qrForm.value.title) {
      qrForm.value.title = `Doc: ${qrDocFile.value.name}`
    }
  }
}

function handleDocFileDrop(e) {
  if (e.dataTransfer.files?.[0]) {
    qrDocFile.value = e.dataTransfer.files[0]
    if (!qrForm.value.title) {
      qrForm.value.title = `Doc: ${qrDocFile.value.name}`
    }
  }
}

function handleFileSelect(e) {
  if (e.target.files?.[0]) selectedFile.value = e.target.files[0]
}

function handleFileDrop(e) {
  if (e.dataTransfer.files?.[0]) selectedFile.value = e.dataTransfer.files[0]
}

async function submit() {
  errorMessage.value = ''
  isSubmitting.value = true

  try {
    if (activeTab.value === 'link') {
      if (!linkForm.value.target_url) throw new Error('Please enter a destination URL')
      await api.createLink(linkForm.value)
    } else if (activeTab.value === 'qr') {
      if (!qrForm.value.title) throw new Error('Please enter a title for the QR code')

      let rawData = ''
      if (qrForm.value.type === 'wifi') {
        rawData = JSON.stringify(wifiData.value)
      } else if (qrForm.value.type === 'vcard') {
        rawData = JSON.stringify(vcardData.value)
      } else if (qrForm.value.type === 'document') {
        qrForm.value.is_dynamic = true
        if (docSource.value === 'existing') {
          if (!selectedExistingDocId.value) {
            throw new Error('Please select an uploaded document from the list')
          }
          const doc = store.documents.find(d => d.id === selectedExistingDocId.value)
          if (!doc) throw new Error('Selected document not found')
          qrForm.value.target_url = `${window.location.origin}/api/v1/documents/files/${doc.stored_path}`
        } else {
          if (!qrDocFile.value) {
            throw new Error('Please choose a document file to upload')
          }
          const uploadRes = await api.uploadDocument(qrDocFile.value)
          qrForm.value.target_url = uploadRes.download_url
        }
      }

      await api.createQRCode({
        ...qrForm.value,
        raw_data: rawData
      })
    } else if (activeTab.value === 'doc') {
      if (!selectedFile.value) throw new Error('Please choose a document to upload')
      await api.uploadDocument(selectedFile.value)
    }

    await store.refreshAll()
    emit('created')
    emit('close')
  } catch (err) {
    errorMessage.value = err.message || 'An error occurred'
  } finally {
    isSubmitting.value = false
  }
}
</script>
