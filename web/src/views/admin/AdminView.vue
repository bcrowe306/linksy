<template>
  <div>
    <div v-if="!status" class="skeleton h-40" />

    <!-- Not staff -->
    <div v-else-if="!status.is_admin" class="card mx-auto max-w-md p-8 text-center" data-testid="admin-denied">
      <ShieldOff class="mx-auto mb-3 h-8 w-8 text-fg-subtle" />
      <h1 class="text-base font-semibold text-fg">Admin console</h1>
      <p class="mt-1 text-sm text-fg-muted">This area is limited to Lynqr staff.</p>
      <router-link to="/" class="btn-secondary mt-5">Back to dashboard</router-link>
    </div>

    <!-- Confirm identity -->
    <div v-else-if="!elevated" class="card mx-auto max-w-md p-6 sm:p-8" data-testid="admin-reauth">
      <div class="mb-5 flex items-center gap-3">
        <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-500/10 text-brand-500"><ShieldCheck class="h-5 w-5" /></div>
        <div>
          <h1 class="text-base font-semibold text-fg">Confirm it's you</h1>
          <p class="text-xs text-fg-muted">The admin console unlocks for {{ ttlLabel }} on this device.</p>
        </div>
      </div>

      <form v-if="mode === 'password'" class="space-y-3" @submit.prevent="submitPassword">
        <label class="label" for="admin-password">Password for {{ auth.user?.email }}</label>
        <input id="admin-password" v-model="password" type="password" autocomplete="current-password" required class="input" data-testid="admin-password" />
        <button class="btn-primary w-full justify-center" :disabled="busy || !password">Unlock admin console</button>
      </form>

      <form v-else class="space-y-3" @submit.prevent="submitCode">
        <p v-if="codeSent" class="text-sm text-fg-muted">We emailed a 6-digit code to <span class="font-medium text-fg">{{ auth.user?.email }}</span>.</p>
        <template v-if="codeSent">
          <input v-model="code" inputmode="numeric" autocomplete="one-time-code" maxlength="6" placeholder="123456" required class="input text-center font-mono text-lg tracking-[0.4em]" data-testid="admin-code" />
          <button class="btn-primary w-full justify-center" :disabled="busy || code.length !== 6">Unlock admin console</button>
          <button type="button" class="btn-ghost btn-sm w-full justify-center" :disabled="busy" @click="sendCode">Send a new code</button>
        </template>
        <button v-else type="button" class="btn-primary w-full justify-center" :disabled="busy" data-testid="admin-send-code" @click="sendCode">Email me a code</button>
      </form>

      <p v-if="error" class="mt-3 text-sm text-rose-500" role="alert">{{ error }}</p>
      <div v-if="status.methods.password && status.methods.email_code" class="mt-4 border-t border-line pt-4 text-center">
        <button class="text-xs font-medium text-brand-500 hover:underline" @click="switchMode">
          {{ mode === 'password' ? 'Signed in with Google, Microsoft or GitHub? Use an email code' : 'Use your password instead' }}
        </button>
      </div>
    </div>

    <!-- Console -->
    <div v-else>
      <PageHeader title="Admin console" subtitle="Platform-wide tools for Lynqr staff. Every action is recorded in the platform audit log.">
        <template #actions>
          <span class="hidden items-center gap-1.5 text-xs text-fg-muted sm:inline-flex"><Clock class="h-3.5 w-3.5" /> Unlocked until {{ untilLabel }}</span>
          <button class="btn-secondary btn-sm" data-testid="admin-lock" @click="lock"><Lock class="h-3.5 w-3.5" /> Lock</button>
        </template>
      </PageHeader>
      <nav class="mb-6 flex gap-1 overflow-x-auto border-b border-line" aria-label="Admin sections">
        <router-link
          v-for="t in tabs"
          :key="t.to"
          :to="t.to"
          class="-mb-px flex shrink-0 items-center gap-2 border-b-2 px-3 py-2.5 text-sm font-medium transition"
          :class="isActive(t) ? 'border-brand-500 text-fg' : 'border-transparent text-fg-muted hover:text-fg'"
          :data-testid="`admin-tab-${t.id}`"
        >
          <component :is="t.icon" class="h-4 w-4" />{{ t.label }}
        </router-link>
      </nav>
      <router-view />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ShieldCheck, ShieldOff, Lock, Clock, Gauge, Building2, Users, TicketPercent, Flag, ScrollText } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useAuthStore } from '../../stores/auth'
import { adminState } from '../../composables/useAdmin'
import PageHeader from '../../components/ui/PageHeader.vue'

const auth = useAuthStore()
const route = useRoute()
const status = ref(null)
const mode = ref('password')
const password = ref('')
const code = ref('')
const codeSent = ref(false)
const busy = ref(false)
const error = ref('')

const elevated = computed(() => status.value?.elevated && !adminState.needsReauth)
const ttlLabel = '1 hour'
const untilLabel = computed(() => (adminState.elevatedUntil ? new Date(adminState.elevatedUntil).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' }) : ''))

const tabs = [
  { id: 'overview', to: '/admin', label: 'Overview', icon: Gauge, exact: true },
  { id: 'workspaces', to: '/admin/workspaces', label: 'Workspaces', icon: Building2 },
  { id: 'users', to: '/admin/users', label: 'Users', icon: Users },
  { id: 'codes', to: '/admin/codes', label: 'Codes', icon: TicketPercent },
  { id: 'moderation', to: '/admin/moderation', label: 'Moderation', icon: Flag },
  { id: 'audit', to: '/admin/audit', label: 'Audit log', icon: ScrollText }
]
const isActive = (t) => (t.exact ? route.path === t.to : route.path.startsWith(t.to))

async function load() {
  status.value = await api.admin.session()
  adminState.elevatedUntil = status.value.elevated_until
  adminState.needsReauth = false
  if (!status.value.methods.password) mode.value = 'code'
}

// An expired admin session (mid-use) returns to this screen.
watch(() => adminState.needsReauth, (v) => v && status.value && (status.value.elevated = false))

function switchMode() {
  mode.value = mode.value === 'password' ? 'code' : 'password'
  error.value = ''
}

async function unlock(body) {
  busy.value = true
  error.value = ''
  try {
    const res = await api.admin.reauth(body)
    status.value.elevated = true
    adminState.elevatedUntil = res.elevated_until
    adminState.needsReauth = false
    password.value = code.value = ''
    codeSent.value = false
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = false
  }
}
const submitPassword = () => unlock({ password: password.value })
const submitCode = () => unlock({ code: code.value })

async function sendCode() {
  busy.value = true
  error.value = ''
  try {
    await api.admin.sendCode()
    codeSent.value = true
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = false
  }
}

async function lock() {
  await api.admin.leave()
  status.value.elevated = false
}

onMounted(load)
</script>
