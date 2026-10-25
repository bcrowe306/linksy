<template>
  <div class="space-y-6">
    <!-- Members -->
    <section class="card overflow-hidden">
      <header class="flex flex-col gap-3 border-b border-line px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-sm font-semibold text-fg">Members <span class="text-fg-subtle">· {{ members.length }}</span></h2>
          <p class="text-xs text-fg-muted">Everyone with access to {{ teamName }}.</p>
        </div>
        <div class="flex gap-2">
          <div class="relative">
            <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-subtle" />
            <input v-model="filter" type="search" placeholder="Filter members" class="input input-sm w-44 pl-8" aria-label="Filter members" />
          </div>
          <button v-if="canManage" class="btn-primary btn-sm" data-testid="invite-member" @click="inviteOpen = true">
            <UserPlus class="h-3.5 w-3.5" /> Invite
          </button>
        </div>
      </header>

      <div v-if="loading" class="space-y-3 p-5"><div v-for="i in 3" :key="i" class="skeleton h-12" /></div>
      <ul v-else class="divide-y divide-line" data-testid="member-list">
        <li v-for="m in visibleMembers" :key="m.user_id" class="flex flex-col gap-3 px-5 py-3.5 sm:flex-row sm:items-center">
          <div class="flex min-w-0 flex-1 items-center gap-3">
            <img v-if="m.avatar_url" :src="m.avatar_url" alt="" class="h-9 w-9 rounded-full object-cover" />
            <span v-else class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-brand-500 to-violet-500 text-xs font-bold text-white">{{ initials(m) }}</span>
            <div class="min-w-0">
              <p class="flex items-center gap-2 truncate text-sm font-medium text-fg">
                {{ m.name }}
                <span v-if="m.user_id === currentUserId" class="badge-neutral">You</span>
                <span v-if="m.status === 'suspended'" class="badge-danger">Suspended</span>
              </p>
              <p class="truncate text-xs text-fg-subtle">{{ m.email }} · joined {{ formatDate(m.created_at) }}</p>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <select
              v-if="canChangeRole(m)"
              :value="m.role"
              class="input input-sm w-32"
              :aria-label="`Role for ${m.name}`"
              data-testid="member-role"
              @change="changeRole(m, $event.target.value)"
            >
              <option v-for="r in assignableRoles" :key="r.id" :value="r.id">{{ r.label }}</option>
            </select>
            <Tip v-else :text="roleMeta(m.role).description">
              <span :class="roleMeta(m.role).badge" class="capitalize"><component :is="roleMeta(m.role).icon" class="h-3 w-3" />{{ m.role }}</span>
            </Tip>

            <DropdownMenu v-if="canManageMember(m)">
              <template #trigger>
                <button class="btn-ghost btn-icon" :aria-label="`Actions for ${m.name}`" data-testid="member-actions"><MoreHorizontal class="h-4 w-4" /></button>
              </template>
              <MenuItem v-if="m.status === 'active'" :icon="PauseCircle" @click="suspend(m)">Suspend access</MenuItem>
              <MenuItem v-else :icon="PlayCircle" @click="reactivate(m)">Reactivate</MenuItem>
              <MenuItem :icon="KeyRound" @click="resetPassword(m)">Send password reset</MenuItem>
              <MenuItem v-if="isOwner && m.status === 'active'" :icon="Crown" @click="transfer(m)">Make owner</MenuItem>
              <div class="my-1 border-t border-line" />
              <MenuItem :icon="UserMinus" danger data-testid="remove-member" @click="remove(m)">Remove from workspace</MenuItem>
            </DropdownMenu>
          </div>
        </li>
      </ul>
    </section>

    <!-- Pending invitations -->
    <section v-if="canManage" class="card overflow-hidden">
      <header class="border-b border-line px-5 py-4">
        <h2 class="text-sm font-semibold text-fg">Pending invitations <span class="text-fg-subtle">· {{ invitations.length }}</span></h2>
        <p class="text-xs text-fg-muted">Invite links expire after 7 days and only work for the invited email address.</p>
      </header>
      <p v-if="!invitations.length" class="px-5 py-6 text-center text-sm text-fg-muted">No pending invitations.</p>
      <ul v-else class="divide-y divide-line" data-testid="invitation-list">
        <li v-for="inv in invitations" :key="inv.id" class="flex flex-col gap-2 px-5 py-3 sm:flex-row sm:items-center">
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm text-fg">{{ inv.email }}</p>
            <p class="text-xs text-fg-subtle"><span class="capitalize">{{ inv.role }}</span> · expires {{ formatDate(inv.expires_at) }}</p>
          </div>
          <CopyButton v-if="inviteLinks[inv.id]" :value="inviteLinks[inv.id]" label="Copy invite link" />
          <button class="btn-danger-ghost btn-sm" @click="revoke(inv)">Revoke</button>
        </li>
      </ul>
    </section>

    <!-- Role reference -->
    <section class="card overflow-hidden">
      <header class="border-b border-line px-5 py-4">
        <h2 class="text-sm font-semibold text-fg">Roles & permissions</h2>
      </header>
      <div class="overflow-x-auto">
        <table class="table-base">
          <thead>
            <tr>
              <th>Permission</th>
              <th v-for="r in ROLES" :key="r.id" class="text-center">{{ r.label }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in PERMISSION_MATRIX" :key="p.label">
              <td class="text-fg-muted">{{ p.label }}</td>
              <td v-for="r in ROLES" :key="r.id" class="text-center">
                <Check v-if="p.roles.includes(r.id)" class="mx-auto h-4 w-4 text-emerald-500" aria-label="Allowed" />
                <Minus v-else class="mx-auto h-4 w-4 text-fg-subtle" aria-label="Not allowed" />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Invite modal -->
    <AppModal :open="inviteOpen" title="Invite to workspace" :description="`They'll join ${teamName} after signing in with this email.`" @close="closeInvite">
      <div v-if="createdInvite" class="space-y-3">
        <div class="flex items-start gap-3 rounded-xl bg-emerald-500/10 p-3 text-sm text-emerald-700 dark:text-emerald-300">
          <CheckCircle2 class="mt-0.5 h-4 w-4 shrink-0" />
          <span v-if="createdInvite.emailed">We emailed an invitation to <strong>{{ createdInvite.email }}</strong>. You can also share this link with them; it's shown only once.</span>
          <span v-else>Invitation created for <strong>{{ createdInvite.email }}</strong>. Share this link with them. It's shown only once.</span>
        </div>
        <div class="flex items-center gap-2 rounded-xl border border-line bg-subtle/60 p-2">
          <code class="flex-1 truncate px-1 text-xs text-fg" data-testid="invite-url">{{ createdInvite.accept_url }}</code>
          <CopyButton :value="createdInvite.accept_url" label="Copy invite link" />
        </div>
      </div>
      <form v-else id="invite-form" class="space-y-4" @submit.prevent="invite">
        <div>
          <label class="label" for="invite-email">Email address</label>
          <input id="invite-email" v-model="inviteEmail" type="email" required placeholder="teammate@company.com" class="input" data-testid="invite-email" />
        </div>
        <div>
          <label class="label">Role</label>
          <div class="space-y-2">
            <label v-for="r in assignableRoles" :key="r.id" class="flex cursor-pointer items-start gap-3 rounded-xl border px-3.5 py-2.5"
              :class="inviteRole === r.id ? 'border-brand-500 bg-brand-500/5' : 'border-line hover:border-line-strong'">
              <input v-model="inviteRole" type="radio" :value="r.id" class="mt-1 accent-brand-500" />
              <span>
                <span class="flex items-center gap-1.5 text-sm font-medium text-fg"><component :is="r.icon" class="h-3.5 w-3.5" />{{ r.label }}</span>
                <span class="block text-xs text-fg-muted">{{ r.description }}</span>
              </span>
            </label>
          </div>
        </div>
      </form>
      <template #footer>
        <template v-if="createdInvite">
          <button class="btn-secondary" @click="createdInvite = null">Invite another</button>
          <button class="btn-primary" @click="closeInvite">Done</button>
        </template>
        <template v-else>
          <button class="btn-secondary" @click="closeInvite">Cancel</button>
          <button class="btn-primary" type="submit" form="invite-form" :disabled="busy" data-testid="send-invite">
            <Loader2 v-if="busy" class="h-4 w-4 animate-spin" /> Create invitation
          </button>
        </template>
      </template>
    </AppModal>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import {
  Search, UserPlus, MoreHorizontal, PauseCircle, PlayCircle, KeyRound, Crown, UserMinus, Check, Minus, CheckCircle2, Loader2
} from 'lucide-vue-next'
import { api } from '../../api/client'
import { useAuthStore } from '../../stores/auth'
import { useToast } from '../../composables/useToast'
import { useConfirm } from '../../composables/useConfirm'
import { ROLES, roleMeta, PERMISSION_MATRIX } from '../../lib/roles'
import AppModal from '../ui/AppModal.vue'
import DropdownMenu from '../ui/DropdownMenu.vue'
import MenuItem from '../ui/MenuItem.vue'
import Tip from '../ui/Tip.vue'
import CopyButton from '../ui/CopyButton.vue'

const auth = useAuthStore()
const toast = useToast()
const confirm = useConfirm()

const members = ref([])
const invitations = ref([])
const teamName = ref('')
const currentUserId = ref('')
const loading = ref(true)
const filter = ref('')
const inviteOpen = ref(false)
const inviteEmail = ref('')
const inviteRole = ref('member')
const createdInvite = ref(null)
const busy = ref(false)
const inviteLinks = ref({}) // accept URLs are returned once, at creation

const canManage = computed(() => auth.can('members:manage'))
const isOwner = computed(() => auth.role === 'owner')
const assignableRoles = computed(() => ROLES.filter((r) => r.id !== 'owner' && (r.id !== 'admin' || isOwner.value)))

const visibleMembers = computed(() => {
  const q = filter.value.trim().toLowerCase()
  return q ? members.value.filter((m) => `${m.name} ${m.email}`.toLowerCase().includes(q)) : members.value
})

// Mirrors service.canManage: nobody manages the owner or themselves; only owners manage admins.
function canManageMember(m) {
  if (!canManage.value || m.user_id === currentUserId.value || m.role === 'owner') return false
  return isOwner.value || m.role !== 'admin'
}
const canChangeRole = (m) => canManageMember(m)

const initials = (m) => (m.name || m.email).split(/\s+/).map((p) => p[0]).slice(0, 2).join('').toUpperCase()
const formatDate = (d) => new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })

