<template>
  <div class="space-y-3">
    <!-- Password -->
    <details class="group rounded-xl border border-line" :open="openSection === 'password' || undefined">
      <summary class="flex cursor-pointer list-none items-center gap-3 px-3.5 py-2.5 text-sm">
        <Lock class="h-4 w-4 text-fg-subtle" />
        <span class="flex-1 font-medium text-fg">Password protection</span>
        <span v-if="hasPassword && model.password === undefined" class="badge-brand">On</span>
        <span v-else-if="model.password" class="badge-brand">Set</span>
        <ChevronDown class="h-4 w-4 text-fg-subtle transition group-open:rotate-180" />
      </summary>
      <div class="space-y-2 border-t border-line px-3.5 py-3">
        <p class="text-xs text-fg-muted">Visitors must enter this password before being redirected. QR scans of this link are protected too.</p>
        <template v-if="hasPassword && model.password === undefined">
          <div class="flex items-center gap-2">
            <span class="text-sm text-fg">This link is protected.</span>
            <button type="button" class="btn-ghost btn-sm" @click="model.password = 'change'">Change</button>
            <button type="button" class="btn-danger-ghost btn-sm" data-testid="remove-password" @click="model.password = ''">Remove</button>
          </div>
        </template>
        <template v-else-if="model.password === '' && hasPassword">
          <p class="text-sm text-rose-500">Protection will be removed when you save. <button type="button" class="underline" @click="model.password = undefined">Undo</button></p>
        </template>
        <div v-else class="relative">
          <input
            :value="model.password === 'change' ? '' : model.password"
            :type="show ? 'text' : 'password'"
            minlength="4"
            maxlength="72"
            autocomplete="new-password"
            placeholder="At least 4 characters"
            class="input pr-10"
            data-testid="link-password"
            @input="model.password = $event.target.value"
          />
          <button type="button" class="absolute right-2.5 top-1/2 -translate-y-1/2 text-fg-subtle hover:text-fg" :aria-label="show ? 'Hide password' : 'Show password'" @click="show = !show">
            <EyeOff v-if="show" class="h-4 w-4" /><Eye v-else class="h-4 w-4" />
          </button>
        </div>
      </div>
    </details>

    <!-- UTM -->
    <details class="group rounded-xl border border-line" :open="openSection === 'utm' || hasUTM || undefined">
      <summary class="flex cursor-pointer list-none items-center gap-3 px-3.5 py-2.5 text-sm">
        <Tag class="h-4 w-4 text-fg-subtle" />
        <span class="flex-1 font-medium text-fg">UTM campaign tags</span>
        <span v-if="hasUTM" class="badge-brand">{{ utmCount }}</span>
        <ChevronDown class="h-4 w-4 text-fg-subtle transition group-open:rotate-180" />
      </summary>
      <div class="space-y-3 border-t border-line px-3.5 py-3">
        <p class="text-xs text-fg-muted">
          Added to the destination automatically so Google Analytics and similar tools attribute the visit.
          Tags already on the destination URL are never overwritten.
        </p>
        <div class="flex flex-wrap gap-1.5">
          <button v-for="p in presets" :key="p.label" type="button" class="badge-neutral hover:ring-line-strong" @click="applyPreset(p)">{{ p.label }}</button>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div v-for="f in utmFields" :key="f.key">
            <label class="label" :for="`utm-${f.key}`">{{ f.label }} <Tip :text="f.tip" /></label>
            <input :id="`utm-${f.key}`" v-model="model[f.key]" maxlength="100" :placeholder="f.placeholder" class="input input-sm" :data-testid="`link-${f.key}`" />
          </div>
        </div>
        <p v-if="preview" class="truncate rounded-lg bg-subtle px-2.5 py-1.5 font-mono text-[11px] text-fg-muted" :title="preview">→ {{ preview }}</p>
      </div>
    </details>

    <!-- Expiry & fallback -->
    <details class="group rounded-xl border border-line" :open="openSection === 'expiry' || !!model.expires_local || !!model.fallback_url || undefined">
      <summary class="flex cursor-pointer list-none items-center gap-3 px-3.5 py-2.5 text-sm">
        <CalendarClock class="h-4 w-4 text-fg-subtle" />
        <span class="flex-1 font-medium text-fg">Expiration & fallback</span>
        <span v-if="model.expires_local" class="badge-warning">Expires</span>
        <ChevronDown class="h-4 w-4 text-fg-subtle transition group-open:rotate-180" />
      </summary>
      <div class="grid gap-3 border-t border-line px-3.5 py-3 sm:grid-cols-2">
        <div>
          <label class="label" for="link-expiry">Expires <Tip text="After this moment the link stops redirecting to its destination." /></label>
          <input id="link-expiry" v-model="model.expires_local" type="datetime-local" class="input input-sm" />
        </div>
        <div>
          <label class="label" for="link-fallback">Fallback URL <Tip text="Where visitors go when the link is paused or expired. Leave empty to show a branded 'expired' page." /></label>
          <input id="link-fallback" v-model="model.fallback_url" type="url" placeholder="https://example.com/offer-ended" class="input input-sm" data-testid="link-fallback" />
        </div>
      </div>
    </details>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { Lock, Tag, CalendarClock, ChevronDown, Eye, EyeOff } from 'lucide-vue-next'
import Tip from './ui/Tip.vue'

// model: { target_url, password, utm_source…utm_content, expires_local, fallback_url }
// password: undefined = unchanged, '' = remove, string = new password
const props = defineProps({
  modelValue: { type: Object, required: true },
  hasPassword: { type: Boolean, default: false },
  openSection: { type: String, default: '' }
})
const model = computed(() => props.modelValue)
const show = ref(false)

const utmFields = [
  { key: 'utm_source', label: 'Source', placeholder: 'newsletter', tip: 'Where the traffic comes from: google, newsletter, instagram.' },
  { key: 'utm_medium', label: 'Medium', placeholder: 'email', tip: 'The marketing medium: email, cpc, social, print.' },
  { key: 'utm_campaign', label: 'Campaign', placeholder: 'spring_sale', tip: 'The campaign or promotion name.' },
  { key: 'utm_term', label: 'Term', placeholder: 'running shoes', tip: 'Paid search keywords (optional).' },
  { key: 'utm_content', label: 'Content', placeholder: 'hero_button', tip: 'Distinguishes ads or links pointing to the same URL (optional).' }
]
const presets = [
  { label: 'Newsletter', utm_source: 'newsletter', utm_medium: 'email' },
  { label: 'Instagram', utm_source: 'instagram', utm_medium: 'social' },
  { label: 'LinkedIn', utm_source: 'linkedin', utm_medium: 'social' },
  { label: 'Print / QR', utm_source: 'qr', utm_medium: 'print' },
  { label: 'Paid search', utm_source: 'google', utm_medium: 'cpc' }
]

const utmCount = computed(() => utmFields.filter((f) => model.value[f.key]).length)
const hasUTM = computed(() => utmCount.value > 0)

function applyPreset(p) {
  model.value.utm_source = p.utm_source
  model.value.utm_medium = p.utm_medium
}

const preview = computed(() => {
  if (!hasUTM.value || !model.value.target_url) return ''
  try {
    const u = new URL(model.value.target_url)
    utmFields.forEach((f) => {
      if (model.value[f.key] && !u.searchParams.get(f.key)) u.searchParams.set(f.key, model.value[f.key])
    })
    return u.toString()
  } catch {
    return ''
  }
})
</script>
