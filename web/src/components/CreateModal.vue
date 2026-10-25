<template>
  <AppModal :open="isOpen" size="lg" title="Create new" description="Short links, QR codes and hosted documents for your workspace." @close="close">
    <!-- Tabs -->
    <div class="mb-5 grid grid-cols-3 gap-1 rounded-xl bg-subtle p-1" role="tablist">
      <button
        v-for="t in tabs"
        :key="t.id"
        role="tab"
        :aria-selected="activeTab === t.id"
        class="flex items-center justify-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all"
        :class="activeTab === t.id ? 'bg-surface text-fg shadow-card' : 'text-fg-muted hover:text-fg'"
        :data-testid="`create-tab-${t.id}`"
        @click="activeTab = t.id"
      >
        <component :is="t.icon" class="w-4 h-4" />{{ t.label }}
      </button>
    </div>

    <div v-if="errorMessage" class="mb-4 flex items-start gap-2 rounded-xl border border-rose-500/20 bg-rose-500/10 px-3 py-2.5 text-sm text-rose-600 dark:text-rose-400">
      <AlertCircle class="mt-0.5 h-4 w-4 shrink-0" /><span>{{ errorMessage }}</span>
    </div>

    <form id="create-form" class="space-y-4" @submit.prevent="submit">
      <!-- ---------- Short link ---------- -->
      <template v-if="activeTab === 'link'">
        <div>
          <label class="label" for="link-target">Destination URL</label>
          <input id="link-target" v-model="linkForm.target_url" type="url" required placeholder="https://example.com/landing-page" class="input" data-testid="link-target" />
        </div>
        <div>
          <label class="label" for="link-slug">
            Custom back-half <span class="text-fg-subtle font-normal">(optional)</span>
            <Tip text="3–64 letters, numbers, dashes or underscores. Leave blank for a random collision-resistant ID." />
          </label>
          <div class="flex items-stretch overflow-hidden rounded-xl border border-line focus-within:border-brand-500 focus-within:ring-4 focus-within:ring-brand-500/15">
            <span class="flex items-center bg-subtle px-3 text-sm text-fg-subtle font-mono">{{ origin }}/s/</span>
            <input id="link-slug" v-model="linkForm.custom_slug" pattern="[A-Za-z0-9][A-Za-z0-9_\-]{2,63}" placeholder="summer-sale" class="flex-1 bg-surface px-3 py-2 text-sm text-fg outline-none" data-testid="link-slug" />
          </div>
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="label" for="link-title">Title <span class="text-fg-subtle font-normal">(optional)</span></label>
            <input id="link-title" v-model="linkForm.title" placeholder="Summer campaign" class="input" />
          </div>
          <div>
            <label class="label" for="link-expiry">
              Expires <span class="text-fg-subtle font-normal">(optional)</span>
              <Tip text="After this moment the link shows a branded 'expired' page instead of redirecting." />
            </label>
            <input id="link-expiry" v-model="linkForm.expires_local" type="datetime-local" class="input" />
          </div>
        </div>
      </template>

      <!-- ---------- QR code ---------- -->
      <template v-else-if="activeTab === 'qr'">
        <div>
          <label class="label" for="qr-title">Name</label>
          <input id="qr-title" v-model="qrTitle" required maxlength="120" placeholder="Storefront window decal" class="input" data-testid="qr-title" />
        </div>
        <QRTypeFields v-model="qrContent" />
        <p class="text-xs text-fg-muted">
          Want custom colors, logos and frames?
          <router-link :to="{ path: '/qr-studio', query: { type: qrContent.type } }" class="font-medium text-brand-500 hover:underline" @click="close">Open QR Studio</router-link>
        </p>
      </template>

      <!-- ---------- Document ---------- -->
      <template v-else>
        <FileDrop v-model="selectedFile" />
        <p class="text-xs text-fg-muted">A dynamic QR code is created automatically. Replace the file any time and the QR keeps working.</p>
      </template>
    </form>

    <template #footer>
      <button class="btn-secondary" type="button" @click="close">Cancel</button>
      <button class="btn-primary" type="submit" form="create-form" :disabled="isSubmitting" data-testid="create-submit">
        <Loader2 v-if="isSubmitting" class="w-4 h-4 animate-spin" />
        {{ submitLabel }}
      </button>
    </template>
  </AppModal>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { Link2, QrCode, FileUp, AlertCircle, Loader2 } from 'lucide-vue-next'
import { api } from '../api/client'
import { useLynqrStore } from '../stores/lynqr'
import { useToast } from '../composables/useToast'
import AppModal from './ui/AppModal.vue'
import Tip from './ui/Tip.vue'
import FileDrop from './ui/FileDrop.vue'
import QRTypeFields from './qr/QRTypeFields.vue'
import { defaultsFor, rawDataFor, validateContent } from '../lib/qrContent'

const props = defineProps({
  isOpen: { type: Boolean, default: false },
  initialTab: { type: String, default: 'link' }
})
const emit = defineEmits(['close', 'created'])
const store = useLynqrStore()
const toast = useToast()
const origin = window.location.host

const tabs = [
  { id: 'link', label: 'Short link', icon: Link2 },
  { id: 'qr', label: 'QR code', icon: QrCode },
  { id: 'doc', label: 'Document', icon: FileUp }
]
const activeTab = ref('link')
const isSubmitting = ref(false)
const errorMessage = ref('')

const blankLink = () => ({ target_url: '', title: '', custom_slug: '', expires_local: '' })
const blankQR = () => ({ type: 'website', target_url: '', is_dynamic: true, document_id: '', data: defaultsFor('website') })
const linkForm = ref(blankLink())
const qrContent = ref(blankQR())
const qrTitle = ref('')
const selectedFile = ref(null)

const submitLabel = computed(() => ({ link: 'Create link', qr: 'Create QR code', doc: 'Upload document' })[activeTab.value])

watch(() => props.isOpen, (open) => {
  if (!open) return
  activeTab.value = props.initialTab || 'link'
  errorMessage.value = ''
  store.fetchDocuments()
})

function reset() {
  linkForm.value = blankLink()
  qrContent.value = blankQR()
  qrTitle.value = ''
  selectedFile.value = null
}

function close() {
  emit('close')
}

async function submit() {
  errorMessage.value = ''
  isSubmitting.value = true
  try {
    if (activeTab.value === 'link') {
      const { expires_local, ...rest } = linkForm.value
      const link = await api.createLink({ ...rest, expires_at: expires_local ? new Date(expires_local).toISOString() : null })
      toast.success(`${window.location.host}/s/${link.slug} is live`, { title: 'Short link created' })
    } else if (activeTab.value === 'qr') {
      const problem = validateContent(qrContent.value)
      if (problem) throw new Error(problem)
      const c = qrContent.value
      const created = await api.createQRCode({
        type: c.type,
        title: qrTitle.value,
        target_url: c.target_url,
        raw_data: rawDataFor(c.type, c.data),
        is_dynamic: c.is_dynamic,
        document_id: c.document_id
      })
      toast.success(created.title, { title: 'QR code created' })
    } else {
      if (!selectedFile.value) throw new Error('Choose a file to upload')
      await api.uploadDocument(selectedFile.value)
      toast.success(selectedFile.value.name, { title: 'Document uploaded' })
    }
    reset()
    await store.refreshAll()
    emit('created')
    emit('close')
  } catch (err) {
    errorMessage.value = err.message || 'Something went wrong'
  } finally {
    isSubmitting.value = false
  }
}
</script>
