<template>
  <div>
    <PageHeader title="Your account" subtitle="Profile, sign-in security, API keys and notifications." />

    <nav class="mb-6 flex gap-1 overflow-x-auto border-b border-line" aria-label="Account sections">
      <router-link
        v-for="t in tabs"
        :key="t.id"
        :to="{ query: { tab: t.id } }"
        class="-mb-px flex shrink-0 items-center gap-2 border-b-2 px-3 py-2.5 text-sm font-medium transition"
        :class="tab === t.id ? 'border-brand-500 text-fg' : 'border-transparent text-fg-muted hover:text-fg'"
        :data-testid="`profile-tab-${t.id}`"
      >
        <component :is="t.icon" class="h-4 w-4" />{{ t.label }}
        <span v-if="t.badge" class="badge-brand">{{ t.badge }}</span>
      </router-link>
    </nav>

    <!-- Profile -->
    <section v-if="tab === 'profile'" class="card max-w-2xl p-5 sm:p-6">
      <form class="space-y-5" @submit.prevent="saveProfile">
        <div class="flex items-center gap-4">
          <img v-if="form.avatar_url" :src="form.avatar_url" alt="" class="h-16 w-16 rounded-full object-cover" />
          <span v-else class="flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-brand-500 to-violet-500 text-lg font-bold text-white">{{ auth.initials }}</span>
          <div class="flex gap-2">
            <label class="btn-secondary btn-sm cursor-pointer">
              <Upload class="h-3.5 w-3.5" /> Upload photo
              <input type="file" accept="image/png,image/jpeg,image/webp" class="sr-only" @change="onAvatar" />
            </label>
            <button v-if="form.avatar_url" type="button" class="btn-ghost btn-sm" @click="form.avatar_url = ''">Remove</button>
          </div>
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="label" for="p-first">First name</label>
            <input id="p-first" v-model="form.first_name" maxlength="80" placeholder="Jane" class="input" data-testid="profile-first-name" />
          </div>
          <div>
            <label class="label" for="p-last">Last name</label>
            <input id="p-last" v-model="form.last_name" maxlength="80" placeholder="Doe" class="input" />
          </div>
        </div>
        <div>
          <label class="label">Email <Tip text="Your sign-in email is managed by your identity provider." /></label>
          <input :value="auth.user?.email" disabled class="input" />
        </div>
        <div class="flex justify-end">
          <button class="btn-primary" :disabled="saving" data-testid="save-profile"><Loader2 v-if="saving" class="h-4 w-4 animate-spin" /> Save changes</button>
        </div>
      </form>
    </section>

    <!-- Security -->
    <div v-else-if="tab === 'security'" class="max-w-3xl space-y-6">
      <section class="card p-5 sm:p-6">
        <h2 class="text-sm font-semibold text-fg">Change password</h2>
        <p class="mb-4 text-xs text-fg-muted">Changing your password signs you out everywhere else.</p>
        <form class="grid gap-4 sm:grid-cols-3" @submit.prevent="changePassword">
          <input v-model="pw.current" type="password" required autocomplete="current-password" placeholder="Current password" class="input" aria-label="Current password" />
          <input v-model="pw.next" type="password" required minlength="8" autocomplete="new-password" placeholder="New password" class="input" aria-label="New password" />
          <input v-model="pw.confirm" type="password" required autocomplete="new-password" placeholder="Confirm new password" class="input" aria-label="Confirm new password" />
          <div class="sm:col-span-3 flex items-center justify-between">
            <router-link :to="{ path: '/reset-password', query: { email: auth.user?.email } }" class="text-xs text-brand-500 hover:underline">Forgot your password? Send a reset link</router-link>
            <button class="btn-primary" :disabled="saving">Update password</button>
          </div>
        </form>
      </section>

      <section class="card overflow-hidden">
        <header class="flex items-center justify-between border-b border-line px-5 py-4">
          <div>
            <h2 class="text-sm font-semibold text-fg">Active sessions</h2>
            <p class="text-xs text-fg-muted">Devices currently signed in to your account.</p>
          </div>
          <button v-if="sessions.length > 1" class="btn-danger-ghost btn-sm" @click="revokeOthers">Sign out other sessions</button>
        </header>
        <ul class="divide-y divide-line" data-testid="session-list">
          <li v-for="s in sessions" :key="s.id" class="flex items-center gap-3 px-5 py-3">
            <component :is="deviceIcon(s.user_agent)" class="h-5 w-5 text-fg-subtle" />
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm text-fg">{{ describeUA(s.user_agent) }} <span v-if="s.current" class="badge-success ml-1">This device</span></p>
              <p class="text-xs text-fg-subtle">{{ s.ip || 'Unknown IP' }} · active {{ new Date(s.last_seen_at).toLocaleString() }}</p>
            </div>
            <button v-if="!s.current" class="btn-ghost btn-sm" @click="revokeSession(s)">Sign out</button>
          </li>
        </ul>
      </section>
    </div>

    <!-- API keys -->
    <div v-else-if="tab === 'api-keys'" class="space-y-6">
      <section class="card overflow-hidden">
        <header class="flex flex-col gap-3 border-b border-line px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-sm font-semibold text-fg">API keys for {{ auth.team?.name }}</h2>
            <p class="text-xs text-fg-muted">Keys act as you, with your current role in this workspace. Send them as <code class="font-mono">Authorization: Bearer lk_live_…</code></p>
          </div>
          <button class="btn-primary btn-sm" data-testid="new-api-key" @click="keyModal = true"><Plus class="h-3.5 w-3.5" /> New API key</button>
        </header>
        <EmptyState v-if="!apiKeys.length" :icon="KeyRound" title="No API keys" description="Create a key to use the REST API from scripts, CI or your backend." />
        <table v-else class="table-base" data-testid="api-key-table">
          <thead><tr><th>Name</th><th>Key</th><th class="hidden md:table-cell">Created</th><th class="hidden md:table-cell">Last used</th><th class="hidden sm:table-cell">Expires</th><th class="w-px" /></tr></thead>
          <tbody>
            <tr v-for="k in apiKeys" :key="k.id">
              <td class="font-medium text-fg">{{ k.name }}</td>
              <td class="font-mono text-xs text-fg-muted">{{ k.key_prefix }}</td>
              <td class="hidden md:table-cell text-fg-muted">{{ new Date(k.created_at).toLocaleDateString() }}</td>
              <td class="hidden md:table-cell text-fg-muted">{{ k.last_used_at ? new Date(k.last_used_at).toLocaleString() : 'Never' }}</td>
              <td class="hidden sm:table-cell text-fg-muted">{{ k.expires_at ? new Date(k.expires_at).toLocaleDateString() : 'Never' }}</td>
              <td><button class="btn-danger-ghost btn-sm" :aria-label="`Revoke ${k.name}`" data-testid="revoke-key" @click="revokeKey(k)">Revoke</button></td>
            </tr>
          </tbody>
        </table>
      </section>
      <router-link to="/api-docs" class="inline-flex items-center gap-1.5 text-sm text-brand-500 hover:underline"><BookOpen class="h-4 w-4" /> Read the API reference</router-link>
    </div>

    <!-- Workspaces -->
    <div v-else-if="tab === 'workspaces'" class="max-w-3xl space-y-6">
      <section v-if="myInvites.length" class="card overflow-hidden border-brand-500/30">
        <header class="border-b border-line px-5 py-4"><h2 class="text-sm font-semibold text-fg">Invitations for you</h2></header>
        <ul class="divide-y divide-line" data-testid="my-invitations">
          <li v-for="inv in myInvites" :key="inv.id" class="flex flex-col gap-3 px-5 py-3 sm:flex-row sm:items-center">
            <div class="flex-1">
              <p class="text-sm font-medium text-fg">{{ inv.team_name }}</p>
              <p class="text-xs text-fg-subtle">Join as <span class="capitalize">{{ inv.role }}</span> · expires {{ new Date(inv.expires_at).toLocaleDateString() }}</p>
            </div>
            <div class="flex gap-2">
              <button class="btn-secondary btn-sm" @click="respond(inv, false)">Decline</button>
              <button class="btn-primary btn-sm" @click="respond(inv, true)">Accept</button>
            </div>
          </li>
        </ul>
      </section>
      <section class="card overflow-hidden">
        <header class="border-b border-line px-5 py-4"><h2 class="text-sm font-semibold text-fg">Your workspaces</h2></header>
        <ul class="divide-y divide-line">
          <li v-for="t in auth.teams" :key="t.id" class="flex items-center gap-3 px-5 py-3">
            <span class="flex h-9 w-9 items-center justify-center rounded-lg bg-subtle text-sm font-bold text-fg">{{ t.name.charAt(0).toUpperCase() }}</span>
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-fg">{{ t.name }} <span v-if="t.id === auth.team?.id" class="badge-success ml-1">Current</span></p>
              <p class="text-xs text-fg-subtle"><span class="capitalize">{{ t.role }}</span> · {{ t.member_count }} member{{ t.member_count === 1 ? '' : 's' }}<span v-if="t.status !== 'active'" class="text-rose-500"> · suspended</span></p>
            </div>
            <button v-if="t.id !== auth.team?.id && t.status === 'active'" class="btn-secondary btn-sm" @click="auth.switchTeam(t.id)">Switch</button>
          </li>
        </ul>
      </section>
    </div>

    <!-- Notifications -->
    <section v-else-if="tab === 'notifications'" class="card max-w-2xl p-5 sm:p-6">
      <h2 class="text-sm font-semibold text-fg">Email notifications</h2>
      <p class="mb-4 text-xs text-fg-muted">Choose what Lynqr emails you about. Security emails can't be turned off for your protection.</p>
      <div v-if="!prefsLoaded" class="space-y-3"><div v-for="i in 4" :key="i" class="skeleton h-10" /></div>
      <ul v-else class="divide-y divide-line">
        <li v-for="n in notificationOptions" :key="n.key" class="flex items-center justify-between gap-4 py-3">
          <div>
            <p class="text-sm font-medium text-fg">{{ n.label }}</p>
            <p class="text-xs text-fg-muted">{{ n.description }}</p>
          </div>
          <Toggle :model-value="prefs[n.key]" :disabled="n.locked" :data-testid="`pref-${n.key}`" @update:model-value="setPref(n.key, $event)" />
        </li>
      </ul>
      <div class="mt-4">
        <label class="label" for="tz">Report time zone</label>
        <select id="tz" :value="prefs.default_timezone" class="input sm:w-72" @change="setPref('default_timezone', $event.target.value)">
          <option value="">Use my browser's time zone ({{ browserTZ }})</option>
          <option v-for="z in timezones" :key="z" :value="z">{{ z }}</option>
        </select>
      </div>
    </section>

    <!-- Create API key -->
    <AppModal :open="keyModal" title="Create API key" @close="closeKeyModal">
      <div v-if="newKey" class="space-y-3">
        <div class="flex items-start gap-3 rounded-xl bg-amber-500/10 p-3 text-sm text-amber-700 dark:text-amber-300">
          <AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" /> Copy this key now. For your security it won't be shown again.
        </div>
        <div class="flex items-center gap-2 rounded-xl border border-line bg-subtle/60 p-2">
          <code class="flex-1 break-all px-1 font-mono text-xs text-fg" data-testid="new-api-key-value">{{ newKey }}</code>
          <CopyButton :value="newKey" label="Copy API key" />
        </div>
      </div>
      <form v-else id="key-form" class="space-y-4" @submit.prevent="createKey">
        <div>
          <label class="label" for="key-name">Name</label>
          <input id="key-name" v-model="keyName" required maxlength="80" placeholder="e.g. Production backend" class="input" data-testid="api-key-name" />
        </div>
        <div>
          <label class="label" for="key-exp">Expires</label>
          <select id="key-exp" v-model.number="keyExpiry" class="input">
            <option :value="30">In 30 days</option><option :value="90">In 90 days</option><option :value="365">In 1 year</option><option :value="0">Never</option>
          </select>
        </div>
      </form>
      <template #footer>
        <button v-if="newKey" class="btn-primary" data-testid="api-key-done" @click="closeKeyModal">I've saved this key</button>
        <template v-else>
          <button class="btn-secondary" @click="closeKeyModal">Cancel</button>
          <button class="btn-primary" type="submit" form="key-form" :disabled="saving" data-testid="create-api-key">Create key</button>
        </template>
      </template>
    </AppModal>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import {
  UserCircle, Shield, KeyRound, Building2, Bell, Upload, Loader2, Plus, BookOpen, AlertTriangle, Monitor, Smartphone, Tablet
} from 'lucide-vue-next'
import { api } from '../api/client'
import { useAuthStore } from '../stores/auth'
import { useToast } from '../composables/useToast'
import { useConfirm } from '../composables/useConfirm'
import PageHeader from '../components/ui/PageHeader.vue'
import AppModal from '../components/ui/AppModal.vue'
import EmptyState from '../components/ui/EmptyState.vue'
import CopyButton from '../components/ui/CopyButton.vue'
import Toggle from '../components/ui/Toggle.vue'
import Tip from '../components/ui/Tip.vue'

