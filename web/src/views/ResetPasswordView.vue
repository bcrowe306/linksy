<template>
  <div class="min-h-screen flex items-center justify-center p-6">
    <div class="w-full max-w-md bg-gray-900/80 border border-gray-800 rounded-2xl p-8 shadow-2xl">
      <router-link to="/login" class="inline-flex items-center gap-1.5 text-xs text-gray-400 hover:text-gray-200 mb-6">
        <ArrowLeft class="w-3.5 h-3.5" /> Back to sign in
      </router-link>

      <!-- Step 2: choose a new password (arrived from the emailed link) -->
      <template v-if="token">
        <h1 class="text-xl font-bold text-white mb-1">Choose a new password</h1>
        <p class="text-xs text-gray-400 mb-6">Your new password must be at least 8 characters.</p>
        <div v-if="done" class="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-300 text-sm">
          Password updated. <router-link to="/login" class="underline font-semibold">Sign in</router-link> with your new password.
        </div>
        <form v-else @submit.prevent="confirm" class="space-y-4">
          <input v-model="password" type="password" required minlength="8" placeholder="New password" autocomplete="new-password"
            class="w-full bg-gray-800 border border-gray-700 rounded-xl px-3.5 py-2 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-lynqr-500" />
          <input v-model="confirmPassword" type="password" required placeholder="Confirm new password" autocomplete="new-password"
            class="w-full bg-gray-800 border border-gray-700 rounded-xl px-3.5 py-2 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-lynqr-500" />
          <p v-if="error" class="text-xs text-rose-400">{{ error }}</p>
          <button type="submit" :disabled="busy" class="w-full py-2.5 bg-lynqr-600 hover:bg-lynqr-500 text-white rounded-xl text-sm font-semibold disabled:opacity-50">
            {{ busy ? 'Saving…' : 'Update password' }}
          </button>
        </form>
      </template>

      <!-- Step 1: request a reset email -->
      <template v-else>
        <h1 class="text-xl font-bold text-white mb-1">Reset your password</h1>
        <p class="text-xs text-gray-400 mb-6">Enter your account email and we'll send you a reset link.</p>
        <div v-if="done" class="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-300 text-sm">
          {{ message }}
        </div>
        <form v-else @submit.prevent="requestReset" class="space-y-4">
          <input v-model="email" type="email" required placeholder="you@company.com" autocomplete="email"
            class="w-full bg-gray-800 border border-gray-700 rounded-xl px-3.5 py-2 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-lynqr-500" />
          <p v-if="error" class="text-xs text-rose-400">{{ error }}</p>
          <button type="submit" :disabled="busy" class="w-full py-2.5 bg-lynqr-600 hover:bg-lynqr-500 text-white rounded-xl text-sm font-semibold disabled:opacity-50">
            {{ busy ? 'Sending…' : 'Send reset link' }}
          </button>
        </form>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowLeft } from 'lucide-vue-next'
import { api } from '../api/client'

const route = useRoute()
const token = computed(() => route.query.token || '')
const email = ref(route.query.email || '')
const password = ref('')
const confirmPassword = ref('')
const busy = ref(false)
const done = ref(false)
const error = ref('')
const message = ref('')

async function requestReset() {
  busy.value = true
  error.value = ''
  try {
    const res = await api.requestPasswordReset(email.value)
    message.value = res.message
    done.value = true
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = false
  }
}

async function confirm() {
  error.value = ''
  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }
  busy.value = true
  try {
    await api.confirmPasswordReset(token.value, password.value)
    done.value = true
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = false
  }
}
</script>
