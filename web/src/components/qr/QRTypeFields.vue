<template>
  <div class="space-y-5">
    <!-- Type picker -->
    <div v-if="!lockType">
      <label class="label">Content type</label>
      <div class="grid grid-cols-3 gap-2" role="radiogroup" aria-label="QR content type">
        <button
          v-for="t in QR_TYPES"
          :key="t.id"
          type="button"
          role="radio"
          :aria-checked="model.type === t.id"
          class="group flex flex-col items-center gap-1.5 rounded-xl border px-2 py-3 text-xs font-medium transition"
          :class="model.type === t.id ? 'border-brand-500 bg-brand-500/10 text-brand-600 dark:text-brand-300 ring-4 ring-brand-500/10' : 'border-line text-fg-muted hover:border-line-strong hover:text-fg'"
          :data-testid="`qr-type-${t.id}`"
          :title="t.description"
          @click="setType(t.id)"
        >
          <component :is="t.icon" class="h-5 w-5" />
          {{ t.label }}
        </button>
      </div>
      <p class="hint">{{ typeMeta(model.type).description }}</p>
    </div>
    <div v-else class="flex items-center gap-2 rounded-xl bg-subtle px-3 py-2 text-sm text-fg-muted">
      <component :is="typeMeta(model.type).icon" class="h-4 w-4" />
      <span><span class="font-medium text-fg">{{ typeMeta(model.type).label }}</span> code. The type can't change after printing.</span>
    </div>

    <!-- Website -->
    <template v-if="model.type === 'website'">
      <div>
        <label class="label" for="qr-url">
          Destination URL
          <Tip text="Where scanners land. Must be a public http(s) address." />
        </label>
        <input id="qr-url" v-model="model.target_url" type="url" placeholder="https://example.com/spring-menu" class="input" data-testid="qr-target-url" />
      </div>
      <DynamicToggle v-if="!lockType" v-model="model.is_dynamic" />
    </template>

    <!-- Document -->
    <template v-else-if="model.type === 'document'">
      <div>
        <label class="label" for="qr-doc">
          Document
          <Tip text="The code always opens the latest version. Replace the file on the Documents page and printed codes update instantly." />
        </label>
        <select id="qr-doc" v-model="model.document_id" class="input">
          <option value="" disabled>Select a document…</option>
          <option v-for="d in store.documents" :key="d.id" :value="d.id">{{ d.filename }} (v{{ d.version }})</option>
        </select>
      </div>
      <div class="space-y-2">
        <p class="text-xs text-fg-muted">…or upload a new file</p>
        <FileDrop v-model="pendingFile" />
        <button v-if="pendingFile" type="button" class="btn-secondary btn-sm" :disabled="uploading" @click="uploadDoc">
          <Loader2 v-if="uploading" class="h-3.5 w-3.5 animate-spin" /> Upload {{ pendingFile.name }}
        </button>
      </div>
    </template>

    <!-- Contact card -->
    <template v-else-if="model.type === 'vcard'">
      <div class="grid gap-3 sm:grid-cols-2">
        <Field v-model="model.data.first_name" label="First name" />
        <Field v-model="model.data.last_name" label="Last name" />
        <Field v-model="model.data.organization" label="Company" />
        <Field v-model="model.data.title" label="Job title" />
        <Field v-model="model.data.phone" label="Work phone" type="tel" />
        <Field v-model="model.data.mobile" label="Mobile" type="tel" />
        <Field v-model="model.data.email" label="Email" type="email" />
        <Field v-model="model.data.website" label="Website" type="url" placeholder="https://" />
        <Field v-model="model.data.street" label="Street" class="sm:col-span-2" />
        <Field v-model="model.data.city" label="City" />
        <Field v-model="model.data.region" label="State / region" />
        <Field v-model="model.data.postal_code" label="Postal code" />
        <Field v-model="model.data.country" label="Country" />
      </div>
      <div>
        <label class="label">Note</label>
        <textarea v-model="model.data.note" rows="2" class="input" />
      </div>
      <div>
        <label class="label">
          Format
          <Tip text="vCard 3.0 works on virtually every phone. vCard 4.0 is newer; MeCard is the most compact (smaller, easier-to-scan code) but supports fewer fields." />
        </label>
        <select v-model="model.data.format" class="input" :disabled="model.is_dynamic">
          <option value="vcard3">vCard 3.0 (recommended)</option>
          <option value="vcard4">vCard 4.0</option>
          <option value="mecard">MeCard (compact)</option>
        </select>
      </div>
      <DynamicToggle v-if="!lockType" v-model="model.is_dynamic" description="Host the contact on Lynqr: a smaller code you can edit later, with scan analytics." />
    </template>

    <!-- Wi-Fi -->
    <template v-else-if="model.type === 'wifi'">
      <Field v-model="model.data.ssid" label="Network name (SSID)" required />
      <div>
        <label class="label">
          Security
          <Tip text="Choose WPA for WPA/WPA2 networks. WPA3-only networks need recent phones." />
        </label>
        <select v-model="model.data.encryption" class="input">
          <option value="WPA">WPA / WPA2 (most common)</option>
          <option value="WPA3">WPA3</option>
          <option value="WEP">WEP (legacy)</option>
          <option value="nopass">None (open network)</option>
        </select>
      </div>
      <Field v-if="model.data.encryption !== 'nopass'" v-model="model.data.password" label="Password" />
      <label class="flex items-center justify-between rounded-xl border border-line px-3.5 py-2.5 text-sm">
        <span>Hidden network <Tip text="Turn on if the network doesn't broadcast its name." /></span>
        <Toggle v-model="model.data.hidden" />
      </label>
      <p class="hint">Wi-Fi details are encoded directly in the code, so it works offline and can't be changed after printing.</p>
    </template>

    <!-- App stores -->
    <template v-else-if="model.type === 'app'">
      <Field v-model="model.data.app_store_url" label="Apple App Store link" type="url" placeholder="https://apps.apple.com/app/…" />
      <Field v-model="model.data.play_store_url" label="Google Play link" type="url" placeholder="https://play.google.com/store/apps/details?id=…" />
      <Field v-model="model.data.web_url" label="Fallback web page (optional)" type="url" placeholder="https://example.com/app" />
      <p class="hint">Scanners are routed by device: iPhone → App Store, Android → Google Play, others → fallback page or a store chooser.</p>
    </template>

    <!-- Text -->
    <template v-else-if="model.type === 'text'">
      <div>
        <label class="label">Text</label>
        <textarea v-model="model.data.text" rows="5" maxlength="1200" class="input" placeholder="Shown on the phone after scanning" />
        <p class="hint">{{ (model.data.text || '').length }} / 1200 characters. Shorter text makes a less dense, easier-to-scan code.</p>
      </div>
    </template>

    <!-- Email -->
    <template v-else-if="model.type === 'email'">
      <Field v-model="model.data.email" label="To" type="email" required />
      <Field v-model="model.data.subject" label="Subject" />
      <div><label class="label">Message</label><textarea v-model="model.data.body" rows="3" class="input" /></div>
    </template>

    <!-- SMS -->
    <template v-else-if="model.type === 'sms'">
      <Field v-model="model.data.phone" label="Phone number" type="tel" placeholder="+1 555 010 2000" required />
      <div><label class="label">Message</label><textarea v-model="model.data.message" rows="3" class="input" /></div>
    </template>

    <!-- Phone -->
    <template v-else-if="model.type === 'phone'">
      <Field v-model="model.data.phone" label="Phone number" type="tel" placeholder="+1 555 010 2000" required />
      <p class="hint">Include the country code (e.g. +1) so it dials correctly abroad.</p>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, defineComponent, h } from 'vue'
