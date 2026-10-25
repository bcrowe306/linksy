<template>
  <div class="space-y-8 pb-12">
    <!-- Profile Header Banner -->
    <div class="bg-gray-900 border border-gray-800 rounded-2xl p-6 sm:p-8 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-6 relative overflow-hidden">
      <div class="flex items-center gap-5 z-10">
        <!-- Large Avatar -->
        <div class="relative group">
          <div v-if="profile.avatar_url" class="w-20 h-20 rounded-2xl overflow-hidden border-2 border-linksy-500/40 shadow-xl shadow-linksy-500/10">
            <img :src="profile.avatar_url" alt="Avatar" class="w-full h-full object-cover" />
          </div>
          <div v-else class="w-20 h-20 rounded-2xl bg-gradient-to-tr from-linksy-600 to-indigo-500 flex items-center justify-center font-bold text-2xl text-white border-2 border-linksy-500/30 shadow-xl shadow-linksy-500/10">
            {{ userInitials }}
          </div>
          <div class="absolute -bottom-1 -right-1 w-6 h-6 rounded-lg bg-emerald-500 border-2 border-gray-900 flex items-center justify-center" title="Active Account">
            <Check class="w-3.5 h-3.5 text-white stroke-[3]" />
          </div>
        </div>

        <div>
          <div class="flex items-center gap-3 mb-1">
            <h1 class="text-2xl font-extrabold text-white tracking-tight">{{ profile.name || 'Account User' }}</h1>
            <span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-linksy-500/15 text-linksy-400 border border-linksy-500/20 capitalize font-mono">
              {{ profile.role || 'Owner' }}
            </span>
          </div>
          <p class="text-sm text-gray-400 flex items-center gap-2">
            <span>{{ profile.email }}</span>
            <span class="inline-block w-1 h-1 rounded-full bg-gray-600"></span>
            <span class="text-xs text-gray-500">ID: {{ profile.id }}</span>
          </p>
        </div>
      </div>

      <!-- Quick Actions -->
      <div class="flex items-center gap-3 z-10">
        <button
          @click="authStore.logout"
          class="px-4 py-2 border border-rose-500/30 text-rose-400 hover:bg-rose-500/10 rounded-xl text-xs font-semibold flex items-center gap-2 transition-colors"
        >
          <LogOut class="w-3.5 h-3.5" />
          <span>Sign Out</span>
        </button>
      </div>

      <!-- Background Glow Effect -->
      <div class="absolute -right-16 -top-16 w-64 h-64 bg-linksy-500/10 rounded-full blur-3xl pointer-events-none"></div>
    </div>

    <!-- Navigation Tabs -->
    <div class="border-b border-gray-800 flex items-center gap-2 overflow-x-auto">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        @click="activeTab = tab.id"
        class="px-4 py-3 text-sm font-semibold border-b-2 flex items-center gap-2 transition-all whitespace-nowrap"
        :class="[
          activeTab === tab.id
            ? 'border-linksy-500 text-linksy-400 font-bold'
            : 'border-transparent text-gray-400 hover:text-gray-200 hover:border-gray-700'
        ]"
      >
        <component :is="tab.icon" class="w-4 h-4" />
        <span>{{ tab.label }}</span>
        <span v-if="tab.badge" class="ml-1.5 px-2 py-0.5 rounded-full text-[10px] bg-gray-800 text-gray-300">
          {{ tab.badge }}
        </span>
      </button>
    </div>

    <!-- Feedback Alerts -->
    <div v-if="feedbackMessage" class="p-4 rounded-xl flex items-center gap-3 text-sm" :class="feedbackIsError ? 'bg-rose-500/10 border border-rose-500/30 text-rose-400' : 'bg-emerald-500/10 border border-emerald-500/30 text-emerald-400'">
      <AlertCircle v-if="feedbackIsError" class="w-5 h-5 shrink-0" />
      <CheckCircle2 v-else class="w-5 h-5 shrink-0" />
      <span>{{ feedbackMessage }}</span>
      <button @click="feedbackMessage = ''" class="ml-auto text-xs opacity-70 hover:opacity-100">Dismiss</button>
    </div>

    <!-- TAB 1: General Profile -->
    <div v-if="activeTab === 'profile'" class="grid grid-cols-1 lg:grid-cols-3 gap-8">
      <div class="lg:col-span-2 bg-gray-900 border border-gray-800 rounded-2xl p-6 sm:p-8 space-y-6">
        <div>
          <h2 class="text-lg font-bold text-white mb-1">Personal Details</h2>
          <p class="text-xs text-gray-400">Update your public name and display avatar across Linksy reports.</p>
        </div>

        <form @submit.prevent="saveProfile" class="space-y-4">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label class="block text-xs font-medium text-gray-300 mb-1">First Name</label>
              <input
                v-model="editForm.first_name"
                type="text"
                placeholder="Jane"
                class="w-full bg-gray-800 border border-gray-700 rounded-xl px-3.5 py-2.5 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-linksy-500"
              />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-300 mb-1">Last Name</label>
              <input
                v-model="editForm.last_name"
                type="text"
                placeholder="Doe"
                class="w-full bg-gray-800 border border-gray-700 rounded-xl px-3.5 py-2.5 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-linksy-500"
              />
            </div>
          </div>

          <div>
            <label class="block text-xs font-medium text-gray-300 mb-1">Email Address</label>
            <div class="relative">
              <input
                :value="profile.email"
                disabled
                class="w-full bg-gray-800/50 border border-gray-700/60 rounded-xl px-3.5 py-2.5 text-sm text-gray-400 cursor-not-allowed"
              />
              <span class="absolute right-3 top-1/2 -translate-y-1/2 text-[11px] font-semibold text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-md border border-emerald-500/20">
                Verified
              </span>
            </div>
            <p class="mt-1 text-[11px] text-gray-500">Email addresses are verified through WorkOS Authentication.</p>
          </div>

          <div>
            <label class="block text-xs font-medium text-gray-300 mb-1">Avatar Image URL</label>
            <input
              v-model="editForm.avatar_url"
              type="url"
              placeholder="https://example.com/photo.jpg"
              class="w-full bg-gray-800 border border-gray-700 rounded-xl px-3.5 py-2.5 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-linksy-500"
            />
          </div>

          <div class="pt-2">
            <button
              type="submit"
              :disabled="isSavingProfile"
              class="px-5 py-2.5 bg-linksy-600 hover:bg-linksy-500 text-white rounded-xl text-sm font-semibold flex items-center gap-2 shadow-lg shadow-linksy-600/20 transition-all disabled:opacity-50"
            >
              <Loader2 v-if="isSavingProfile" class="w-4 h-4 animate-spin" />
              <span>Save Changes</span>
            </button>
          </div>
        </form>
      </div>

      <!-- Side Profile Info Card -->
      <div class="space-y-6">
        <div class="bg-gray-900 border border-gray-800 rounded-2xl p-6 space-y-4">
          <h3 class="text-sm font-bold text-white flex items-center gap-2">
            <ShieldCheck class="w-4 h-4 text-linksy-400" />
            <span>Workspace & Role</span>
          </h3>
          <div class="p-3.5 rounded-xl bg-gray-800/60 border border-gray-700/50 space-y-2">
            <div class="flex items-center justify-between text-xs">
              <span class="text-gray-400">Team</span>
              <span class="font-semibold text-gray-200">Main Workspace</span>
            </div>
            <div class="flex items-center justify-between text-xs">
              <span class="text-gray-400">Role</span>
              <span class="font-mono text-linksy-400 uppercase text-[11px] font-bold">{{ profile.role || 'owner' }}</span>
            </div>
            <div class="flex items-center justify-between text-xs">
              <span class="text-gray-400">Environment</span>
              <span class="text-gray-200 font-mono text-[11px]">Production</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- TAB 2: Authentication & Password Reset -->
    <div v-if="activeTab === 'security'" class="grid grid-cols-1 lg:grid-cols-2 gap-8">
      <!-- Self-Service Password Reset via WorkOS -->
      <div class="bg-gray-900 border border-gray-800 rounded-2xl p-6 sm:p-8 space-y-5">
        <div>
          <h2 class="text-lg font-bold text-white mb-1 flex items-center gap-2">
            <Mail class="w-5 h-5 text-linksy-400" />
            <span>Password Reset (Email)</span>
          </h2>
          <p class="text-xs text-gray-400 leading-relaxed">
            Trigger a secure WorkOS one-time password reset link sent directly to your registered email address.
          </p>
        </div>

        <div class="p-4 rounded-xl bg-gray-800/60 border border-gray-700/60 space-y-3">
          <div class="text-xs text-gray-300">
            Recipient: <span class="font-semibold text-white font-mono">{{ profile.email }}</span>
          </div>
          <button
            @click="sendPasswordResetEmail"
            :disabled="isSendingReset"
            class="w-full py-2.5 px-4 bg-gray-800 hover:bg-gray-700 border border-gray-600 rounded-xl text-sm font-semibold text-white flex items-center justify-center gap-2 transition-all disabled:opacity-50"
          >
            <Loader2 v-if="isSendingReset" class="w-4 h-4 animate-spin" />
            <Send v-else class="w-4 h-4 text-linksy-400" />
            <span>Send Reset Instructions Email</span>
          </button>
        </div>

        <!-- Generated Reset URL Display if returned -->
        <div v-if="passwordResetUrl" class="p-4 rounded-xl bg-linksy-950/60 border border-linksy-500/30 space-y-2">
          <div class="text-xs font-semibold text-linksy-400 flex items-center gap-1.5">
            <ExternalLink class="w-3.5 h-3.5" />
            <span>Direct One-Time Reset Link</span>
          </div>
          <p class="text-[11px] text-gray-400">You can also use this secure reset link directly:</p>
          <div class="flex items-center gap-2">
            <input
              type="text"
              readonly
              :value="passwordResetUrl"
              class="w-full bg-gray-900 border border-gray-700 rounded-lg px-2.5 py-1.5 text-xs text-gray-300 font-mono select-all"
            />
            <a
              :href="passwordResetUrl"
              target="_blank"
              class="px-3 py-1.5 bg-linksy-600 hover:bg-linksy-500 text-white rounded-lg text-xs font-semibold whitespace-nowrap"
            >
              Open Link
            </a>
          </div>
        </div>
      </div>

      <!-- Direct Password Update Form -->
      <div class="bg-gray-900 border border-gray-800 rounded-2xl p-6 sm:p-8 space-y-5">
        <div>
          <h2 class="text-lg font-bold text-white mb-1 flex items-center gap-2">
            <KeyRound class="w-5 h-5 text-indigo-400" />
            <span>Set New Password</span>
          </h2>
          <p class="text-xs text-gray-400 leading-relaxed">
            Directly update your account password across all Linksy client sessions.
          </p>
        </div>

        <form @submit.prevent="changePassword" class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-gray-300 mb-1">New Password</label>
            <input
              v-model="newPassword"
              type="password"
              required
              placeholder="At least 8 characters (Letters, numbers, symbols)"
              class="w-full bg-gray-800 border border-gray-700 rounded-xl px-3.5 py-2.5 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-linksy-500"
            />
          </div>

          <div>
            <label class="block text-xs font-medium text-gray-300 mb-1">Confirm New Password</label>
            <input
              v-model="confirmPassword"
              type="password"
              required
              placeholder="Re-enter new password"
              class="w-full bg-gray-800 border border-gray-700 rounded-xl px-3.5 py-2.5 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-linksy-500"
            />
          </div>

          <div class="pt-2">
            <button
              type="submit"
              :disabled="isUpdatingPassword"
              class="w-full py-2.5 px-4 bg-gradient-to-r from-linksy-600 to-indigo-600 hover:from-linksy-500 hover:to-indigo-500 text-white rounded-xl text-sm font-semibold flex items-center justify-center gap-2 shadow-lg shadow-linksy-600/20 transition-all disabled:opacity-50"
            >
              <Loader2 v-if="isUpdatingPassword" class="w-4 h-4 animate-spin" />
              <span>Update Password</span>
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- TAB 3: API Keys Management -->
    <div v-if="activeTab === 'api-keys'" class="space-y-6">
      <!-- Section Header -->
      <div class="bg-gray-900 border border-gray-800 rounded-2xl p-6 sm:p-8 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <h2 class="text-lg font-bold text-white mb-1 flex items-center gap-2">
            <Terminal class="w-5 h-5 text-linksy-400" />
            <span>Developer API Keys</span>
          </h2>
          <p class="text-xs text-gray-400">
            Authenticate programmatic requests to Linksy REST APIs via <code class="text-linksy-400 bg-gray-800 px-1.5 py-0.5 rounded font-mono">Authorization: Bearer lk_live_...</code>
          </p>
        </div>

        <button
          @click="showCreateKeyModal = true"
          class="px-4 py-2.5 bg-linksy-600 hover:bg-linksy-500 text-white rounded-xl text-xs font-semibold flex items-center gap-2 shadow-lg shadow-linksy-600/20 transition-all active:scale-95"
        >
          <Plus class="w-4 h-4 stroke-[2.5]" />
          <span>Generate New API Key</span>
        </button>
      </div>

      <!-- API Keys Table / List -->
      <div class="bg-gray-900 border border-gray-800 rounded-2xl overflow-hidden shadow-xl">
        <div v-if="isLoadingKeys" class="p-12 text-center text-gray-500 flex flex-col items-center justify-center gap-3">
          <Loader2 class="w-6 h-6 animate-spin text-linksy-400" />
          <span class="text-xs">Loading API keys...</span>
        </div>

        <div v-else-if="apiKeys.length === 0" class="p-12 text-center text-gray-500 space-y-3">
          <div class="w-12 h-12 rounded-2xl bg-gray-800/80 mx-auto flex items-center justify-center text-gray-400">
            <KeyRound class="w-6 h-6" />
          </div>
          <p class="text-sm font-semibold text-gray-300">No active API keys yet</p>
          <p class="text-xs text-gray-500 max-w-sm mx-auto">Generate an API key to integrate Linksy QR code generation, short links, and analytics into your external systems.</p>
        </div>

        <div v-else class="divide-y divide-gray-800">
          <div
            v-for="k in apiKeys"
            :key="k.id"
            class="p-5 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 hover:bg-gray-800/30 transition-colors"
          >
            <div class="space-y-1">
              <div class="flex items-center gap-2.5">
                <span class="text-sm font-bold text-gray-200">{{ k.name }}</span>
                <span class="px-2 py-0.5 rounded bg-gray-800 text-[11px] font-mono text-gray-400 border border-gray-700">
                  {{ k.key_prefix }}
                </span>
              </div>
              <div class="flex items-center gap-4 text-xs text-gray-500 font-mono">
                <span>Created: {{ formatDate(k.created_at) }}</span>
                <span v-if="k.last_used_at">Last used: {{ formatDate(k.last_used_at) }}</span>
                <span v-else class="text-gray-600">Never used</span>
              </div>
            </div>

            <button
              @click="revokeApiKey(k.id)"
              class="px-3 py-1.5 rounded-lg border border-rose-500/30 text-rose-400 hover:bg-rose-500/10 text-xs font-semibold flex items-center gap-1.5 transition-colors"
            >
              <Trash2 class="w-3.5 h-3.5" />
              <span>Revoke Key</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- MODAL: Generate New API Key -->
    <div v-if="showCreateKeyModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-950/80 backdrop-blur-sm">
      <div class="bg-gray-900 border border-gray-800 rounded-2xl max-w-md w-full p-6 shadow-2xl space-y-5 animate-in fade-in zoom-in-95">
        <div class="flex items-center justify-between">
          <h3 class="text-base font-bold text-white flex items-center gap-2">
            <KeyRound class="w-4 h-4 text-linksy-400" />
            <span>Generate New API Key</span>
          </h3>
          <button @click="showCreateKeyModal = false" class="text-gray-400 hover:text-white">
            <X class="w-4 h-4" />
          </button>
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-gray-300 mb-1">Key Label / Name</label>
            <input
              v-model="newKeyName"
              type="text"
              placeholder="e.g. Production Backend Bot"
              class="w-full bg-gray-800 border border-gray-700 rounded-xl px-3.5 py-2.5 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-linksy-500"
            />
          </div>

          <div>
            <label class="block text-xs font-medium text-gray-300 mb-1">Expiration</label>
            <select
              v-model="newKeyExpires"
              class="w-full bg-gray-800 border border-gray-700 rounded-xl px-3.5 py-2.5 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-linksy-500"
            >
              <option :value="0">Never expires (Default)</option>
              <option :value="30">30 Days</option>
              <option :value="90">90 Days</option>
              <option :value="365">1 Year</option>
            </select>
          </div>
        </div>

        <div class="flex items-center justify-end gap-3 pt-2">
          <button
            @click="showCreateKeyModal = false"
            class="px-4 py-2 rounded-xl text-xs font-semibold text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
          >
            Cancel
          </button>
          <button
            @click="createApiKey"
            :disabled="isCreatingKey"
            class="px-4 py-2 bg-linksy-600 hover:bg-linksy-500 text-white rounded-xl text-xs font-semibold flex items-center gap-2 shadow-lg shadow-linksy-600/20 transition-all disabled:opacity-50"
          >
            <Loader2 v-if="isCreatingKey" class="w-3.5 h-3.5 animate-spin" />
            <span>Generate Key</span>
          </button>
        </div>
      </div>
    </div>

    <!-- MODAL: One-time Secret Display -->
    <div v-if="newlyCreatedKeySecret" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-950/80 backdrop-blur-sm">
      <div class="bg-gray-900 border border-emerald-500/30 rounded-2xl max-w-lg w-full p-6 shadow-2xl space-y-5 animate-in fade-in zoom-in-95">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
            <Check class="w-5 h-5 stroke-[2.5]" />
          </div>
          <div>
            <h3 class="text-base font-bold text-white">API Key Created Successfully</h3>
            <p class="text-xs text-gray-400">Copy this secret key now. It will not be shown again.</p>
          </div>
        </div>

        <div class="p-3.5 rounded-xl bg-gray-950 border border-gray-800 space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-[11px] font-mono text-gray-500 uppercase tracking-wider">Bearer Token</span>
            <button
              @click="copyKeySecret"
              class="text-xs font-semibold text-linksy-400 hover:text-linksy-300 flex items-center gap-1.5 transition-colors"
            >
              <Copy class="w-3.5 h-3.5" />
              <span>{{ hasCopied ? 'Copied!' : 'Copy Key' }}</span>
            </button>
          </div>
          <div class="font-mono text-xs text-emerald-400 break-all select-all bg-gray-900/60 p-2.5 rounded-lg border border-gray-800">
            {{ newlyCreatedKeySecret }}
          </div>
        </div>

        <div class="flex items-center justify-end">
          <button
            @click="newlyCreatedKeySecret = ''"
            class="px-5 py-2 bg-linksy-600 hover:bg-linksy-500 text-white rounded-xl text-xs font-semibold transition-all shadow-md"
          >
            I Have Saved This Key
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import {
  UserCircle,
  KeyRound,
  ShieldCheck,
  Check,
  CheckCircle2,
  AlertCircle,
  Mail,
  Send,
  ExternalLink,
  Terminal,
  Plus,
  Trash2,
  Copy,
  X,
  LogOut,
  Loader2
} from 'lucide-vue-next'
import { useAuthStore } from '../stores/auth'