const auth = useAuthStore()
const route = useRoute()
const toast = useToast()
const confirm = useConfirm()

const myInvites = ref([])
const tabs = computed(() => [
  { id: 'profile', label: 'Profile', icon: UserCircle },
  { id: 'security', label: 'Security', icon: Shield },
  { id: 'api-keys', label: 'API keys', icon: KeyRound },
  { id: 'workspaces', label: 'Workspaces', icon: Building2, badge: myInvites.value.length || null },
  { id: 'notifications', label: 'Notifications', icon: Bell }
])
const tab = computed(() => (tabs.value.some((t) => t.id === route.query.tab) ? route.query.tab : 'profile'))

const saving = ref(false)
async function busy(fn) {
  saving.value = true
  try {
    return await fn()
  } catch (err) {
    toast.error(err.message)
  } finally {
    saving.value = false
  }
}

// ----- Profile -----
const form = ref({ first_name: '', last_name: '', avatar_url: '' })
watch(
  () => auth.user,
  (u) => u && (form.value = { first_name: u.first_name, last_name: u.last_name, avatar_url: u.avatar_url || '' }),
  { immediate: true }
)

function onAvatar(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  const img = new Image()
  img.onload = () => {
    // Center-crop and downscale to 256px so avatars stay small.
    const size = 256
    const c = document.createElement('canvas')
    c.width = c.height = size
    const s = Math.min(img.width, img.height)
    c.getContext('2d').drawImage(img, (img.width - s) / 2, (img.height - s) / 2, s, s, 0, 0, size, size)
    form.value.avatar_url = c.toDataURL('image/jpeg', 0.85)
    URL.revokeObjectURL(img.src)
  }
  img.src = URL.createObjectURL(file)
}

