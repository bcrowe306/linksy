<template>
  <form @submit.prevent="submit" class="space-y-4" data-testid="verify-email-form">
    <div class="p-3 rounded-xl bg-lynqr-500/10 border border-lynqr-500/20 text-xs text-lynqr-200 flex items-start gap-2">
      <MailCheck class="w-4 h-4 shrink-0 mt-0.5" />
      <span>We sent a 6-digit verification code to <strong>{{ email }}</strong>. Enter it below to finish signing in.</span>
    </div>

    <div v-if="error" class="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs flex items-center gap-2">
      <AlertCircle class="w-4 h-4 shrink-0" />
      <span>{{ error }}</span>
    </div>

    <div>
      <label for="verify-code" class="block text-xs font-medium text-gray-300 mb-1">Verification code</label>
      <input
        id="verify-code"
        v-model="code"
        inputmode="numeric"
        autocomplete="one-time-code"
        maxlength="8"
        required
        placeholder="123456"
        class="w-full bg-gray-800 border border-gray-700 rounded-xl px-3.5 py-2 text-lg tracking-[0.4em] font-mono text-center text-gray-100 placeholder-gray-600 focus:outline-none focus:ring-2 focus:ring-lynqr-500"
      />
    </div>

    <button
      type="submit"
      :disabled="busy || code.trim().length < 4"
      class="w-full py-2.5 px-4 bg-lynqr-600 hover:bg-lynqr-500 text-white rounded-xl text-sm font-semibold flex items-center justify-center gap-2 disabled:opacity-50"
    >
      <Loader2 v-if="busy" class="w-4 h-4 animate-spin" />
      <span v-else>Verify &amp; continue</span>
    </button>
    <button type="button" @click="$emit('cancel')" class="w-full text-xs text-gray-400 hover:text-gray-200">
      Use a different account
    </button>
  </form>
</template>

<script setup>
import { ref } from 'vue'
import { MailCheck, AlertCircle, Loader2 } from 'lucide-vue-next'
import { useAuthStore } from '../stores/auth'

const props = defineProps({
  pendingToken: { type: String, required: true },
  email: { type: String, default: '' }
})
const emit = defineEmits(['verified', 'cancel'])
const authStore = useAuthStore()
const code = ref('')
const busy = ref(false)
const error = ref('')

async function submit() {
  busy.value = true
  error.value = ''
  try {
    await authStore.verifyEmail(props.pendingToken, code.value.trim())
    emit('verified')
  } catch (err) {
    error.value = err.message || 'That code did not work. Please try again.'
  } finally {
    busy.value = false
  }
}
</script>
