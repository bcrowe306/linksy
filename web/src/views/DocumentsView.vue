<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold text-white">Document Sharing & Reusable QR Codes</h1>
        <p class="text-xs text-gray-400 mt-0.5">
          Upload menus, catalogs, and reports. Replace the underlying document at any time without changing the physical QR code.
        </p>
      </div>

      <div class="flex items-center gap-2.5">
        <!-- Delete Selected Button (Requirement: activate when at least one item is marked for delete) -->
        <button
          @click="openDeleteModal(selectedDocIds)"
          :disabled="selectedDocIds.length === 0"
          :class="[
            'px-3.5 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 transition-all',
            selectedDocIds.length > 0
              ? 'bg-rose-600 hover:bg-rose-500 text-white shadow-lg shadow-rose-900/30 cursor-pointer'
              : 'bg-gray-800/80 text-gray-500 border border-gray-700/60 cursor-not-allowed opacity-50'
          ]"
          title="Delete selected documents"
        >
          <Trash2 class="w-4 h-4" />
          <span>Delete Selected</span>
          <span
            v-if="selectedDocIds.length > 0"
            class="px-1.5 py-0.5 rounded-full bg-rose-700/80 text-[10px] font-bold"
          >
            {{ selectedDocIds.length }}
          </span>
        </button>

        <button
          @click="$refs.fileUploadInput.click()"
          class="bg-lynqr-600 hover:bg-lynqr-500 text-white px-3.5 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 cursor-pointer transition-colors shadow-lg shadow-lynqr-900/30"
        >
          <FileUp class="w-4 h-4" />
          <span>Upload Document</span>
        </button>
        <input ref="fileUploadInput" type="file" @change="onUploadNew" class="hidden" />
      </div>
    </div>

    <!-- Alert / Toast Banner -->
    <div
      v-if="statusMessage"
      class="p-3.5 rounded-xl border flex items-center justify-between text-xs transition-all"
      :class="statusMessage.type === 'error' ? 'bg-rose-950/40 border-rose-800 text-rose-300' : 'bg-emerald-950/40 border-emerald-800 text-emerald-300'"
    >
      <div class="flex items-center gap-2">
        <CheckCircle2 v-if="statusMessage.type !== 'error'" class="w-4 h-4" />
        <AlertTriangle v-else class="w-4 h-4" />
        <span>{{ statusMessage.text }}</span>
      </div>
      <button @click="statusMessage = null" class="text-gray-400 hover:text-white">
        <X class="w-3.5 h-3.5" />
      </button>
    </div>

    <!-- Documents List -->
    <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl overflow-hidden shadow-xl">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs text-gray-300">
          <thead class="bg-gray-800/80 text-[11px] uppercase tracking-wider text-gray-400 border-b border-gray-700">
            <tr>
              <th class="px-5 py-3">Document File</th>
              <th class="px-5 py-3">Version</th>
              <th class="px-5 py-3">File Size</th>
              <th class="px-5 py-3">Last Updated</th>
              <th class="px-5 py-3 text-right">Actions</th>
              <!-- Checkbox header at the right of the table -->
              <th class="px-4 py-3 text-right w-12">
                <input
                  type="checkbox"
                  :checked="isAllSelected"
                  @change="toggleSelectAll"
                  class="rounded border-gray-600 bg-gray-900 text-lynqr-500 focus:ring-lynqr-500 w-4 h-4 cursor-pointer"
                  title="Select all documents"
                />
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-750">
            <tr
              v-for="doc in store.documents"
              :key="doc.id"
              :class="[
                'hover:bg-gray-800/40 transition-colors',
                selectedDocIds.includes(doc.id) ? 'bg-lynqr-950/20' : ''
              ]"
            >
              <td class="px-5 py-3.5 flex items-center gap-3">
                <div class="w-8 h-8 rounded-lg bg-rose-500/10 border border-rose-500/20 text-rose-400 flex items-center justify-center font-bold">
                  <FileText class="w-4 h-4" />
                </div>
                <div>
                  <div class="font-semibold text-gray-200">{{ doc.filename }}</div>
                  <div class="text-[11px] text-gray-500">{{ doc.mime_type }}</div>
                </div>
              </td>
              <td class="px-5 py-3.5">
                <span class="px-2 py-0.5 rounded-full bg-lynqr-500/20 text-lynqr-400 font-mono text-[11px]">
                  v{{ doc.version }}
                </span>
              </td>
              <td class="px-5 py-3.5 font-mono text-gray-400">
                {{ (doc.file_size / 1024 / 1024).toFixed(2) }} MB
              </td>
              <td class="px-5 py-3.5 text-gray-400 text-[11px]">
                {{ new Date(doc.updated_at).toLocaleString() }}
              </td>
              <td class="px-5 py-3.5 text-right space-x-2">
                <!-- Create QR Code for Document -->
                <button
                  @click="generateQRForDoc(doc)"
                  class="px-2.5 py-1.5 rounded-lg bg-lynqr-600/20 hover:bg-lynqr-600/30 text-lynqr-300 text-xs font-medium inline-flex items-center gap-1.5 cursor-pointer"
                  title="Generate dynamic QR code for this document"
                >
                  <QrCode class="w-3.5 h-3.5 text-lynqr-400" />
                  <span>Generate QR</span>
                </button>

                <!-- Replace Document File -->
                <button
                  @click="triggerReplace(doc)"
                  class="px-2.5 py-1.5 rounded-lg bg-gray-700 hover:bg-gray-600 text-gray-200 text-xs font-medium inline-flex items-center gap-1.5 cursor-pointer"
                  title="Upload newer version of file while preserving QR code"
                >
                  <RefreshCw class="w-3.5 h-3.5 text-lynqr-400" />
                  <span>Replace File</span>
                </button>

                <!-- Download File -->
                <a
                  :href="`/api/v1/documents/files/${doc.stored_path}`"
                  target="_blank"
                  class="p-1.5 rounded-lg bg-gray-700 hover:bg-gray-600 text-gray-300 inline-block align-middle cursor-pointer"
                  title="Download File"
                >
                  <Download class="w-3.5 h-3.5" />
                </a>

                <!-- Row Delete Button -->
                <button
                  @click="openDeleteModal([doc.id])"
                  class="p-1.5 rounded-lg bg-gray-700/80 hover:bg-rose-600/20 text-gray-400 hover:text-rose-400 inline-block align-middle cursor-pointer transition-colors"
                  title="Delete Document"
                >
                  <Trash2 class="w-3.5 h-3.5" />
                </button>
              </td>

              <!-- Checkbox cell at the right of the table -->
              <td class="px-4 py-3.5 text-right">
                <input
                  type="checkbox"
                  :value="doc.id"
                  v-model="selectedDocIds"
                  class="rounded border-gray-600 bg-gray-900 text-lynqr-500 focus:ring-lynqr-500 w-4 h-4 cursor-pointer"
                />
              </td>
            </tr>

            <tr v-if="!store.documents.length">
              <td colspan="6" class="px-5 py-12 text-center text-gray-500">
                No documents uploaded yet. Upload your first menu or report to get a dynamic reusable QR code.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Hidden file input for replacement -->
    <input ref="replaceFileInput" type="file" @change="onFileReplaceSelected" class="hidden" />

    <!-- Confirmation Modal for Deletion -->
    <div
      v-if="showDeleteModal"
      class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm"
    >
      <div class="bg-gray-800 border border-gray-700 rounded-2xl max-w-md w-full p-6 space-y-4 shadow-2xl">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-rose-500/20 text-rose-400 flex items-center justify-center font-bold">
            <Trash2 class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-base font-bold text-white">Delete Document(s)</h3>
            <p class="text-xs text-gray-400">
              Permanently remove selected files from storage and database.
            </p>
          </div>
        </div>

        <div class="p-3 bg-gray-900/60 rounded-xl border border-gray-750 text-xs text-gray-300">
          You are about to delete <span class="font-bold text-white">{{ pendingDeleteIds.length }}</span> document(s).
          This action cannot be undone.
        </div>

        <!-- Option to delete associated QR code -->
        <label class="flex items-start gap-3 p-3 bg-gray-900/40 rounded-xl border border-gray-750 cursor-pointer select-none">
          <input
            type="checkbox"
            v-model="deleteAssociatedQR"
            class="mt-0.5 rounded border-gray-600 bg-gray-900 text-lynqr-500 focus:ring-lynqr-500 w-4 h-4"
          />
          <div class="text-xs">
            <div class="font-semibold text-gray-200">Delete associated dynamic QR code(s)</div>
            <div class="text-[11px] text-gray-400 mt-0.5">
              Also deletes any dynamic QR codes generated for these documents.
            </div>
          </div>
        </label>

        <div class="flex items-center justify-end gap-2.5 pt-2">
          <button
            @click="showDeleteModal = false"
            class="px-4 py-2 rounded-xl text-xs font-medium text-gray-300 hover:text-white bg-gray-700/60 hover:bg-gray-700 transition-colors"
          >
            Cancel
          </button>
          <button
            @click="executeDelete"
            :disabled="isDeleting"
            class="px-4 py-2 rounded-xl text-xs font-semibold text-white bg-rose-600 hover:bg-rose-500 disabled:opacity-50 transition-colors flex items-center gap-1.5"
          >
            <span v-if="isDeleting">Deleting...</span>
            <span v-else>Confirm Delete</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { FileUp, FileText, QrCode, RefreshCw, Download, Trash2, CheckCircle2, AlertTriangle, X } from 'lucide-vue-next'
