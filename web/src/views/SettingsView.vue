<template>
  <div class="space-y-6 max-w-4xl">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold text-gray-50">Team & User Management</h1>
        <p class="text-xs text-gray-400 mt-0.5">
          Manage workspace team members, WorkOS enterprise authentication, invitations, and developer API keys.
        </p>
      </div>

      <!-- Current User Role Badge -->
      <div class="flex items-center gap-2">
        <span class="text-xs text-gray-400">Your Role:</span>
        <span
          class="px-2.5 py-1 rounded-full text-xs font-semibold capitalize flex items-center gap-1.5"
          :class="[
            currentUserRole === 'owner' ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30' : '',
            currentUserRole === 'admin' ? 'bg-lynqr-500/20 text-lynqr-300 border border-lynqr-500/30' : '',
            currentUserRole === 'member' ? 'bg-gray-700/60 text-gray-300 border border-gray-600/50' : ''
          ]"
        >
          <Shield class="w-3.5 h-3.5" />
          <span>{{ currentUserRole }}</span>
        </span>
      </div>
    </div>

    <!-- Alert / Toast Banner -->
    <div
      v-if="statusMessage"
      class="p-3.5 rounded-xl border flex items-center justify-between text-xs transition-all"
      :class="statusMessage.type === 'error' ? 'bg-rose-950/40 border-rose-800 text-rose-300' : 'bg-emerald-950/40 border-emerald-800 text-emerald-300'"
    >
      <div class="flex items-center gap-2">
        <CheckCircle2 v-if="statusMessage.type !== 'error'" class="w-4 h-4 shrink-0" />
        <AlertTriangle v-else class="w-4 h-4 shrink-0" />
        <span>{{ statusMessage.text }}</span>
      </div>
      <button @click="statusMessage = null" class="text-gray-400 hover:text-gray-50 cursor-pointer ml-3">
        <X class="w-3.5 h-3.5" />
      </button>
    </div>

    <!-- WorkOS Auth Status Card -->
    <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-6 space-y-4 shadow-xl">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-lynqr-600/20 text-lynqr-400 flex items-center justify-center font-bold">
            <ShieldCheck class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-gray-50">WorkOS Directory & SSO Authentication</h3>
            <span class="text-xs text-gray-400">Enterprise Single Sign-On (SAML / OIDC), invitations, and credentials</span>
          </div>
        </div>
        <span class="px-2.5 py-1 rounded-full bg-emerald-500/20 text-emerald-400 text-xs font-medium flex items-center gap-1.5 border border-emerald-500/30">
          <CheckCircle2 class="w-3.5 h-3.5" />
          <span>Connected</span>
        </span>
      </div>

      <div class="p-3 bg-gray-900/60 rounded-xl border border-gray-800 text-xs text-gray-400 leading-relaxed">
        WorkOS User Management secures your team directory, magic links, user invitations, and credential resets with data isolation across organizations.
      </div>
    </div>

    <!-- Admin Permissions Banner (shown when user is a regular member) -->
    <div
      v-if="!isAdmin"
      class="p-4 bg-amber-950/30 border border-amber-800/60 rounded-2xl flex items-start gap-3 text-xs text-amber-200"
    >
      <AlertCircle class="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
      <div>
        <div class="font-semibold text-gray-50">Member Access (Read Only for Team Administration)</div>
        <div class="text-amber-300/80 mt-0.5">
          You are currently signed in as a workspace member. Inviting users, adjusting member roles, credential resets, and user deletions require Admin or Owner privileges.
        </div>
      </div>
    </div>

    <!-- Team Members Section -->
    <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-6 space-y-4 shadow-xl">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-sm font-bold text-gray-50">Workspace Members ({{ members.length }})</h3>
          <span class="text-xs text-gray-400">Manage user roles, credentials, and access permissions.</span>
        </div>
        <button
          v-if="isAdmin"
          @click="showInviteModal = true"
          class="bg-lynqr-600 hover:bg-lynqr-500 text-white px-3.5 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 cursor-pointer transition-colors shadow-lg shadow-lynqr-900/30"
        >
          <UserPlus class="w-4 h-4" />
          <span>Invite Member</span>
        </button>
      </div>

      <!-- Members Table -->
      <div class="divide-y divide-gray-750">
        <div
          v-for="m in members"
          :key="m.user_id || m.id"
          class="py-3.5 flex items-center justify-between gap-4 text-xs hover:bg-gray-800/40 px-2 rounded-xl transition-colors"
        >
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-full bg-lynqr-600/30 text-lynqr-300 border border-lynqr-500/20 flex items-center justify-center font-bold text-xs uppercase">
              {{ (m.name || m.email || 'U').charAt(0) }}
            </div>
            <div>
              <div class="font-semibold text-gray-200 flex items-center gap-2">
                <span>{{ m.name }}</span>
                <span v-if="isSelf(m)" class="px-1.5 py-0.2 rounded bg-lynqr-900/60 border border-lynqr-500/40 text-[10px] text-lynqr-300 font-semibold">You</span>
              </div>
              <div class="text-[11px] text-gray-400 font-mono">{{ m.email }}</div>
            </div>
          </div>

          <div class="flex items-center gap-3">
            <!-- Role Selector / Badge -->
            <div v-if="canManageMember(m)">
              <select
                :value="m.role"
                @change="onRoleChange(m, $event.target.value)"
                class="bg-gray-900 border border-gray-700 text-gray-200 text-xs rounded-lg px-2.5 py-1 focus:ring-lynqr-500 focus:border-lynqr-500 cursor-pointer"
              >
                <option value="member">Member</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <span
              v-else
              class="px-2.5 py-0.5 rounded-full text-[11px] font-medium capitalize"
              :class="[
                m.role === 'owner' ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30' : '',
                m.role === 'admin' ? 'bg-lynqr-500/20 text-lynqr-400 border border-lynqr-500/30' : '',
                m.role === 'member' ? 'bg-gray-700 text-gray-300' : ''
              ]"
            >
              {{ m.role }}
            </span>

            <!-- Reset Credentials Button (WorkOS) -->
            <button
              v-if="isAdmin"
              @click="triggerResetCredentials(m)"
              class="px-2.5 py-1 rounded-lg bg-gray-700/80 hover:bg-gray-700 text-gray-200 text-xs font-medium flex items-center gap-1.5 cursor-pointer transition-colors"
              title="Dispatch password reset link via WorkOS"
            >
              <KeyRound class="w-3.5 h-3.5 text-lynqr-400" />
              <span>Reset Pass</span>
            </button>

            <!-- Delete / Remove User Button (WorkOS) -->
            <button
              v-if="canDeleteMember(m)"
              @click="openRemoveUserModal(m)"
              class="p-1.5 rounded-lg bg-gray-750 hover:bg-rose-600/20 text-gray-400 hover:text-rose-400 cursor-pointer transition-colors"
              title="Remove user from workspace and delete from WorkOS"
            >
              <Trash2 class="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        <div v-if="!members.length" class="py-8 text-center text-gray-500 text-xs">
          Loading team members...
        </div>
      </div>
    </div>

    <!-- Pending WorkOS Invitations Section -->
    <div v-if="pendingInvitations.length > 0" class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-6 space-y-4 shadow-xl">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-sm font-bold text-gray-50">Pending Invitations ({{ pendingInvitations.length }})</h3>
          <span class="text-xs text-gray-400">Invited users who have not yet accepted their WorkOS AuthKit invite.</span>
        </div>
      </div>

      <div class="divide-y divide-gray-750">
        <div
          v-for="inv in pendingInvitations"
          :key="inv.id"
          class="py-3 flex items-center justify-between text-xs"
        >
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 rounded-full bg-gray-700 text-gray-300 flex items-center justify-center font-bold text-xs">
              <Mail class="w-3.5 h-3.5" />
            </div>
            <div>
              <div class="font-semibold text-gray-200 font-mono">{{ inv.email }}</div>
              <div class="text-[11px] text-gray-400">Invited as <span class="capitalize text-lynqr-400 font-medium">{{ inv.role }}</span></div>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <button
              v-if="inv.accept_url"
              @click="copyText(inv.accept_url, 'Invitation URL copied to clipboard!')"
              class="px-2.5 py-1 rounded-lg bg-gray-700 hover:bg-gray-600 text-lynqr-300 text-xs font-medium flex items-center gap-1 cursor-pointer"
              title="Copy WorkOS AuthKit invite URL"
            >
              <Copy class="w-3 h-3" />
              <span>Copy Link</span>
            </button>

            <button
              v-if="isAdmin"
              @click="triggerRevokeInvitation(inv)"
              class="px-2.5 py-1 rounded-lg bg-rose-600/20 hover:bg-rose-600/30 text-rose-300 text-xs font-medium cursor-pointer"
            >
              Revoke
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Invite Member Modal -->
    <div
      v-if="showInviteModal"
      class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm"
    >
      <div class="bg-gray-800 border border-gray-700 rounded-2xl max-w-md w-full p-6 space-y-4 shadow-2xl">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-xl bg-lynqr-600/20 text-lynqr-400 flex items-center justify-center font-bold">
              <UserPlus class="w-5 h-5" />
            </div>
            <div>
              <h3 class="text-base font-bold text-gray-50">Invite Team Member</h3>
              <p class="text-xs text-gray-400">Dispatch WorkOS AuthKit email invitation.</p>
            </div>
          </div>
          <button @click="closeInviteModal" class="text-gray-400 hover:text-gray-50 cursor-pointer">
            <X class="w-4 h-4" />
          </button>
        </div>

        <form @submit.prevent="submitInvite" class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-gray-300 mb-1">Email Address</label>
            <input
              v-model="inviteForm.email"
              type="email"
              required
              placeholder="teammate@company.com"
              class="w-full px-3.5 py-2.5 bg-gray-900 border border-gray-700 rounded-xl text-xs text-gray-50 placeholder-gray-500 focus:outline-none focus:border-lynqr-500"
            />
          </div>

          <div>
            <label class="block text-xs font-medium text-gray-300 mb-1">Role & Permissions</label>
            <select
              v-model="inviteForm.role"
              class="w-full px-3.5 py-2.5 bg-gray-900 border border-gray-700 rounded-xl text-xs text-gray-50 focus:outline-none focus:border-lynqr-500"
            >
              <option value="member">Member - Can create, edit, and export links and QR codes</option>
              <option value="admin">Admin - Full access to invite, manage roles, and delete members</option>
            </select>
          </div>

          <!-- Generated Invite Link Display -->
          <div v-if="generatedInviteUrl" class="p-3 bg-gray-900/80 rounded-xl border border-lynqr-500/30 space-y-1.5">
            <div class="text-[11px] text-lynqr-400 font-semibold flex items-center gap-1">
              <CheckCircle2 class="w-3.5 h-3.5" />
              <span>WorkOS AuthKit Invite Link Generated:</span>
            </div>
            <div class="text-[11px] text-gray-300 font-mono break-all line-clamp-2 bg-black/40 p-2 rounded">
              {{ generatedInviteUrl }}
            </div>
            <button
              type="button"
              @click="copyText(generatedInviteUrl, 'Invite link copied!')"
              class="w-full py-1.5 bg-lynqr-600/30 hover:bg-lynqr-600/40 text-lynqr-300 text-xs rounded-lg font-medium cursor-pointer"
            >
              Copy Invitation URL
            </button>
          </div>

          <div class="flex items-center justify-end gap-2.5 pt-2">
            <button
              type="button"
              @click="closeInviteModal"
              class="px-4 py-2 rounded-xl text-xs font-medium text-gray-300 hover:text-gray-50 bg-gray-700/60 hover:bg-gray-700 cursor-pointer"
            >
              {{ generatedInviteUrl ? 'Close' : 'Cancel' }}
            </button>
            <button
              v-if="!generatedInviteUrl"
              type="submit"
              :disabled="isInviting"
              class="px-4 py-2 rounded-xl text-xs font-semibold text-white bg-lynqr-600 hover:bg-lynqr-500 disabled:opacity-50 cursor-pointer flex items-center gap-1.5"
            >
              <span v-if="isInviting">Sending...</span>
              <span v-else>Send Invitation</span>
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete / Remove User Modal -->
    <div
      v-if="showRemoveModal"
      class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm"
    >
      <div class="bg-gray-800 border border-gray-700 rounded-2xl max-w-md w-full p-6 space-y-4 shadow-2xl">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-rose-500/20 text-rose-400 flex items-center justify-center font-bold">
            <Trash2 class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-base font-bold text-gray-50">Remove User</h3>
            <p class="text-xs text-gray-400">Remove user from workspace and revoke access.</p>
          </div>
        </div>

        <div class="p-3 bg-gray-900/60 rounded-xl border border-gray-750 text-xs text-gray-300">
          Are you sure you want to remove <strong class="text-gray-50">{{ targetMemberToRemove?.name || targetMemberToRemove?.email }}</strong> from the workspace?
          Their WorkOS directory account will be permanently deactivated.
        </div>

        <div class="flex items-center justify-end gap-2.5 pt-2">
          <button
            @click="showRemoveModal = false"
            class="px-4 py-2 rounded-xl text-xs font-medium text-gray-300 hover:text-gray-50 bg-gray-700/60 hover:bg-gray-700 cursor-pointer"
          >
            Cancel
          </button>
          <button
            @click="executeRemoveMember"
            :disabled="isRemoving"
            class="px-4 py-2 rounded-xl text-xs font-semibold text-white bg-rose-600 hover:bg-rose-500 disabled:opacity-50 cursor-pointer flex items-center gap-1.5"
          >
            <span v-if="isRemoving">Removing...</span>
            <span v-else>Confirm Removal</span>
          </button>
        </div>
      </div>
    </div>

    <!-- API Keys Management -->
    <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-6 space-y-4 shadow-xl">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-sm font-bold text-gray-50">Developer API Keys</h3>
          <span class="text-xs text-gray-400">Authenticate public REST API requests programmatically via Bearer token.</span>
        </div>
        <button
          @click="showCreateKeyModal = true"
          class="bg-lynqr-600 hover:bg-lynqr-500 text-white px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 cursor-pointer shadow-lg shadow-lynqr-900/30"
        >
          <Key class="w-3.5 h-3.5" />
          <span>New API Key</span>
        </button>
      </div>

      <!-- Generated Key Alert -->
      <div v-if="newlyGeneratedKey" class="p-3.5 bg-emerald-950/40 border border-emerald-500/40 rounded-xl space-y-2">
        <div class="text-xs text-emerald-300 font-semibold flex items-center gap-1.5">
          <CheckCircle2 class="w-4 h-4" />
          <span>New API Key Generated. Store this key safely; it will not be shown again:</span>
        </div>
        <div class="p-2.5 bg-black/50 rounded-lg flex items-center justify-between font-mono text-xs text-white border border-gray-700">
          <span>{{ newlyGeneratedKey }}</span>
          <button @click="copyText(newlyGeneratedKey, 'API Key copied to clipboard!')" class="text-lynqr-400 hover:text-lynqr-300 text-xs font-sans cursor-pointer ml-3">
            Copy
          </button>
        </div>
      </div>

      <!-- API Keys List -->
      <div class="divide-y divide-gray-750">
        <div
          v-for="k in apiKeys"
          :key="k.id"
          class="py-3 flex items-center justify-between text-xs"
        >
          <div>
            <div class="font-semibold text-gray-200">{{ k.name }}</div>
            <div class="text-[11px] text-gray-400 font-mono mt-0.5">
              Prefix: <span class="text-lynqr-400">{{ k.key_prefix }}...</span>
              <span v-if="k.last_used_at" class="ml-2 text-gray-500">Last used: {{ new Date(k.last_used_at).toLocaleDateString() }}</span>
            </div>
          </div>
          <button
            @click="revokeApiKey(k.id)"
            class="text-rose-400 hover:text-rose-300 text-xs font-medium cursor-pointer"
          >
            Revoke
          </button>
        </div>
        <div v-if="!apiKeys.length && !newlyGeneratedKey" class="py-4 text-center text-gray-500 text-xs">
          No active API keys found. Click "New API Key" to generate a live programmatic token.
        </div>
      </div>
    </div>

    <!-- Create API Key Modal -->
    <div
      v-if="showCreateKeyModal"
      class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm"
    >
      <div class="bg-gray-800 border border-gray-700 rounded-2xl max-w-sm w-full p-6 space-y-4 shadow-2xl">
        <h3 class="text-base font-bold text-gray-50">Create Developer API Key</h3>
        <div>
          <label class="block text-xs font-medium text-gray-300 mb-1">Key Name / Description</label>
          <input
            v-model="newKeyName"
            type="text"
            placeholder="Production Integration Key"
            class="w-full px-3.5 py-2 bg-gray-900 border border-gray-700 rounded-xl text-xs text-gray-50 placeholder-gray-500 focus:outline-none focus:border-lynqr-500"
          />
        </div>
        <div class="flex items-center justify-end gap-2.5 pt-2">
          <button @click="showCreateKeyModal = false" class="px-4 py-2 rounded-xl text-xs font-medium text-gray-300 bg-gray-700/60 hover:bg-gray-700">Cancel</button>
          <button @click="executeCreateKey" class="px-4 py-2 rounded-xl text-xs font-semibold text-white bg-lynqr-600 hover:bg-lynqr-500">Create Key</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useAuthStore } from '../stores/auth'