const saveProfile = () =>
  busy(async () => {
    await api.updateProfile(form.value)
    await auth.refresh()
    toast.success('Profile updated successfully')
  })

// ----- Security -----
const pw = ref({ current: '', next: '', confirm: '' })
const sessions = ref([])
const loadSessions = async () => (sessions.value = (await api.getSessions()).data)

const changePassword = () =>
  busy(async () => {
    if (pw.value.next !== pw.value.confirm) throw new Error('New passwords do not match')
    const res = await api.changePassword(pw.value.current, pw.value.next)
    pw.value = { current: '', next: '', confirm: '' }
    toast.success(res.message)
    loadSessions()
  })

async function revokeSession(s) {
  await busy(() => api.revokeSession(s.id))
  loadSessions()
}
async function revokeOthers() {
  const ok = await confirm({ title: 'Sign out everywhere else?', message: 'All other browsers and devices will need to sign in again.', confirmText: 'Sign out others' })
  if (!ok) return
  await busy(() => api.revokeOtherSessions())
  toast.success('Signed out of other sessions')
  loadSessions()
}

function describeUA(ua = '') {
  const browser = /edg/i.test(ua) ? 'Edge' : /chrome|crios/i.test(ua) ? 'Chrome' : /firefox|fxios/i.test(ua) ? 'Firefox' : /safari/i.test(ua) ? 'Safari' : 'Browser'
  const os = /iphone|ipad/i.test(ua) ? 'iOS' : /android/i.test(ua) ? 'Android' : /mac os/i.test(ua) ? 'macOS' : /windows/i.test(ua) ? 'Windows' : /linux/i.test(ua) ? 'Linux' : ''
  return os ? `${browser} on ${os}` : ua ? browser : 'API client'
}
const deviceIcon = (ua = '') => (/ipad|tablet/i.test(ua) ? Tablet : /mobile|iphone|android/i.test(ua) ? Smartphone : Monitor)