import { useLynqrStore } from '../stores/lynqr'
import { api } from '../api/client'

const store = useLynqrStore()
const fileUploadInput = ref(null)
const replaceFileInput = ref(null)
const activeReplacingDoc = ref(null)

const selectedDocIds = ref([])
const showDeleteModal = ref(false)
const pendingDeleteIds = ref([])
const deleteAssociatedQR = ref(true)
const isDeleting = ref(false)
const statusMessage = ref(null)

const isAllSelected = computed(() => {
  return store.documents.length > 0 && selectedDocIds.value.length === store.documents.length
})

function toggleSelectAll() {
  if (isAllSelected.value) {
    selectedDocIds.value = []
  } else {
    selectedDocIds.value = store.documents.map(d => d.id)
  }
}

function openDeleteModal(ids) {
  pendingDeleteIds.value = [...ids]
  showDeleteModal.value = true
}

async function executeDelete() {
  if (pendingDeleteIds.value.length === 0) return
  isDeleting.value = true
  try {
    const res = await api.bulkDeleteDocuments(pendingDeleteIds.value, deleteAssociatedQR.value)
    showDeleteModal.value = false
    selectedDocIds.value = selectedDocIds.value.filter(id => !pendingDeleteIds.value.includes(id))
    pendingDeleteIds.value = []

    statusMessage.value = {
      type: 'success',
      text: `Successfully deleted ${res.deleted_docs_count || 1} document(s)${res.deleted_qr_count ? ` and ${res.deleted_qr_count} associated QR code(s)` : ''}.`
    }

    await store.fetchDocuments()
    await store.fetchQRCodes()
  } catch (err) {
    statusMessage.value = {
      type: 'error',
      text: err.message || 'Failed to delete documents'
    }
  } finally {
    isDeleting.value = false
  }
}