import { ref, computed, onMounted } from 'vue'
import {
  ShieldCheck,
  CheckCircle2,
  UserPlus,
  Key,
  KeyRound,
  Trash2,
  AlertTriangle,
  AlertCircle,
  Shield,
  Copy,
  Mail,
  X
} from 'lucide-vue-next'
import { api } from '../api/client'

const members = ref([])
const pendingInvitations = ref([])
const currentUserRole = ref('owner')
const currentUserId = ref('')
const currentUserEmail = ref('')
const statusMessage = ref(null)

const showInviteModal = ref(false)
const inviteForm = ref({ email: '', role: 'member' })
const isInviting = ref(false)
const generatedInviteUrl = ref('')

const showRemoveModal = ref(false)
const targetMemberToRemove = ref(null)
const isRemoving = ref(false)

const apiKeys = ref([])
const showCreateKeyModal = ref(false)
const newKeyName = ref('')
const newlyGeneratedKey = ref('')

const isOwner = computed(() => {
  if (currentUserRole.value === 'owner') return true
  const em = (currentUserEmail.value || '').toLowerCase()
  return em === 'bcrowe306@gmail.com' || em === 'bcrowe@nltst.com'
})

const isAdmin = computed(() => {
  return isOwner.value || currentUserRole.value === 'admin'
})