const authStore = useAuthStore()

const activeTab = ref('profile')
const feedbackMessage = ref('')
const feedbackIsError = ref(false)

const profile = ref({
  id: '',
  email: '',
  name: '',
  first_name: '',
  last_name: '',
  avatar_url: '',
  role: 'owner'
})

const editForm = ref({
  first_name: '',
  last_name: '',
  avatar_url: ''
})

const isSavingProfile = ref(false)
const isSendingReset = ref(false)
const isUpdatingPassword = ref(false)
const passwordResetUrl = ref('')
const newPassword = ref('')
const confirmPassword = ref('')

// API Keys State
const apiKeys = ref([])
const isLoadingKeys = ref(false)
const showCreateKeyModal = ref(false)
const isCreatingKey = ref(false)
const newKeyName = ref('')
const newKeyExpires = ref(0)
const newlyCreatedKeySecret = ref('')
const hasCopied = ref(false)

const tabs = computed(() => [
  { id: 'profile', label: 'General Profile', icon: UserCircle },
  { id: 'security', label: 'Password & Security', icon: KeyRound },
  { id: 'api-keys', label: 'API Keys', icon: Terminal, badge: apiKeys.value.length || null },
])

const userInitials = computed(() => {
  if (profile.value.name) {
    const parts = profile.value.name.trim().split(' ')
    if (parts.length >= 2) return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
    return profile.value.name.substring(0, 2).toUpperCase()
  }
  return 'LK'
})

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

