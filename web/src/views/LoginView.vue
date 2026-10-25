<template>
  <div class="min-h-screen bg-gray-950 text-gray-100 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
    <div class="sm:mx-auto sm:w-full sm:max-w-md text-center">
      <!-- Logo -->
      <router-link to="/" class="inline-flex items-center gap-2.5 mb-6 group">
        <div class="w-10 h-10 rounded-xl bg-gradient-to-tr from-linksy-600 to-indigo-500 flex items-center justify-center text-white shadow-lg shadow-linksy-500/25 group-hover:scale-105 transition-transform">
          <QrCode class="w-6 h-6 stroke-[2.5]" />
        </div>
        <span class="text-2xl font-black tracking-tight text-white font-mono">
          LINKSY<span class="text-linksy-400">.</span>
        </span>
      </router-link>

      <h2 class="text-2xl font-extrabold text-white tracking-tight">
        Welcome back
      </h2>
      <p class="mt-2 text-xs text-gray-400">
        Sign in to your account to manage your QR codes and shortened links.
      </p>
    </div>

    <div class="mt-8 sm:mx-auto sm:w-full sm:max-w-md px-4 sm:px-0">
      <div class="bg-gray-900/90 border border-gray-800 rounded-2xl p-6 sm:p-8 shadow-2xl backdrop-blur-xl">

        <!-- Social OAuth Sign In -->
        <div class="space-y-2.5 mb-6">
          <!-- Google -->
          <button
            @click="authStore.loginWithOAuth('google')"
            type="button"
            class="w-full flex items-center justify-center gap-3 px-4 py-2.5 border border-gray-700 rounded-xl text-sm font-semibold text-gray-200 bg-gray-800/80 hover:bg-gray-800 hover:border-gray-600 transition-all shadow-sm active:scale-[0.98]"
          >
            <svg class="w-4 h-4" viewBox="0 0 24 24">
              <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
              <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
              <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.06H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.94l2.85-2.22.81-.63z" />
              <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52z" />
            </svg>
            <span>Continue with Google</span>
          </button>

          <!-- Microsoft -->
          <button
            @click="authStore.loginWithOAuth('microsoft')"
            type="button"
            class="w-full flex items-center justify-center gap-3 px-4 py-2.5 border border-gray-700 rounded-xl text-sm font-semibold text-gray-200 bg-gray-800/80 hover:bg-gray-800 hover:border-gray-600 transition-all shadow-sm active:scale-[0.98]"
          >
            <svg class="w-4 h-4" viewBox="0 0 23 23">
              <path fill="#f35325" d="M1 1h10v10H1z"/>
              <path fill="#81bc06" d="M12 1h10v10H12z"/>
              <path fill="#05a6f0" d="M1 12h10v10H1z"/>
              <path fill="#ffba08" d="M12 12h10v10H12z"/>
            </svg>
            <span>Continue with Microsoft</span>
          </button>

          <!-- GitHub -->
          <button
            @click="authStore.loginWithOAuth('github')"
            type="button"
            class="w-full flex items-center justify-center gap-3 px-4 py-2.5 border border-gray-700 rounded-xl text-sm font-semibold text-gray-200 bg-gray-800/80 hover:bg-gray-800 hover:border-gray-600 transition-all shadow-sm active:scale-[0.98]"
          >
            <svg class="w-4 h-4 fill-current text-white" viewBox="0 0 24 24">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
            </svg>
            <span>Continue with GitHub</span>
          </button>
        </div>

        <div class="relative mb-6">
          <div class="absolute inset-0 flex items-center">
            <div class="w-full border-t border-gray-800"></div>
          </div>
          <div class="relative flex justify-center text-xs uppercase">
            <span class="bg-gray-900 px-3 text-gray-500 font-mono tracking-wider">or sign in with email</span>
          </div>
        </div>

        <!-- Error Alert -->
        <div v-if="errorMessage" class="mb-5 p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs flex items-center gap-2">
          <AlertCircle class="w-4 h-4 shrink-0" />
          <span>{{ errorMessage }}</span>
        </div>

        <!-- Direct Email / Password Sign In Form -->
        <form @submit.prevent="handleSubmit" class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-gray-300 mb-1">Work Email</label>
            <input
              v-model="email"
              type="email"
              required
              placeholder="jane@company.com"
              class="w-full bg-gray-800 border border-gray-700 rounded-xl px-3.5 py-2 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-linksy-500"
            />
          </div>

          <div>
            <div class="flex items-center justify-between mb-1">
              <label class="block text-xs font-medium text-gray-300">Password</label>
              <button
                type="button"
                @click="authStore.loginWithOAuth('authkit')"
                class="text-[11px] text-linksy-400 hover:text-linksy-300 transition-colors"
              >
                Forgot password?
              </button>
            </div>
            <div class="relative">
              <input
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                required
                placeholder="Enter your password"
                class="w-full bg-gray-800 border border-gray-700 rounded-xl pl-3.5 pr-10 py-2 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-linksy-500"
              />
              <button
                type="button"
                @click="showPassword = !showPassword"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-200"
              >
                <EyeOff v-if="showPassword" class="w-4 h-4" />
                <Eye v-else class="w-4 h-4" />
              </button>
            </div>
          </div>

          <button
            type="submit"
            :disabled="isSubmitting"
            class="w-full py-2.5 px-4 bg-gradient-to-r from-linksy-600 to-indigo-600 hover:from-linksy-500 hover:to-indigo-500 text-white rounded-xl text-sm font-semibold flex items-center justify-center gap-2 shadow-lg shadow-linksy-600/25 transition-all hover:scale-[1.01] active:scale-[0.99] disabled:opacity-50"
          >
            <Loader2 v-if="isSubmitting" class="w-4 h-4 animate-spin" />
            <span v-else>Sign In</span>
          </button>
        </form>

        <div class="mt-6 text-center">
          <p class="text-xs text-gray-400">
            Don't have an account?
            <router-link to="/signup" class="text-linksy-400 hover:text-linksy-300 font-semibold underline underline-offset-2 ml-1">
              Create an account
            </router-link>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { QrCode, AlertCircle, Eye, EyeOff, Loader2 } from 'lucide-vue-next'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const showPassword = ref(false)
const isSubmitting = ref(false)
const errorMessage = ref('')

onMounted(() => {
  const err = route.query.auth_error
  if (err) {
    errorMessage.value = String(err)
  }
})

async function handleSubmit() {
  errorMessage.value = ''
  isSubmitting.value = true
  try {
    await authStore.loginWithPassword(email.value, password.value)
    const target = route.query.redirect || '/'
    router.push(target)
  } catch (err) {
    errorMessage.value = err.message || 'Invalid email or password'
  } finally {
    isSubmitting.value = false
  }
}
</script>