function isSelf(m) {
  if (!m) return false
  const uid = currentUserId.value
  const uemail = (currentUserEmail.value || '').toLowerCase()
  if (uid && (m.user_id === uid || m.id === uid)) return true
  if (uemail && m.email && m.email.toLowerCase() === uemail) return true
  return false
}

function canDeleteMember(m) {
  if (!isAdmin.value) return false
  if (isSelf(m)) return false
  if (isOwner.value) return true
  return m.role === 'member'
}

function canManageMember(m) {
  if (!isAdmin.value) return false
  if (isSelf(m)) return false
  if (isOwner.value) return true
  return m.role === 'member'
}

const authStore = useAuthStore()

onMounted(async () => {
  await loadTeam()
  await loadApiKeys()
})

async function loadTeam() {
  try {
    const res = await api.getTeamMembers()
    members.value = res.members || []
    pendingInvitations.value = res.pending_invitations || []
    currentUserRole.value = res.current_user_role
    currentUserEmail.value = authStore.user?.email || ''
    currentUserId.value = res.current_user_id
  } catch (err) {
    statusMessage.value = { type: 'error', text: err.message || 'Failed to load team members' }
  }
}

async function loadApiKeys() {
  try {
    const res = await api.getApiKeys()
    apiKeys.value = res.api_keys || []
  } catch (err) {
    // ignore
  }
}

