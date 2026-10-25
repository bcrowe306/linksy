<template>
  <div class="min-h-screen flex items-center justify-center p-6">
    <div class="w-full max-w-md bg-gray-900/80 border border-gray-800 rounded-2xl p-8 shadow-2xl text-center">
      <div class="w-12 h-12 mx-auto rounded-2xl bg-lynqr-600/20 text-lynqr-300 flex items-center justify-center mb-4">
        <Users class="w-6 h-6" />
      </div>

      <div v-if="loading" class="text-sm text-gray-400">Loading invitation…</div>

      <template v-else-if="invite">
        <h1 class="text-xl font-bold text-white mb-1">Join {{ invite.team_name }}</h1>
        <p class="text-sm text-gray-400 mb-6">
          You've been invited as <span class="font-semibold text-gray-200 capitalize">{{ invite.role }}</span>
          ({{ invite.email }}).
        </p>

        <p v-if="error" class="mb-4 text-xs text-rose-400">{{ error }}</p>

        <template v-if="authStore.isAuthenticated">
          <p v-if="emailMismatch" class="mb-4 text-xs text-amber-300">
            You're signed in as {{ authStore.user.email }}. Sign in as {{ invite.email }} to accept.
          </p>
          <div class="flex gap-3 justify-center">
            <button v-if="!emailMismatch" @click="accept" :disabled="busy"
              class="px-5 py-2.5 bg-lynqr-600 hover:bg-lynqr-500 text-white rounded-xl text-sm font-semibold disabled:opacity-50">
              {{ busy ? 'Joining…' : 'Accept invitation' }}
            </button>
            <button v-else @click="authStore.logout" class="px-5 py-2.5 bg-gray-800 hover:bg-gray-700 text-gray-100 rounded-xl text-sm font-semibold">
              Switch account
            </button>
          </div>
        </template>
        <div v-else class="flex gap-3 justify-center">
          <router-link :to="{ path: '/login', query: { redirect: route.fullPath } }"
            class="px-5 py-2.5 bg-lynqr-600 hover:bg-lynqr-500 text-white rounded-xl text-sm font-semibold">Sign in to accept</router-link>
          <router-link :to="{ path: '/signup', query: { redirect: route.fullPath } }"
            class="px-5 py-2.5 bg-gray-800 hover:bg-gray-700 text-gray-100 rounded-xl text-sm font-semibold">Create account</router-link>
        </div>
      </template>

      <template v-else>
        <h1 class="text-xl font-bold text-white mb-2">Invitation unavailable</h1>
        <p class="text-sm text-gray-400 mb-6">{{ error || 'This invitation is invalid, expired, or has already been used.' }}</p>
        <router-link to="/" class="text-sm text-lynqr-400 hover:text-lynqr-300 font-semibold">Go to dashboard</router-link>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Users } from 'lucide-vue-next'
import { api } from '../api/client'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const invite = ref(null)
const loading = ref(true)
const busy = ref(false)
const error = ref('')

const emailMismatch = computed(() =>
  invite.value && authStore.user && authStore.user.email.toLowerCase() !== invite.value.email.toLowerCase()
)

onMounted(async () => {
  try {
    invite.value = await api.previewInvitation(route.params.token)
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
})

async function accept() {
  busy.value = true
  error.value = ''
  try {
    await api.acceptInvitation(route.params.token)
    await authStore.refresh()
    router.push('/')
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = false
  }
}
</script>