// ----- API keys -----
const apiKeys = ref([])
const keyModal = ref(false)
const keyName = ref('')
const keyExpiry = ref(90)
const newKey = ref('')
const loadKeys = async () => (apiKeys.value = (await api.getApiKeys()).api_keys)

const createKey = () =>
  busy(async () => {
    const res = await api.createApiKey(keyName.value, keyExpiry.value)
    newKey.value = res.api_key.key
    keyName.value = ''
    loadKeys()
  })
function closeKeyModal() {
  keyModal.value = false
  newKey.value = ''
}
async function revokeKey(k) {
  const ok = await confirm({ title: `Revoke "${k.name}"?`, message: 'Anything using this key loses access immediately.', confirmText: 'Revoke key', danger: true })
  if (!ok) return
  await busy(() => api.revokeApiKey(k.id))
  toast.success('API key revoked')
  loadKeys()
}

// ----- Workspaces -----
const loadInvites = async () => (myInvites.value = (await api.getMyInvitations()).data)
async function respond(inv, accept) {
  await busy(async () => {
    await api.respondToInvitation(inv.id, accept)
    toast.success(accept ? `You joined ${inv.team_name}` : 'Invitation declined')
    await auth.refresh()
    if (accept) await auth.switchTeam(inv.team_id)
    loadInvites()
  })
}