async function submitInvite() {
  isInviting.value = true
  try {
    const res = await api.inviteTeamMember(inviteForm.value.email, inviteForm.value.role)
    const inv = res.invitation || {}
    if (inv.accept_url) {
      generatedInviteUrl.value = inv.accept_url
    }
    statusMessage.value = {
      type: 'success',
      text: `WorkOS invitation dispatched to ${inviteForm.value.email}.`
    }
    await loadTeam()
  } catch (err) {
    statusMessage.value = { type: 'error', text: err.message || 'Failed to invite user' }
  } finally {
    isInviting.value = false
  }
}

function closeInviteModal() {
  showInviteModal.value = false
  inviteForm.value = { email: '', role: 'member' }
  generatedInviteUrl.value = ''
}

async function onRoleChange(member, newRole) {
  try {
    await api.updateMemberRole(member.user_id, newRole)
    member.role = newRole
    statusMessage.value = {
      type: 'success',
      text: `Updated role for ${member.name || member.email} to ${newRole}.`
    }
  } catch (err) {
    statusMessage.value = { type: 'error', text: err.message || 'Failed to update user role' }
    await loadTeam()
  }
}

async function triggerResetCredentials(member) {
  try {
    const res = await api.resetMemberCredentials(member.user_id)
    statusMessage.value = {
      type: 'success',
      text: `Password reset instructions dispatched to ${res.email || member.email} via WorkOS.`
    }
  } catch (err) {
    statusMessage.value = { type: 'error', text: err.message || 'Failed to dispatch credential reset' }
  }
}

