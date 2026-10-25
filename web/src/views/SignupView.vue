<template>
  <AuthLayout title="Create your account" subtitle="Free to start. Your personal workspace is ready instantly.">
    <div v-if="errorMessage" class="mb-5 flex items-start gap-2 rounded-xl border border-rose-500/20 bg-rose-500/10 px-3 py-2.5 text-sm text-rose-600 dark:text-rose-400" role="alert">
      <AlertCircle class="mt-0.5 h-4 w-4 shrink-0" /><span>{{ errorMessage }}</span>
    </div>

    <VerifyEmailForm v-if="verification" :pending-token="verification.pendingToken" :email="verification.email || email" @verified="finish" @cancel="verification = null" />

    <template v-else>
      <OAuthButtons screen-hint="sign-up" />

      <form class="space-y-4" @submit.prevent="handleSubmit">
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="label" for="su-first">First name</label>
            <input id="su-first" v-model="firstName" required autocomplete="given-name" placeholder="Jane" class="input" />
          </div>
          <div>
            <label class="label" for="su-last">Last name</label>
            <input id="su-last" v-model="lastName" required autocomplete="family-name" placeholder="Doe" class="input" />
          </div>
        </div>
        <div>
          <label class="label" for="su-email">Work email</label>
          <input id="su-email" v-model="email" type="email" required autocomplete="email" placeholder="jane@company.com" class="input" />
        </div>
        <div>
          <label class="label" for="su-password">Password</label>
          <div class="relative">
            <input id="su-password" v-model="password" :type="showPassword ? 'text' : 'password'" required minlength="8" autocomplete="new-password" placeholder="At least 8 characters" class="input pr-10" />
            <button type="button" class="absolute right-2.5 top-1/2 -translate-y-1/2 text-fg-subtle hover:text-fg" :aria-label="showPassword ? 'Hide password' : 'Show password'" @click="showPassword = !showPassword">
              <EyeOff v-if="showPassword" class="h-4 w-4" /><Eye v-else class="h-4 w-4" />
            </button>
          </div>
          <div class="mt-2 flex gap-1" aria-hidden="true">
            <span v-for="i in 4" :key="i" class="h-1 flex-1 rounded-full transition-colors" :class="i <= strength ? strengthColor : 'bg-line'" />
          </div>
        </div>
        <div>
          <label class="label" for="su-confirm">Confirm password</label>
          <input id="su-confirm" v-model="confirmPassword" type="password" required autocomplete="new-password" placeholder="Re-enter password" class="input" />
        </div>
        <button type="submit" :disabled="isSubmitting" class="btn-primary btn-lg w-full">
          <Loader2 v-if="isSubmitting" class="h-4 w-4 animate-spin" /><span v-else>Create Free Account</span>
        </button>
        <p class="text-center text-xs text-fg-subtle">
          By creating an account you agree to the <a href="/terms" target="_blank" class="underline hover:text-fg">Terms of Service</a>
          and <a href="/privacy" target="_blank" class="underline hover:text-fg">Privacy Policy</a>.
        </p>
      </form>

      <p class="mt-8 text-center text-sm text-fg-muted">
        Already have an account?
        <router-link :to="{ path: '/login', query: route.query }" class="font-semibold text-brand-500 hover:underline">Sign in</router-link>
      </p>
    </template>
  </AuthLayout>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { AlertCircle, Eye, EyeOff, Loader2 } from 'lucide-vue-next'
import { useAuthStore } from '../stores/auth'
import AuthLayout from '../components/AuthLayout.vue'
import OAuthButtons from '../components/OAuthButtons.vue'
import VerifyEmailForm from '../components/VerifyEmailForm.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const firstName = ref('')
const lastName = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)
const isSubmitting = ref(false)
const errorMessage = ref('')
const verification = ref(null)

const strength = computed(() => {
  const p = password.value
  let s = 0
  if (p.length >= 8) s++
  if (p.length >= 12) s++
  if (/[A-Z]/.test(p) && /[a-z]/.test(p)) s++
  if (/\d/.test(p) && /[^A-Za-z0-9]/.test(p)) s++
  return s
})
const strengthColor = computed(() => ['bg-rose-500', 'bg-rose-500', 'bg-amber-500', 'bg-emerald-500', 'bg-emerald-500'][strength.value])

function finish() {
  const t = String(route.query.redirect || '/')
  router.push(t.startsWith('/') && !t.startsWith('//') ? t : '/')
}

async function handleSubmit() {
  errorMessage.value = ''
  if (password.value !== confirmPassword.value) {
    errorMessage.value = 'Passwords do not match'
    return
  }
  isSubmitting.value = true
  try {
    const result = await authStore.signup({ first_name: firstName.value, last_name: lastName.value, email: email.value, password: password.value })
    if (result.verificationRequired) {
      verification.value = result
      return
    }
    finish()
  } catch (err) {
    errorMessage.value = err.message || 'Registration failed'
  } finally {
    isSubmitting.value = false
  }
}
</script>
