<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold text-white">Document Sharing & Reusable QR Codes</h1>
        <p class="text-xs text-gray-400 mt-0.5">
          Upload menus, catalogs, and reports. Replace the underlying document at any time without changing the physical QR code.
        </p>
      </div>

      <button
        @click="$refs.fileUploadInput.click()"
        class="bg-linksy-600 hover:bg-linksy-500 text-white px-3.5 py-2 rounded-xl text-xs font-semibold flex items-center gap-2"
      >
        <FileUp class="w-4 h-4" />
        <span>Upload Document</span>
      </button>
      <input ref="fileUploadInput" type="file" @change="onUploadNew" class="hidden" />
    </div>

    <!-- Documents List -->
    <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs text-gray-300">
          <thead class="bg-gray-800/80 text-[11px] uppercase tracking-wider text-gray-400 border-b border-gray-700">
            <tr>
              <th class="px-5 py-3">Document File</th>
              <th class="px-5 py-3">Version</th>
              <th class="px-5 py-3">File Size</th>
              <th class="px-5 py-3">Last Updated</th>
              <th class="px-5 py-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-750">
            <tr v-for="doc in store.documents" :key="doc.id" class="hover:bg-gray-800/40 transition-colors">
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
                <span class="px-2 py-0.5 rounded-full bg-linksy-500/20 text-linksy-400 font-mono text-[11px]">
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
                <!-- Replace Document File (Requirement 8) -->
                <button
                  @click="triggerReplace(doc)"
                  class="px-2.5 py-1.5 rounded-lg bg-gray-700 hover:bg-gray-600 text-gray-200 text-xs font-medium inline-flex items-center gap-1.5"
                  title="Upload newer version of file while preserving QR code"
                >
                  <RefreshCw class="w-3.5 h-3.5 text-linksy-400" />
                  <span>Replace File</span>
                </button>

                <!-- Download File -->
                <a
                  :href="`/api/v1/documents/files/${doc.stored_path}`"
                  target="_blank"
                  class="p-1.5 rounded-lg bg-gray-700 hover:bg-gray-600 text-gray-300 inline-block align-middle"
                  title="Download File"
                >
                  <Download class="w-3.5 h-3.5" />
                </a>
              </td>
            </tr>
            <tr v-if="!store.documents.length">
              <td colspan="5" class="px-5 py-12 text-center text-gray-500">
                No documents uploaded yet. Upload your first menu or report to get a dynamic reusable QR code.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Hidden file input for replacement -->
    <input ref="replaceFileInput" type="file" @change="onFileReplaceSelected" class="hidden" />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { FileUp, FileText, RefreshCw, Download } from 'lucide-vue-next'
import { useLinksyStore } from '../stores/linksy'
import { api } from '../api/client'

const store = useLinksyStore()
const fileUploadInput = ref(null)
const replaceFileInput = ref(null)
const activeReplacingDoc = ref(null)

async function onUploadNew(e) {
  const file = e.target.files?.[0]
  if (!file) return
  try {
    await api.uploadDocument(file)
    await store.fetchDocuments()
    await store.fetchQRCodes()
  } catch (err) {
    alert(err.message)
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
    alert(`Successfully updated "${activeReplacingDoc.value.filename}" to new version. All existing QR codes will now serve this file!`)
    await store.fetchDocuments()
  } catch (err) {
    alert(err.message)
  }
}
</script>
