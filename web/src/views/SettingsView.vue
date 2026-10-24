<template>
  <div class="space-y-6 max-w-4xl">
    <div>
      <h1 class="text-xl font-bold text-white">Team & Account Settings</h1>
      <p class="text-xs text-gray-400 mt-0.5">Manage team members, WorkOS enterprise authentication, and developer API keys.</p>
    </div>

    <!-- WorkOS Auth Status Card (Requirement 9 & 11) -->
    <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-6 space-y-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-linksy-600/20 text-linksy-400 flex items-center justify-center font-bold">
            <ShieldCheck class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-white">WorkOS Authentication & SSO</h3>
            <span class="text-xs text-gray-400">Enterprise Single Sign-On (SAML / OIDC) and Team Directory</span>
          </div>
        </div>
        <span class="px-2.5 py-1 rounded-full bg-emerald-500/20 text-emerald-400 text-xs font-medium flex items-center gap-1.5">
          <CheckCircle2 class="w-3.5 h-3.5" />
          <span>Active</span>
        </span>
      </div>

      <div class="p-3 bg-gray-900/60 rounded-xl border border-gray-800 text-xs text-gray-400 leading-relaxed">
        WorkOS handles organization memberships, multi-factor authentication, and magic link logins securely across all Linksy team workspaces.
      </div>
    </div>

    <!-- Team Members & Permissions (Requirement 9 & 11) -->
    <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-6 space-y-4">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-sm font-bold text-white">Team Members</h3>
          <span class="text-xs text-gray-400">Manage who has access to generate and modify QR codes.</span>
        </div>
        <button class="bg-gray-700 hover:bg-gray-600 text-white px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5">
          <UserPlus class="w-3.5 h-3.5" />
          <span>Invite Member</span>
        </button>
      </div>

      <div class="divide-y divide-gray-750">
        <div v-for="m in teamMembers" :key="m.email" class="py-3 flex items-center justify-between text-xs">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 rounded-full bg-linksy-600/30 text-linksy-400 flex items-center justify-center font-bold">
              {{ m.name.charAt(0) }}
            </div>
            <div>
              <div class="font-semibold text-gray-200">{{ m.name }}</div>
              <div class="text-[11px] text-gray-500 font-mono">{{ m.email }}</div>
            </div>
          </div>

          <div class="flex items-center gap-3">
            <span class="px-2.5 py-0.5 rounded-full text-[11px] font-medium capitalize"
              :class="{
                'bg-amber-500/20 text-amber-300': m.role === 'owner',
                'bg-linksy-500/20 text-linksy-400': m.role === 'admin',
                'bg-gray-700 text-gray-300': m.role === 'member'
              }"
            >
              {{ m.role }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- API Keys (Requirement 10 & 12) -->
    <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-6 space-y-4">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-sm font-bold text-white">Personal API Keys</h3>
          <span class="text-xs text-gray-400">Authenticate public API requests programmatically.</span>
        </div>
        <button @click="generateKey" class="bg-linksy-600 hover:bg-linksy-500 text-white px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5">
          <Key class="w-3.5 h-3.5" />
          <span>Roll New Key</span>
        </button>
      </div>

      <div class="p-3 bg-gray-900/60 rounded-xl border border-gray-800 flex items-center justify-between font-mono text-xs">
        <span class="text-gray-300">{{ activeApiKey }}</span>
        <button @click="copyApiKey" class="text-linksy-400 hover:text-linksy-300 text-xs font-sans">
          Copy Token
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ShieldCheck, CheckCircle2, UserPlus, Key } from 'lucide-vue-next'

const teamMembers = ref([
  { name: 'Brandon Crowe', email: 'brandon@linksy.dev', role: 'owner' },
  { name: 'Marketing Lead', email: 'marketing@linksy.dev', role: 'admin' },
  { name: 'Designer Team', email: 'design@linksy.dev', role: 'member' }
])

const activeApiKey = ref('lnksy_live_9f82d1c0a8e7456b3e210')

function generateKey() {
  const rand = Math.random().toString(36).substring(2, 15)
  activeApiKey.value = `lnksy_live_${rand}`
  alert('New API token generated successfully!')
}

function copyApiKey() {
  navigator.clipboard.writeText(activeApiKey.value)
  alert('API Key copied to clipboard!')
}
</script>
