<template>
  <div>
    <PageHeader title="Short Links" subtitle="Branded, trackable links you can re-point at any time.">
      <template #title-extra><HelpLink guide="short-links" /></template>
      <template #actions>
        <button v-if="canEdit" class="btn-primary" data-testid="new-link" @click="$emit('open-create', 'link')">
          <Plus class="w-4 h-4" /> New link
        </button>
      </template>
    </PageHeader>

    <!-- Toolbar -->
    <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center">
      <div class="relative flex-1 sm:max-w-xs">
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-subtle" />
        <input v-model="search" type="search" placeholder="Filter links…" class="input pl-9" aria-label="Filter links" />
      </div>
      <div class="flex gap-1 rounded-xl bg-subtle p-1">
        <button
          v-for="f in filters"
          :key="f.id"
          class="rounded-lg px-3 py-1.5 text-xs font-medium transition"
          :class="status === f.id ? 'bg-surface text-fg shadow-card' : 'text-fg-muted hover:text-fg'"
          @click="status = f.id"
        >
          {{ f.label }} <span class="ml-1 text-fg-subtle">{{ counts[f.id] }}</span>
        </button>
      </div>
      <select v-model="sort" class="input sm:ml-auto sm:w-44" aria-label="Sort links">
        <option value="newest">Newest first</option>
        <option value="oldest">Oldest first</option>
        <option value="clicks">Most clicks</option>
        <option value="title">Title A–Z</option>
      </select>
    </div>

    <div class="card overflow-hidden">
      <div v-if="store.loading && !store.links.length" class="space-y-3 p-5">
        <div v-for="i in 4" :key="i" class="skeleton h-12" />
      </div>

      <EmptyState
        v-else-if="!store.links.length"
        :icon="Link2"
        title="No short links yet"
        description="Create a short link to start tracking clicks, devices and locations."
      >
        <button v-if="canEdit" class="btn-primary" @click="$emit('open-create', 'link')"><Plus class="w-4 h-4" /> Create your first link</button>
      </EmptyState>

      <EmptyState v-else-if="!visible.length" :icon="SearchX" title="No links match your filters" description="Try a different search or status filter." />

      <template v-else>
        <!-- Desktop table -->
        <table class="table-base hidden md:table" data-testid="links-table">
          <thead>
            <tr>
              <th>Link</th>
              <th>Destination</th>
              <th class="text-right">Clicks</th>
              <th>Status</th>
              <th>Created</th>
              <th class="w-px"><span class="sr-only">Actions</span></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="link in visible" :key="link.id" :class="{ 'bg-brand-500/5': link.id === route.query.focus }">
              <td class="max-w-[260px]">
                <p class="flex items-center gap-1.5 font-medium text-fg">
                  <span class="truncate">{{ link.title || link.slug }}</span>
                  <Tip v-if="link.has_password" text="Password protected"><Lock class="h-3.5 w-3.5 shrink-0 text-fg-subtle" /></Tip>
                  <Tip v-if="hasUTM(link)" :text="utmSummary(link)"><Tag class="h-3.5 w-3.5 shrink-0 text-fg-subtle" /></Tip>
                  <Tip v-if="qrCount(link)" :text="`${qrCount(link)} QR code${qrCount(link) > 1 ? 's' : ''} use this link`"><QrCode class="h-3.5 w-3.5 shrink-0 text-brand-500" /></Tip>
                </p>
                <div class="mt-0.5 flex items-center gap-1">
                  <a :href="shortUrl(link)" target="_blank" rel="noopener" class="truncate font-mono text-xs text-brand-500 hover:underline">{{ shortHost }}/s/{{ link.slug }}</a>
                  <CopyButton :value="shortUrl(link)" label="Copy short link" />
                </div>
              </td>
              <td class="max-w-[280px]">
                <a :href="link.target_url" target="_blank" rel="noopener noreferrer" class="block truncate text-fg-muted hover:text-fg" :title="link.target_url">{{ link.target_url }}</a>
              </td>
              <td class="text-right font-semibold tabular-nums text-fg">{{ link.click_count.toLocaleString() }}</td>
              <td><StatusBadge :link="link" /></td>
              <td class="whitespace-nowrap text-fg-muted">{{ formatDate(link.created_at) }}</td>
              <td>
                <div class="flex items-center justify-end gap-0.5">
                  <Tip text="View analytics"><router-link :to="`/analytics?link=${link.id}`" class="btn-ghost btn-icon" aria-label="View analytics"><BarChart3 class="w-4 h-4" /></router-link></Tip>
                  <template v-if="canEdit">
                    <Tip :text="link.is_active ? 'Pause link' : 'Resume link'">
                      <button class="btn-ghost btn-icon" :aria-label="link.is_active ? 'Pause link' : 'Resume link'" @click="toggleActive(link)">
                        <Pause v-if="link.is_active" class="w-4 h-4" /><Play v-else class="w-4 h-4" />
                      </button>
                    </Tip>
                    <Tip text="Create a QR code for this link"><button class="btn-ghost btn-icon" aria-label="Create QR code" data-testid="link-create-qr" @click="createQR(link)"><QrCode class="w-4 h-4" /></button></Tip>
                    <Tip text="Edit"><button class="btn-ghost btn-icon" aria-label="Edit link" data-testid="edit-link" @click="openEdit(link)"><Pencil class="w-4 h-4" /></button></Tip>
                    <Tip text="Delete"><button class="btn-ghost btn-icon hover:!text-rose-500" aria-label="Delete link" @click="remove(link)"><Trash2 class="w-4 h-4" /></button></Tip>
                  </template>
                </div>
              </td>
            </tr>
          </tbody>
        </table>

        <!-- Mobile cards -->
        <ul class="divide-y divide-line md:hidden">
          <li v-for="link in visible" :key="link.id" class="p-4">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="truncate font-medium text-fg">{{ link.title || link.slug }}</p>
                <p class="truncate font-mono text-xs text-brand-500">{{ shortHost }}/s/{{ link.slug }}</p>
                <p class="mt-1 truncate text-xs text-fg-muted">{{ link.target_url }}</p>
              </div>
              <StatusBadge :link="link" />
            </div>
            <div class="mt-3 flex items-center justify-between">
              <span class="text-xs text-fg-muted"><span class="font-semibold text-fg">{{ link.click_count }}</span> clicks</span>
              <div class="flex gap-1">
                <CopyButton :value="shortUrl(link)" />
                <button v-if="canEdit" class="btn-ghost btn-icon" aria-label="Edit link" @click="openEdit(link)"><Pencil class="w-4 h-4" /></button>
                <button v-if="canEdit" class="btn-ghost btn-icon" aria-label="Delete link" @click="remove(link)"><Trash2 class="w-4 h-4" /></button>
              </div>
            </div>
          </li>
        </ul>
      </template>
    </div>

    <!-- Edit modal -->
    <AppModal :open="!!editing" title="Edit short link" :description="editing ? `${shortHost}/s/${editing.slug}` : ''" @close="editing = null">
      <form v-if="editing" id="edit-link-form" class="space-y-4" @submit.prevent="saveEdit">
        <div>
          <label class="label" for="edit-target">
            Destination URL
            <Tip text="Changing this re-points the link (and any QR codes using it) instantly. Nothing needs to be reprinted." />
          </label>
          <input id="edit-target" v-model="form.target_url" type="url" required class="input" data-testid="edit-target" />
        </div>
        <div>
          <label class="label" for="edit-slug">Back-half <Tip text="Changing it breaks the old short URL. Printed QR codes are unaffected." /></label>
          <div class="flex items-stretch overflow-hidden rounded-xl border border-line focus-within:border-brand-500 focus-within:ring-4 focus-within:ring-brand-500/15">
            <span class="flex items-center bg-subtle px-3 font-mono text-sm text-fg-subtle">{{ shortHost }}/s/</span>
            <input id="edit-slug" v-model="form.slug" required pattern="[A-Za-z0-9][A-Za-z0-9_\-]{2,63}" class="flex-1 bg-surface px-3 py-2 text-sm text-fg outline-none" data-testid="edit-slug" />
          </div>
          <p v-if="form.slug !== editing.slug" class="hint text-amber-600 dark:text-amber-400">{{ shortHost }}/s/{{ editing.slug }} will stop working.</p>
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="label" for="edit-title">Title</label>
            <input id="edit-title" v-model="form.title" class="input" />
          </div>
          <label class="flex items-center justify-between gap-3 self-end rounded-xl border border-line px-3.5 py-2">
            <span class="text-sm font-medium text-fg">Active <Tip text="Paused links show a branded notice, or send visitors to the fallback URL." /></span>
            <Toggle v-model="form.is_active" />
          </label>
        </div>
        <div>
          <label class="label" for="edit-desc">Notes</label>
          <textarea id="edit-desc" v-model="form.description" rows="2" class="input" />
        </div>
        <LinkOptionsFields v-model="form" :has-password="editing.has_password" />
      </form>
      <template #footer>
        <button class="btn-secondary" @click="editing = null">Cancel</button>
        <button class="btn-primary" type="submit" form="edit-link-form" :disabled="saving" data-testid="save-link">
          <Loader2 v-if="saving" class="w-4 h-4 animate-spin" /> Save changes
        </button>
      </template>
    </AppModal>
  </div>