async function load() {
  loading.value = true
  try {
    const res = await api.getTeamMembers()
    members.value = res.members
    invitations.value = res.pending_invitations
    teamName.value = res.team_name
    currentUserId.value = res.current_user_id
  } catch (err) {
    toast.error(err.message)
  } finally {
    loading.value = false
  }
}

async function run(fn, success) {
  try {
    await fn()
    if (success) toast.success(success)
    await load()
    auth.refresh()
  } catch (err) {
    toast.error(err.message)
  }
}

async function changeRole(m, role) {
  if (role === m.role) return
  await run(() => api.updateMemberRole(m.user_id, role), `${m.name} is now ${roleMeta(role).label}`)
}

async function suspend(m) {
  const ok = await confirm({ title: `Suspend ${m.name}?`, message: 'They immediately lose access to this workspace, and their API keys for it stop working. You can reactivate them later.', confirmText: 'Suspend', danger: true })
  if (ok) await run(() => api.suspendMember(m.user_id), `${m.name} suspended`)
}
const reactivate = (m) => run(() => api.reactivateMember(m.user_id), `${m.name} reactivated`)

async function resetPassword(m) {
  const ok = await confirm({ title: 'Send password reset?', message: `${m.email} will receive an email with a link to choose a new password.`, confirmText: 'Send email' })
  if (ok) await run(() => api.resetMemberCredentials(m.user_id), `Reset email sent to ${m.email}`)
}

