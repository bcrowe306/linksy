<template>
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold text-white">Dynamic QR Codes</h1>
        <p class="text-xs text-gray-400 mt-0.5">Customize, export in high-res (PNG, JPEG, PDF, SVG), and update destinations.</p>
      </div>
      <div class="flex items-center gap-2.5">
        <router-link
          to="/qr-studio"
          class="bg-gray-800 hover:bg-gray-750 border border-gray-700 text-linksy-400 px-3.5 py-2 rounded-xl text-xs font-semibold flex items-center gap-1.5 transition-colors"
        >
          <Sparkles class="w-4 h-4 text-amber-400" />
          <span>Open Design Studio</span>
        </router-link>
        <button
          @click="$emit('open-create')"
          class="bg-linksy-600 hover:bg-linksy-500 text-white px-3.5 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 shadow-lg shadow-linksy-600/20"
        >
          <Plus class="w-4 h-4" />
          <span>Quick QR</span>
        </button>
      </div>
    </div>

    <!-- QR Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div
        v-for="qr in store.qrCodes"
        :key="qr.id"
        class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-5 flex flex-col justify-between hover:border-gray-600 transition-all group"
      >
        <div>
          <!-- QR Card Header -->
          <div class="flex items-start justify-between gap-3 mb-4">
            <div>
              <h3 class="text-sm font-bold text-white truncate max-w-[200px]">{{ qr.title }}</h3>
              <div class="flex items-center gap-1.5 mt-0.5">
                <span class="text-[10px] px-2 py-0.5 rounded-full font-medium uppercase tracking-wider"
                  :class="qr.link_id ? 'bg-linksy-500/20 text-linksy-400' : 'bg-gray-700 text-gray-300'">
                  {{ qr.link_id ? 'Dynamic' : 'Static' }} • {{ qr.type }}
                </span>
              </div>
            </div>

            <div class="flex items-center gap-1">
              <router-link
                :to="'/qr-studio/' + qr.id"
                class="text-gray-400 hover:text-white p-1 rounded-lg hover:bg-gray-800"
                title="Edit Style in Studio"
              >
                <Palette class="w-3.5 h-3.5" />
              </router-link>
              <button
                @click="deleteQR(qr.id)"
                class="text-gray-500 hover:text-rose-400 p-1 rounded-lg hover:bg-gray-800"
                title="Delete QR"
              >
                <Trash2 class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>

          <!-- QR Preview Image -->
          <div class="bg-white rounded-xl p-4 flex flex-col items-center justify-center mb-4 shadow-inner relative">
            <img
              :src="`/api/v1/qr/${qr.id}/export?format=png&size=240`"
              class="w-36 h-36 object-contain"
            />
            <div v-if="qr.style?.label_text" class="mt-1 text-[10px] font-bold text-black uppercase tracking-wider">
              {{ qr.style.label_text }}
            </div>
          </div>

          <!-- Payload / Destination preview -->
          <div class="text-[11px] text-gray-400 font-mono truncate mb-4 bg-gray-900/60 p-2 rounded-lg border border-gray-800">
            {{ qr.payload }}
          </div>
        </div>

        <!-- Export & Action Bar -->
        <div class="space-y-2 pt-3 border-t border-gray-700/60">
          <div class="flex items-center justify-between text-xs text-gray-400 mb-1">
            <span class="font-mono">{{ qr.scan_count }} scans</span>
            <div class="flex items-center gap-3">
              <router-link
                :to="'/qr-studio/' + qr.id"
                class="text-amber-400 hover:underline flex items-center gap-1 text-[11px]"
              >
                <Palette class="w-3 h-3" />
                <span>Customize</span>
              </router-link>
              <!-- Edit Destination Modal trigger -->
              <button
                v-if="qr.link_id"
                @click="openEditQR(qr)"
                class="text-linksy-400 hover:underline flex items-center gap-1 text-[11px]"
              >
                <Edit3 class="w-3 h-3" />
                <span>Destination</span>
              </button>
            </div>
          </div>

          <!-- 4 Format Export Row (Requirement 5) -->
          <div class="grid grid-cols-4 gap-1.5">
            <a
              :href="`/api/v1/qr/${qr.id}/export?format=png&download=true`"
              class="py-1.5 rounded bg-gray-700 hover:bg-gray-600 text-white text-[10px] font-semibold text-center"
            >
              PNG
            </a>
            <a
              :href="`/api/v1/qr/${qr.id}/export?format=jpeg&download=true`"
              class="py-1.5 rounded bg-gray-700 hover:bg-gray-600 text-white text-[10px] font-semibold text-center"
            >
              JPEG
            </a>
            <a
              :href="`/api/v1/qr/${qr.id}/export?format=pdf&download=true`"
              class="py-1.5 rounded bg-gray-700 hover:bg-gray-600 text-white text-[10px] font-semibold text-center"
            >
              PDF
            </a>
            <a
              :href="`/api/v1/qr/${qr.id}/export?format=svg&download=true`"
              class="py-1.5 rounded bg-gray-700 hover:bg-gray-600 text-white text-[10px] font-semibold text-center"
            >
              SVG
            </a>
          </div>
        </div>
      </div>

      <div v-if="!store.qrCodes.length" class="col-span-full py-16 text-center text-xs text-gray-500 bg-gray-800/40 rounded-2xl border border-dashed border-gray-700">
        No QR codes created yet. Click "Open Design Studio" to craft a custom styled code with borders, logos, and gradients.
      </div>
    </div>

    <!-- Edit Dynamic Destination Modal (Requirement 6) -->
    <div v-if="editingQR" class="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div @click="editingQR = null" class="absolute inset-0 bg-black/75 backdrop-blur-sm"></div>
      <div class="relative bg-gray-900 border border-gray-800 rounded-2xl w-full max-w-lg p-6 shadow-2xl z-10 space-y-4">
        <h3 class="text-base font-bold text-white flex items-center gap-2">
          <Edit3 class="w-4 h-4 text-linksy-400" />
          <span>Change QR Code Destination</span>
        </h3>
        <p class="text-xs text-gray-400">
          This is a dynamic QR code. Updating its destination website will immediately redirect future scans without changing the physical QR image.
        </p>

        <div>
          <label class="block text-xs font-semibold text-gray-300 mb-1">QR Title</label>
          <input v-model="editTitle" type="text" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
        </div>

        <div>
          <label class="block text-xs font-semibold text-gray-300 mb-1">New Target Destination URL</label>
          <input v-model="editTarget" type="url" placeholder="https://newdestination.com" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100" />
        </div>

        <div class="flex justify-end gap-2 pt-2">
          <button @click="editingQR = null" class="px-3.5 py-1.5 text-xs text-gray-400 hover:text-white">Cancel</button>
          <button @click="saveQREdit" class="px-4 py-1.5 text-xs font-semibold bg-linksy-600 hover:bg-linksy-500 text-white rounded-lg">
            Save Changes
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Plus, Trash2, Edit3, Palette, Sparkles } from 'lucide-vue-next'
import { useLinksyStore } from '../stores/linksy'
import { api } from '../api/client'

defineEmits(['open-create'])
const store = useLinksyStore()

const editingQR = ref(null)
const editTitle = ref('')
const editTarget = ref('')

function openEditQR(qr) {
  editingQR.value = qr
  editTitle.value = qr.title
  editTarget.value = ''
}

async function saveQREdit() {
  if (!editingQR.value) return
  try {
    await api.updateQRCode(editingQR.value.id, {
      title: editTitle.value,
      target_url: editTarget.value,
      style: editingQR.value.style
    })
    editingQR.value = null
    await store.fetchQRCodes()
  } catch (err) {
    alert(err.message)
  }
}

async function deleteQR(id) {
  if (!confirm('Are you sure you want to delete this QR code?')) return
  try {
    await api.deleteQRCode(id)
    await store.fetchQRCodes()
  } catch (err) {
    alert(err.message)
  }
}
</script>