// ----- Notifications -----
const prefs = ref({})
const prefsLoaded = ref(false)
const browserTZ = Intl.DateTimeFormat().resolvedOptions().timeZone
const timezones = typeof Intl.supportedValuesOf === 'function' ? Intl.supportedValuesOf('timeZone') : []
const notificationOptions = [
  { key: 'security_alerts', label: 'Security alerts', description: 'Password changes and sign-ins from new devices.', locked: true },
  { key: 'team_activity', label: 'Team activity', description: 'Invitations, role changes and removals in your workspaces.' },
  { key: 'usage_alerts', label: 'Trial & billing', description: 'Trial reminders, payment issues and plan changes.' },
  { key: 'weekly_report', label: 'Weekly analytics digest', description: 'Top links, QR codes and traffic trends every Monday.' },
  { key: 'product_updates', label: 'Product updates', description: 'Occasional news about new Lynqr features.' }
]
async function setPref(key, value) {
  const prev = prefs.value[key]
  prefs.value = { ...prefs.value, [key]: value }
  try {
    prefs.value = await api.updatePreferences(prefs.value)
    toast.success('Preferences saved')
  } catch (err) {
    prefs.value = { ...prefs.value, [key]: prev }
    toast.error(err.message)
  }
}

watch(
  tab,
  (t) => {
    if (t === 'security') loadSessions()
    if (t === 'api-keys') loadKeys()
    if (t === 'notifications')
      api.getPreferences().then((p) => {
        prefs.value = p
        prefsLoaded.value = true
      })
  },
  { immediate: true }
)
watch(() => auth.team?.id, () => tab.value === 'api-keys' && loadKeys())
onMounted(loadInvites)
</script>
