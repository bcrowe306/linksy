<template>
  <div class="min-h-screen flex items-center justify-center p-6 bg-canvas">
    <div class="w-full max-w-md card p-8 shadow-pop text-center animate-slide-up">
      <div class="w-12 h-12 mx-auto rounded-2xl bg-brand-500/10 text-brand-500 flex items-center justify-center mb-4">
        <Users class="w-6 h-6" />
      </div>

      <div v-if="loading" class="text-sm text-fg-muted">Loading invitation…</div>

      <template v-else-if="invite">
        <h1 class="text-xl font-bold text-fg mb-1">Join {{ invite.team_name }}</h1>
        <p class="text-sm text-fg-muted mb-6">
          You've been invited as <span class="font-semibold text-fg capitalize">{{ invite.role }}</span>
          ({{ invite.email }}).
        </p>

        <p v-if="error" class="mb-4 text-xs text-rose-400">{{ error }}</p>

        <template v-if="authStore.isAuthenticated">
          <p v-if="emailMismatch" class="mb-4 text-xs text-amber-600 dark:text-amber-300">
            You're signed in as {{ authStore.user.email }}. Sign in as {{ invite.email }} to accept.
          </p>
          <div class="flex gap-3 justify-center">
            <button v-if="!emailMismatch" @click="accept" :disabled="busy"
              class="btn-primary">
              {{ busy ? 'Joining…' : 'Accept invitation' }}
            </button>
            <button v-else @click="authStore.logout" class="btn-secondary">
              Switch account
            </button>
          </div>
        </template>
        <div v-else class="flex gap-3 justify-center">
          <router-link :to="{ path: '/login', query: { redirect: route.fullPath } }"
            class="btn-primary">Sign in to accept</router-link>
          <router-link :to="{ path: '/signup', query: { redirect: route.fullPath } }"
            class="btn-secondary">Create account</router-link>
        </div>
      </template>

      <template v-else>
        <h1 class="text-xl font-bold text-fg mb-2">Invitation unavailable</h1>
        <p class="text-sm text-fg-muted mb-6">{{ error || 'This invitation is invalid, expired, or has already been used.' }}</p>
        <router-link to="/" class="text-sm text-brand-500 hover:underline font-semibold">Go to dashboard</router-link>
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