</template>

<script setup>
import HelpLink from '../components/ui/HelpLink.vue'
import { ref, computed, h } from 'vue'
import { useRoute } from 'vue-router'
import { Plus, Link2, Search, SearchX, Pencil, Trash2, Pause, Play, BarChart3, Loader2, Lock, Tag, QrCode } from 'lucide-vue-next'
import { useLynqrStore } from '../stores/lynqr'
import { useAuthStore } from '../stores/auth'
import { api } from '../api/client'
import { useToast } from '../composables/useToast'
import { useConfirm } from '../composables/useConfirm'
import PageHeader from '../components/ui/PageHeader.vue'
import EmptyState from '../components/ui/EmptyState.vue'
import AppModal from '../components/ui/AppModal.vue'
import CopyButton from '../components/ui/CopyButton.vue'
import Tip from '../components/ui/Tip.vue'
import Toggle from '../components/ui/Toggle.vue'
import LinkOptionsFields from '../components/LinkOptionsFields.vue'
import { linkToForm, formToPayload } from '../lib/linkForm'
import { useRouter } from 'vue-router'

defineEmits(['open-create'])
const store = useLynqrStore()
const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const toast = useToast()
const confirm = useConfirm()

const canEdit = computed(() => auth.can('assets:edit'))
const shortHost = window.location.host
const shortUrl = (l) => `${window.location.origin}/s/${l.slug}`