async function fetchProfile() {
  try {
    const res = await fetch('/api/v1/user/profile')
    if (res.ok) {
      const data = await res.json()
      if (data.user) {
        profile.value = data.user
        editForm.value.first_name = data.user.first_name || ''
        editForm.value.last_name = data.user.last_name || ''
        editForm.value.avatar_url = data.user.avatar_url || ''
      }
    }
  } catch (err) {
    console.error('Failed to load profile:', err)
  }
}

async function saveProfile() {
  isSavingProfile.value = true
  feedbackMessage.value = ''
  try {
    const res = await fetch('/api/v1/user/profile', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(editForm.value)
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'Failed to update profile')
    
    feedbackIsError.value = false
    feedbackMessage.value = 'Profile updated successfully'
    await fetchProfile()
  } catch (err) {
    feedbackIsError.value = true
    feedbackMessage.value = err.message
  } finally {
    isSavingProfile.value = false
  }
}

async function sendPasswordResetEmail() {
  isSendingReset.value = true
  feedbackMessage.value = ''
  passwordResetUrl.value = ''
  try {
    const res = await fetch('/api/v1/auth/password-reset', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: profile.value.email })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'Failed to send password reset')

    feedbackIsError.value = false
    feedbackMessage.value = 'Password reset instructions have been dispatched'
    if (data.password_reset_url) {
      passwordResetUrl.value = data.password_reset_url
    }
  } catch (err) {
    feedbackIsError.value = true
    feedbackMessage.value = err.message
  } finally {
    isSendingReset.value = false
  }
}

