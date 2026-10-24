<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold text-white">Short Links</h1>
        <p class="text-xs text-gray-400 mt-0.5">Manage shortened links and change dynamic destinations anytime.</p>
      </div>
      <button
        @click="$emit('open-create')"
        class="bg-linksy-600 hover:bg-linksy-500 text-white px-3.5 py-2 rounded-xl text-xs font-semibold flex items-center gap-2"
      >
        <Plus class="w-4 h-4" />
        <span>New Link</span>
      </button>
    </div>

    <!-- Table -->
    <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs text-gray-300">
          <thead class="bg-gray-800/80 text-[11px] uppercase tracking-wider text-gray-400 border-b border-gray-700">
            <tr>
              <th class="px-5 py-3">Link Title & Target</th>
              <th class="px-5 py-3">Short URL</th>
              <th class="px-5 py-3">Clicks</th>
              <th class="px-5 py-3">Created</th>
              <th class="px-5 py-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-750">
            <tr v-for="link in store.links" :key="link.id" class="hover:bg-gray-800/40 transition-colors">
              <td class="px-5 py-3.5">
                <div class="font-semibold text-gray-200">{{ link.title }}</div>
                <a
                  :href="link.target_url"
                  target="_blank"
                  class="text-gray-400 hover:text-linksy-400 truncate max-w-sm block text-[11px] mt-0.5"
                >
                  {{ link.target_url }}
                </a>
              </td>
              <td class="px-5 py-3.5 font-mono text-linksy-400">
                <div class="flex items-center gap-1.5">
                  <span>/s/{{ link.slug }}</span>
                  <button @click="copyShort(link.slug)" class="p-1 text-gray-400 hover:text-white" title="Copy">
                    <Copy class="w-3.5 h-3.5" />
                  </button>
                </div>
              </td>
              <td class="px-5 py-3.5 font-mono font-semibold text-gray-200">
                {{ link.click_count }}
              </td>
              <td class="px-5 py-3.5 text-gray-400 text-[11px]">
                {{ new Date(link.created_at).toLocaleDateString() }}
              </td>
              <td class="px-5 py-3.5 text-right space-x-2">
                <!-- Edit Destination Dynamically (Requirement 6) -->
                <button
                  @click="openEditModal(link)"
                  class="p-1.5 rounded-lg bg-gray-700 hover:bg-gray-600 text-gray-300"
                  title="Change destination"
                >
                  <Edit3 class="w-3.5 h-3.5" />
                </button>
                <button
                  @click="deleteLink(link.id)"
                  class="p-1.5 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-400"
                  title="Delete link"
                >
                  <Trash2 class="w-3.5 h-3.5" />
                </button>
              </td>
            </tr>
            <tr v-if="!store.links.length">
              <td colspan="5" class="px-5 py-10 text-center text-gray-500">
                No short links created yet. Click "New Link" above to get started.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Edit Destination Modal (Requirement 6) -->
    <div v-if="editingLink" class="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div @click="editingLink = null" class="absolute inset-0 bg-black/75 backdrop-blur-sm"></div>
      <div class="relative bg-gray-900 border border-gray-800 rounded-2xl w-full max-w-lg p-6 shadow-2xl z-10 space-y-4">
        <h3 class="text-base font-bold text-white flex items-center gap-2">
          <Edit3 class="w-4 h-4 text-linksy-400" />
          <span>Edit Link Destination (Dynamic Routing)</span>
        </h3>
        <p class="text-xs text-gray-400">
          Updating the target URL will immediately redirect future clicks without altering the short slug or any printed QR codes.
        </p>

        <div>
          <label class="block text-xs font-semibold text-gray-300 mb-1">Link Title</label>
          <input
            v-model="editForm.title"
            type="text"
            class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold text-gray-300 mb-1">New Destination Target URL</label>
          <input
            v-model="editForm.target_url"
            type="url"
            class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100"
          />
        </div>

        <div class="flex justify-end gap-2 pt-2">
          <button @click="editingLink = null" class="px-3.5 py-1.5 text-xs text-gray-400 hover:text-white">Cancel</button>
          <button @click="saveEdit" class="px-4 py-1.5 text-xs font-semibold bg-linksy-600 hover:bg-linksy-500 text-white rounded-lg">
            Save Changes
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Plus, Copy, Edit3, Trash2 } from 'lucide-vue-next'
import { useLinksyStore } from '../stores/linksy'
import { api } from '../api/client'

defineEmits(['open-create'])
const store = useLinksyStore()

const editingLink = ref(null)
const editForm = ref({ title: '', target_url: '', is_active: true })

function copyShort(slug) {
  const url = `${window.location.origin}/s/${slug}`
  navigator.clipboard.writeText(url)
}

function openEditModal(link) {
  editingLink.value = link
  editForm.value = {
    title: link.title,
    target_url: link.target_url,
    is_active: link.is_active
  }
}

async function saveEdit() {
  if (!editingLink.value) return
  try {
    await api.updateLink(editingLink.value.id, editForm.value)
    editingLink.value = null
    await store.fetchLinks()
  } catch (err) {
    alert(err.message)
  }
}

async function deleteLink(id) {
  if (!confirm('Are you sure you want to delete this short link?')) return
  try {
    await api.deleteLink(id)
    await store.fetchLinks()
  } catch (err) {
    alert(err.message)
  }
}
</script>