async function transfer(m) {
  const ok = await confirm({
    title: `Transfer ownership to ${m.name}?`,
    message: 'They become the owner, with billing and deletion rights. You become an admin. This cannot be undone without their help.',
    confirmText: 'Transfer ownership',
    danger: true,
    requireText: 'TRANSFER'
  })
  if (ok) await run(() => api.transferOwnership(m.user_id), `${m.name} is now the owner`)
}

async function remove(m) {
  const ok = await confirm({ title: `Remove ${m.name}?`, message: 'They lose access to this workspace. Their account and other workspaces are not affected.', confirmText: 'Remove', danger: true })
  if (ok) await run(() => api.removeTeamMember(m.user_id), `${m.name} removed`)
}

async function revoke(inv) {
  const ok = await confirm({ title: 'Revoke invitation?', message: `The invite link sent to ${inv.email} will stop working.`, confirmText: 'Revoke', danger: true })
  if (ok) await run(() => api.revokeInvitation(inv.id), 'Invitation revoked')
}

async function invite() {
  busy.value = true
  try {
    const res = await api.inviteTeamMember(inviteEmail.value, inviteRole.value)
    createdInvite.value = res.invitation
    inviteLinks.value[res.invitation.id] = res.invitation.accept_url
    inviteEmail.value = ''
    await load()
  } catch (err) {
    toast.error(err.message)
  } finally {
    busy.value = false
  }
}

function closeInvite() {
  inviteOpen.value = false
  createdInvite.value = null
  inviteRole.value = 'member'
}

onMounted(load)
defineExpose({ load })
</script>
