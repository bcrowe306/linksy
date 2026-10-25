<template>
  <AuthLayout title="Welcome back" subtitle="Sign in to your Lynqr workspace.">
    <div v-if="errorMessage" class="mb-5 flex items-start gap-2 rounded-xl border border-rose-500/20 bg-rose-500/10 px-3 py-2.5 text-sm text-rose-600 dark:text-rose-400" role="alert">
      <AlertCircle class="mt-0.5 h-4 w-4 shrink-0" /><span>{{ errorMessage }}</span>
    </div>

    <VerifyEmailForm v-if="verification" :pending-token="verification.pendingToken" :email="verification.email || email" @verified="finish" @cancel="verification = null" />

    <template v-else>
      <OAuthButtons screen-hint="sign-in" />

      <form class="space-y-4" @submit.prevent="handleSubmit">
        <div>
          <label class="label" for="login-email">Email</label>
          <input id="login-email" v-model="email" type="email" required autocomplete="email" placeholder="jane@company.com" class="input" />
        </div>
        <div>
          <div class="mb-1.5 flex items-center justify-between">
            <label class="label !mb-0" for="login-password">Password</label>
            <router-link :to="{ path: '/reset-password', query: { email } }" class="text-xs font-medium text-brand-500 hover:underline">Forgot password?</router-link>
          </div>
          <div class="relative">
            <input id="login-password" v-model="password" :type="showPassword ? 'text' : 'password'" required autocomplete="current-password" placeholder="••••••••" class="input pr-10" />
            <button type="button" class="absolute right-2.5 top-1/2 -translate-y-1/2 text-fg-subtle hover:text-fg" :aria-label="showPassword ? 'Hide password' : 'Show password'" @click="showPassword = !showPassword">
              <EyeOff v-if="showPassword" class="h-4 w-4" /><Eye v-else class="h-4 w-4" />
            </button>
          </div>
        </div>
        <button type="submit" :disabled="isSubmitting" class="btn-primary btn-lg w-full">
          <Loader2 v-if="isSubmitting" class="h-4 w-4 animate-spin" /><span v-else>Sign in</span>
        </button>
      </form>

      <!-- Local development sign-in (only when the server enables DEV_LOGIN on localhost) -->
      <form v-if="authStore.authConfig.dev_login" class="mt-6 space-y-2 rounded-xl border border-dashed border-amber-500/40 p-3" data-testid="dev-login" @submit.prevent="devLogin">
        <p class="text-[11px] font-semibold uppercase tracking-wider text-amber-500">Local dev sign-in</p>
        <div class="flex gap-2">
          <input v-model="devEmail" type="email" required placeholder="you@example.com" class="input input-sm flex-1" />
          <button type="submit" class="btn btn-sm bg-amber-500 text-white hover:bg-amber-400">Continue</button>
        </div>
      </form>

      <p class="mt-8 text-center text-sm text-fg-muted">
        New to Lynqr?
        <router-link :to="{ path: '/signup', query: route.query }" class="font-semibold text-brand-500 hover:underline">Create an account</router-link>
      </p>
    </template>
  </AuthLayout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { AlertCircle, Eye, EyeOff, Loader2 } from 'lucide-vue-next'
import { useAuthStore } from '../stores/auth'
import AuthLayout from '../components/AuthLayout.vue'
import OAuthButtons from '../components/OAuthButtons.vue'
import VerifyEmailForm from '../components/VerifyEmailForm.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const showPassword = ref(false)
const isSubmitting = ref(false)
const errorMessage = ref('')
const verification = ref(null)
const devEmail = ref('')

function safeRedirect(target) {
  const t = String(target || '/')
  return t.startsWith('/') && !t.startsWith('//') ? t : '/'
}
function finish() {
  router.push(safeRedirect(route.query.redirect))
}

onMounted(() => {
  if (route.query.auth_error) errorMessage.value = String(route.query.auth_error)
})

async function handleSubmit() {
  errorMessage.value = ''
  isSubmitting.value = true
  try {
    const result = await authStore.loginWithPassword(email.value, password.value)
    if (result.verificationRequired) {
      verification.value = result
      return
    }
    finish()
  } catch (err) {
    errorMessage.value = err.message || 'Invalid email or password'
  } finally {
    isSubmitting.value = false
  }
}

async function devLogin() {
  errorMessage.value = ''
  try {
    await authStore.devLogin({ email: devEmail.value, first_name: devEmail.value.split('@')[0] })
    finish()
  } catch (err) {
    errorMessage.value = err.message
  }
}
</script>