const search = ref('')
const status = ref('all')
const sort = ref('newest')
const filters = [
  { id: 'all', label: 'All' },
  { id: 'active', label: 'Active' },
  { id: 'paused', label: 'Paused' },
  { id: 'expired', label: 'Expired' }
]

const isExpired = (l) => l.expires_at && new Date(l.expires_at) < new Date()
const stateOf = (l) => (isExpired(l) ? 'expired' : l.is_active ? 'active' : 'paused')

const counts = computed(() => {
  const c = { all: store.links.length, active: 0, paused: 0, expired: 0 }
  store.links.forEach((l) => c[stateOf(l)]++)
  return c
})

const visible = computed(() => {
  const q = search.value.trim().toLowerCase()
  let list = store.links.filter((l) => status.value === 'all' || stateOf(l) === status.value)
  if (q) list = list.filter((l) => [l.title, l.slug, l.target_url].some((f) => (f || '').toLowerCase().includes(q)))
  const sorters = {
    newest: (a, b) => new Date(b.created_at) - new Date(a.created_at),
    oldest: (a, b) => new Date(a.created_at) - new Date(b.created_at),
    clicks: (a, b) => b.click_count - a.click_count,
    title: (a, b) => (a.title || a.slug).localeCompare(b.title || b.slug)
  }
  return [...list].sort(sorters[sort.value])
})

const StatusBadge = (props) => {
  const s = stateOf(props.link)
  const cls = { active: 'badge-success', paused: 'badge-warning', expired: 'badge-danger' }[s]
  return h('span', { class: cls }, [h('span', { class: 'h-1.5 w-1.5 rounded-full bg-current' }), s[0].toUpperCase() + s.slice(1)])
}
StatusBadge.props = ['link']

function formatDate(d) {
  return new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}

const UTM_KEYS = ['utm_source', 'utm_medium', 'utm_campaign', 'utm_term', 'utm_content']
const hasUTM = (l) => UTM_KEYS.some((k) => l[k])
const utmSummary = (l) => UTM_KEYS.filter((k) => l[k]).map((k) => `${k.slice(4)}=${l[k]}`).join(' · ')
const qrCount = (l) => store.qrCodes.filter((q) => q.link_id === l.id).length

async function createQR(link) {
  try {
    const qr = await api.createQRCode({ type: 'website', title: link.title || link.slug, link_id: link.id })
    toast.success('Style it, then download', { title: 'QR code created' })
    store.fetchQRCodes()
    router.push(`/qr-studio/${qr.id}`)
  } catch (err) {
    toast.error(err.message)
  }
}

const editing = ref(null)
const saving = ref(false)
const form = ref({})

function openEdit(link) {
  editing.value = link
  form.value = linkToForm(link)
}

// Toggling only touches the active flag; omitted optional fields stay unchanged server-side.
function payloadFor(link, overrides = {}) {
  return { title: link.title, target_url: link.target_url, description: link.description, is_active: link.is_active, expires_at: link.expires_at || null, ...overrides }
}

async function saveEdit() {
  saving.value = true
  try {
    await api.updateLink(editing.value.id, formToPayload(form.value))
    toast.success('Link updated')
    editing.value = null
    await store.fetchLinks()
  } catch (err) {
    toast.error(err.message)
  } finally {
    saving.value = false
  }
}

async function toggleActive(link) {
  try {
    await api.updateLink(link.id, payloadFor(link, { is_active: !link.is_active }))
    toast.success(link.is_active ? 'Link paused: visitors see a branded notice' : 'Link resumed')
    await store.fetchLinks()
  } catch (err) {
    toast.error(err.message)
  }
}

async function remove(link) {
  const ok = await confirm({
    title: 'Delete short link?',
    message: `${shortHost}/s/${link.slug} will stop working immediately. Printed materials using it will show a "not found" page.`,
    confirmText: 'Delete link',
    danger: true
  })
  if (!ok) return
  try {
    await api.deleteLink(link.id)
    toast.success('Link deleted')
    await store.fetchLinks()
  } catch (err) {
    toast.error(err.message)
  }
}
</script>