async function onUploadNew(e) {
  const file = e.target.files?.[0]
  if (!file) return
  try {
    await api.uploadDocument(file)
    await store.fetchDocuments()
    await store.fetchQRCodes()
    statusMessage.value = {
      type: 'success',
      text: `Uploaded "${file.name}" and generated dynamic QR code.`
    }
  } catch (err) {
    statusMessage.value = { type: 'error', text: err.message }
  } finally {
    if (e.target) e.target.value = ''
  }
}

async function generateQRForDoc(doc) {
  try {
    const downloadURL = `${window.location.origin}/api/v1/documents/files/${doc.stored_path}`
    await api.createQRCode({
      type: 'document',
      title: `Doc: ${doc.filename}`,
      target_url: downloadURL,
      is_dynamic: true,
      style: {
        fg_color: '#000000',
        bg_color: '#ffffff',
        border_width: 0
      }
    })
    statusMessage.value = {
      type: 'success',
      text: `Dynamic QR code created for "${doc.filename}".`
    }
    await store.fetchQRCodes()
  } catch (err) {
    statusMessage.value = { type: 'error', text: err.message }
  }
}

function triggerReplace(doc) {
  activeReplacingDoc.value = doc
  replaceFileInput.value.click()
}

async function onFileReplaceSelected(e) {
  const file = e.target.files?.[0]
  if (!file || !activeReplacingDoc.value) return
  try {
    await api.updateDocument(activeReplacingDoc.value.id, file)
    statusMessage.value = {
      type: 'success',
      text: `Updated "${activeReplacingDoc.value.filename}" to new version. All existing QR codes now serve this file!`
    }
    await store.fetchDocuments()
  } catch (err) {
    statusMessage.value = { type: 'error', text: err.message }
  } finally {
    if (e.target) e.target.value = ''
  }
}
</script>