import { Loader2 } from 'lucide-vue-next'
import { QR_TYPES, typeMeta, defaultsFor } from '../../lib/qrContent'
import { useLynqrStore } from '../../stores/lynqr'
import { api } from '../../api/client'
import { useToast } from '../../composables/useToast'
import Tip from '../ui/Tip.vue'
import Toggle from '../ui/Toggle.vue'
import FileDrop from '../ui/FileDrop.vue'

const props = defineProps({
  modelValue: { type: Object, required: true }, // { type, target_url, is_dynamic, document_id, data }
  lockType: { type: Boolean, default: false }
})
const emit = defineEmits(['update:modelValue'])
const store = useLynqrStore()
const toast = useToast()
const model = computed(() => props.modelValue)
const pendingFile = ref(null)
const uploading = ref(false)

function setType(type) {
  if (type === model.value.type) return
  emit('update:modelValue', { ...model.value, type, data: defaultsFor(type), is_dynamic: type === 'website' ? true : model.value.is_dynamic })
}

async function uploadDoc() {
  uploading.value = true
  try {
    const res = await api.uploadDocument(pendingFile.value)
    await store.fetchDocuments()
    model.value.document_id = res.document.id
    pendingFile.value = null
    toast.success(`${res.document.filename} uploaded`)
  } catch (err) {
    toast.error(err.message)
  } finally {
    uploading.value = false
  }
}

// Small inline field component keeps the per-type forms compact.
const Field = defineComponent({
  props: { modelValue: [String, Number], label: String, type: { type: String, default: 'text' }, placeholder: String, required: Boolean },
  emits: ['update:modelValue'],
  setup(p, { emit: e, attrs }) {
    const id = `f-${Math.random().toString(36).slice(2, 8)}`
    return () =>
      h('div', { class: attrs.class }, [
        h('label', { class: 'label', for: id }, [p.label, p.required ? h('span', { class: 'text-rose-500' }, ' *') : null]),
        h('input', {
          id,
          class: 'input',
          type: p.type,
          placeholder: p.placeholder,
          value: p.modelValue,
          onInput: (ev) => e('update:modelValue', ev.target.value)
        })
      ])
  }
})

const DynamicToggle = defineComponent({
  props: { modelValue: Boolean, description: { type: String, default: 'Change the destination after printing, and get scan analytics.' } },
  emits: ['update:modelValue'],
  setup(p, { emit: e }) {
    return () =>
      h('label', { class: 'flex items-center justify-between gap-4 rounded-xl border border-line px-3.5 py-3' }, [
        h('span', { class: 'text-sm' }, [
          h('span', { class: 'font-medium text-fg' }, 'Dynamic QR code'),
          h('span', { class: 'block text-xs text-fg-muted' }, p.description)
        ]),
        h(Toggle, { modelValue: p.modelValue, 'onUpdate:modelValue': (v) => e('update:modelValue', v), 'data-testid': 'qr-dynamic-toggle' })
      ])
  }
})
</script>