async function changePassword() {
  if (newPassword.value !== confirmPassword.value) {
    feedbackIsError.value = true
    feedbackMessage.value = 'Passwords do not match'
    return
  }

  if (newPassword.value.length < 8) {
    feedbackIsError.value = true
    feedbackMessage.value = 'Password must be at least 8 characters long'
    return
  }

  isUpdatingPassword.value = true
  feedbackMessage.value = ''
  try {
    const res = await fetch('/api/v1/user/change-password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ new_password: newPassword.value })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'Failed to update password')

    feedbackIsError.value = false
    feedbackMessage.value = 'Password updated successfully across all sessions'
    newPassword.value = ''
    confirmPassword.value = ''
  } catch (err) {
    feedbackIsError.value = true
    feedbackMessage.value = err.message
  } finally {
    isUpdatingPassword.value = false
  }
}

async function fetchApiKeys() {
  isLoadingKeys.value = true
  try {
    const res = await fetch('/api/v1/api-keys')
    if (res.ok) {
      const data = await res.json()
      apiKeys.value = data.api_keys || []
    }
  } catch (err) {
    console.error('Failed to load API keys:', err)
  } finally {
    isLoadingKeys.value = false
  }
}

async function createApiKey() {
  isCreatingKey.value = true
  try {
    const res = await fetch('/api/v1/api-keys', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: newKeyName.value || 'API Key',
        expires_in_days: Number(newKeyExpires.value)
      })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'Failed to create API key')

    showCreateKeyModal.value = false
    newKeyName.value = ''
    newKeyExpires.value = 0
    if (data.api_key && data.api_key.key) {
      newlyCreatedKeySecret.value = data.api_key.key
      hasCopied.value = false
    }
    await fetchApiKeys()
  } catch (err) {
    feedbackIsError.value = true
    feedbackMessage.value = err.message
  } finally {
    isCreatingKey.value = false
  }
}

async function revokeApiKey(id) {
  if (!confirm('Are you sure you want to revoke this API key? Systems using this key will immediately lose access.')) {
    return
  }

  try {
    const res = await fetch(`/api/v1/api-keys/${id}`, { method: 'DELETE' })
    if (!res.ok) {
      const data = await res.json()
      throw new Error(data.error || 'Failed to revoke API key')
    }
    await fetchApiKeys()
    feedbackIsError.value = false
    feedbackMessage.value = 'API key revoked'
  } catch (err) {
    feedbackIsError.value = true
    feedbackMessage.value = err.message
  }
}

function copyKeySecret() {
  if (!newlyCreatedKeySecret.value) return
  navigator.clipboard.writeText(newlyCreatedKeySecret.value)
  hasCopied.value = true
  setTimeout(() => {
    hasCopied.value = false
  }, 2500)
}

onMounted(() => {
  fetchProfile()
  fetchApiKeys()
})
</script>