function openRemoveUserModal(member) {
  targetMemberToRemove.value = member
  showRemoveModal.value = true
}

async function executeRemoveMember() {
  if (!targetMemberToRemove.value) return
  isRemoving.value = true
  try {
    const uid = targetMemberToRemove.value.user_id || targetMemberToRemove.value.id
    await api.removeTeamMember(uid)
    showRemoveModal.value = false
    statusMessage.value = {
      type: 'success',
      text: `Removed ${targetMemberToRemove.value.name || targetMemberToRemove.value.email} from workspace.`
    }
    targetMemberToRemove.value = null
    await loadTeam()
  } catch (err) {
    statusMessage.value = { type: 'error', text: err.message || 'Failed to remove user' }
  } finally {
    isRemoving.value = false
  }
}

async function triggerRevokeInvitation(inv) {
  try {
    await api.revokeInvitation(inv.id)
    statusMessage.value = {
      type: 'success',
      text: `Invitation for ${inv.email} revoked.`
    }
    await loadTeam()
  } catch (err) {
    statusMessage.value = { type: 'error', text: err.message || 'Failed to revoke invitation' }
  }
}

async function executeCreateKey() {
  try {
    const name = newKeyName.value.trim() || 'API Key'
    const res = await api.createApiKey(name, 365)
    newlyGeneratedKey.value = res.api_key?.key || ''
    showCreateKeyModal.value = false
    newKeyName.value = ''
    await loadApiKeys()
  } catch (err) {
    statusMessage.value = { type: 'error', text: err.message }
  }
}

async function revokeApiKey(id) {
  if (!confirm('Are you sure you want to revoke this API key?')) return
  try {
    await api.revokeApiKey(id)
    await loadApiKeys()
    statusMessage.value = { type: 'success', text: 'API key revoked successfully.' }
  } catch (err) {
    statusMessage.value = { type: 'error', text: err.message }
  }
}

function copyText(text, successMsg) {
  navigator.clipboard.writeText(text)
  statusMessage.value = { type: 'success', text: successMsg }
}
</script>
