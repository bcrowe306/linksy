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
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="label" for="qr-title">Name</label>
            <input id="qr-title" v-model="qrForm.title" required placeholder="Storefront window decal" class="input" data-testid="qr-title" />
          </div>
          <div>
            <label class="label" for="qr-type">Type</label>
            <select id="qr-type" v-model="qrForm.type" class="input" @change="onQRTypeChange">
              <option value="website">Website / URL</option>
              <option value="document">Document / file</option>
              <option value="vcard">Contact card (vCard)</option>
              <option value="wifi">Wi-Fi network</option>
              <option value="email">Email</option>
              <option value="sms">SMS</option>
              <option value="text">Plain text</option>
            </select>
          </div>
        </div>

        <div v-if="qrForm.type === 'website'" class="space-y-3">
          <div>
            <label class="label" for="qr-target">Destination URL</label>
            <input id="qr-target" v-model="qrForm.target_url" type="url" required placeholder="https://example.com" class="input" data-testid="qr-target" />
          </div>
          <label class="flex items-center justify-between gap-4 rounded-xl border border-line px-3.5 py-2.5">
            <span class="text-sm">
              <span class="font-medium text-fg">Dynamic QR</span>
              <span class="block text-xs text-fg-muted">Change the destination later without reprinting.</span>
            </span>
            <Toggle v-model="qrForm.is_dynamic" />
          </label>
        </div>

        <div v-else-if="qrForm.type === 'document'" class="space-y-3">
          <div class="grid grid-cols-2 gap-1 rounded-xl bg-subtle p-1">
            <button type="button" class="rounded-lg py-1.5 text-xs font-medium" :class="docSource === 'existing' ? 'bg-surface shadow-card text-fg' : 'text-fg-muted'" @click="docSource = 'existing'">Existing document</button>
            <button type="button" class="rounded-lg py-1.5 text-xs font-medium" :class="docSource === 'new' ? 'bg-surface shadow-card text-fg' : 'text-fg-muted'" @click="docSource = 'new'">Upload new</button>
          </div>
          <select v-if="docSource === 'existing'" v-model="selectedExistingDocId" class="input" required>
            <option value="" disabled>Select a document…</option>
            <option v-for="d in store.documents" :key="d.id" :value="d.id">{{ d.filename }} (v{{ d.version }})</option>
          </select>
          <FileDrop v-else v-model="qrDocFile" />
        </div>

        <div v-else-if="qrForm.type === 'wifi'" class="grid gap-4 sm:grid-cols-2">
          <div><label class="label">Network name (SSID)</label><input v-model="wifiData.ssid" required class="input" /></div>
          <div><label class="label">Password</label><input v-model="wifiData.password" class="input" /></div>
          <div>
            <label class="label">Security</label>
            <select v-model="wifiData.encryption" class="input">
              <option value="WPA">WPA / WPA2 / WPA3</option><option value="WEP">WEP</option><option value="nopass">None</option>
            </select>
          </div>
        </div>

        <div v-else-if="qrForm.type === 'vcard'" class="grid gap-4 sm:grid-cols-2">
          <div><label class="label">First name</label><input v-model="vcardData.first_name" class="input" /></div>
          <div><label class="label">Last name</label><input v-model="vcardData.last_name" class="input" /></div>
          <div><label class="label">Phone</label><input v-model="vcardData.phone" type="tel" class="input" /></div>
          <div><label class="label">Email</label><input v-model="vcardData.email" type="email" class="input" /></div>
        </div>

        <div v-else-if="qrForm.type === 'email'" class="space-y-4">
          <div><label class="label">To</label><input v-model="emailData.email" type="email" required class="input" /></div>
          <div><label class="label">Subject</label><input v-model="emailData.subject" class="input" /></div>
          <div><label class="label">Body</label><textarea v-model="emailData.body" rows="3" class="input" /></div>
        </div>

        <div v-else-if="qrForm.type === 'sms'" class="space-y-4">
          <div><label class="label">Phone number</label><input v-model="smsData.phone" type="tel" required class="input" /></div>
          <div><label class="label">Message</label><textarea v-model="smsData.message" rows="3" class="input" /></div>
        </div>

        <div v-else-if="qrForm.type === 'text'">
          <label class="label">Text</label>
          <textarea v-model="textData" rows="4" maxlength="2000" required class="input" />
          <p class="hint">{{ textData.length }} / 2000 characters</p>
        </div>

        <p class="text-xs text-fg-muted">
          Want custom colors, logos and frames?
          <router-link to="/qr-studio" class="font-medium text-brand-500 hover:underline" @click="close">Open QR Studio</router-link>
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
import Toggle from './ui/Toggle.vue'
import FileDrop from './ui/FileDrop.vue'

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
const blankQR = () => ({ type: 'website', title: '', target_url: '', is_dynamic: true })
const linkForm = ref(blankLink())
const qrForm = ref(blankQR())
const docSource = ref('existing')
const selectedExistingDocId = ref('')
const qrDocFile = ref(null)
const selectedFile = ref(null)
const wifiData = ref({ ssid: '', password: '', encryption: 'WPA' })
const vcardData = ref({ first_name: '', last_name: '', phone: '', email: '' })
const emailData = ref({ email: '', subject: '', body: '' })
const smsData = ref({ phone: '', message: '' })
const textData = ref('')

const submitLabel = computed(() => ({ link: 'Create link', qr: 'Create QR code', doc: 'Upload document' })[activeTab.value])

watch(() => props.isOpen, (open) => {
  if (!open) return
  activeTab.value = props.initialTab || 'link'
  errorMessage.value = ''
  store.fetchDocuments()
})

function onQRTypeChange() {
  if (qrForm.value.type === 'document') docSource.value = store.documents.length ? 'existing' : 'new'
}

function reset() {
  linkForm.value = blankLink()
  qrForm.value = blankQR()
  selectedFile.value = null
  qrDocFile.value = null
  selectedExistingDocId.value = ''
  textData.value = ''
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
      const payload = { ...qrForm.value, raw_data: '' }
      const type = payload.type
      if (type === 'wifi') payload.raw_data = JSON.stringify(wifiData.value)
      else if (type === 'vcard') payload.raw_data = JSON.stringify(vcardData.value)
      else if (type === 'email') payload.raw_data = JSON.stringify(emailData.value)
      else if (type === 'sms') payload.raw_data = JSON.stringify(smsData.value)
      else if (type === 'text') payload.raw_data = textData.value
      else if (type === 'document') {
        payload.is_dynamic = true
        let doc
        if (docSource.value === 'existing') {
          doc = store.documents.find((d) => d.id === selectedExistingDocId.value)
          if (!doc) throw new Error('Select a document to link')
          payload.target_url = `${window.location.origin}/api/v1/documents/files/${doc.stored_path}`
        } else {
          if (!qrDocFile.value) throw new Error('Choose a file to upload')
          const up = await api.uploadDocument(qrDocFile.value)
          doc = up.document
          payload.target_url = up.download_url
        }
        payload.document_id = doc.id
      }
      if (type !== 'website' && type !== 'document') payload.is_dynamic = false
      await api.createQRCode(payload)
      toast.success(qrForm.value.title, { title: 'QR code created' })
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
