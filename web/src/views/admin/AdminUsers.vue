<template>
  <section class="card overflow-hidden">
    <header class="flex flex-col gap-3 border-b border-line px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h2 class="text-sm font-semibold text-fg">Users</h2>
        <p class="text-xs text-fg-muted">Search by email, name or user ID.</p>
      </div>
      <div class="relative sm:w-80">
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-subtle" />
        <input v-model="q" type="search" placeholder="Search users" class="input input-sm pl-9" data-testid="admin-user-search" />
      </div>
    </header>

    <div v-if="loading && !rows.length" class="space-y-3 p-5"><div v-for="i in 6" :key="i" class="skeleton h-10" /></div>
    <EmptyState v-else-if="!rows.length" :icon="Users" title="No users found" />
    <ul v-else class="divide-y divide-line" data-testid="admin-user-list">
      <li v-for="u in rows" :key="u.id" class="flex flex-col gap-3 px-5 py-3 sm:flex-row sm:items-center">
        <div class="min-w-0 flex-1">
          <p class="flex flex-wrap items-center gap-2 text-sm font-medium text-fg">
            {{ u.email }}
            <span v-if="u.is_platform_admin" class="badge-brand">Lynqr staff</span>
            <span v-if="u.suspended_reason" class="badge-danger" :title="u.suspended_reason">Suspended</span>
          </p>
          <p class="text-xs text-fg-muted">
            {{ u.name || 'No name' }} · {{ u.workspaces }} workspace{{ u.workspaces === 1 ? '' : 's' }} · joined {{ shortDate(u.created_at) }}
            <span v-if="u.last_seen_at"> · last seen {{ shortDate(u.last_seen_at) }}</span>
          </p>
          <p v-if="u.suspended_reason" class="mt-0.5 text-xs text-rose-500">{{ u.suspended_reason }}</p>
        </div>
        <DropdownMenu v-if="u.id !== auth.user?.id" align="right">
          <template #trigger><button class="btn-ghost btn-sm" :aria-label="`Actions for ${u.email}`">Actions <ChevronDown class="h-3.5 w-3.5" /></button></template>
          <MenuItem :icon="Building2" @click="router.push({ path: '/admin/workspaces', query: { q: u.email } })">View workspaces</MenuItem>
          <MenuItem v-if="u.suspended_reason" :icon="ShieldCheck" @click="act(u, 'unsuspend')">Lift suspension</MenuItem>
          <MenuItem v-else :icon="Ban" danger @click="target = u">Suspend account</MenuItem>
          <MenuItem :icon="Shield" @click="toggleAdmin(u)">{{ u.is_platform_admin ? 'Remove staff access' : 'Make Lynqr staff' }}</MenuItem>
        </DropdownMenu>
        <span v-else class="text-xs text-fg-subtle">You</span>
      </li>
    </ul>
    <div v-if="hasMore" class="border-t border-line p-3 text-center"><button class="btn-ghost btn-sm" :disabled="loading" @click="load(true)">Load more</button></div>

    <ReasonModal :open="!!target" :title="`Suspend ${target?.email || ''}`" description="Signs them out everywhere and blocks sign-in and their API keys until you lift it." confirm-text="Suspend account" @close="target = null" @submit="suspend" />
  </section>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search, Users, Ban, ShieldCheck, Shield, Building2, ChevronDown } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useAuthStore } from '../../stores/auth'
import { useToast } from '../../composables/useToast'
import { useConfirm } from '../../composables/useConfirm'
import { useAdminCall, shortDate } from '../../composables/useAdmin'
import EmptyState from '../../components/ui/EmptyState.vue'
import DropdownMenu from '../../components/ui/DropdownMenu.vue'
import MenuItem from '../../components/ui/MenuItem.vue'
import ReasonModal from '../../components/admin/ReasonModal.vue'

const call = useAdminCall()
const toast = useToast()
const confirm = useConfirm()
const auth = useAuthStore()
const router = useRouter()
const q = ref('')
const rows = ref([])
const loading = ref(false)
const hasMore = ref(false)
const target = ref(null)

async function load(append = false) {
  loading.value = true
  try {
    const { users } = await call(() => api.admin.users(q.value.trim(), append ? rows.value.length : 0))
    rows.value = append ? [...rows.value, ...users] : users
    hasMore.value = users.length === 50
  } finally {
    loading.value = false
  }
}
let timer
watch(q, () => {
  clearTimeout(timer)
  timer = setTimeout(() => load(), 250)
})
onMounted(() => load())

async function act(u, action, body = {}) {
  const { user } = await call(() => api.admin.userAction(u.id, action, body))
  rows.value = rows.value.map((r) => (r.id === user.id ? user : r))
  toast.success('User updated')
}
async function suspend(reason) {
  const u = target.value
  target.value = null
  await act(u, 'suspend', { reason })
}
async function toggleAdmin(u) {
  const grant = !u.is_platform_admin
  const ok = await confirm({
    title: grant ? 'Make Lynqr staff?' : 'Remove staff access?',
    message: grant ? `${u.email} will be able to open this admin console (after confirming their identity).` : `${u.email} will lose access to the admin console.`,
    confirmText: grant ? 'Grant access' : 'Remove access',
    danger: !grant
  })
  if (ok) await act(u, 'admin', { admin: grant })
}
</script>
