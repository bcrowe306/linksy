<template>
  <form @submit.prevent="submit" class="space-y-4" data-testid="verify-email-form">
    <div class="p-3 rounded-xl bg-brand-500/10 border border-brand-500/20 text-xs text-brand-700 dark:text-brand-200 flex items-start gap-2">
      <MailCheck class="w-4 h-4 shrink-0 mt-0.5" />
      <span>We sent a 6-digit verification code to <strong>{{ email }}</strong>. Enter it below to finish signing in.</span>
    </div>

    <div v-if="error" class="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs flex items-center gap-2">
      <AlertCircle class="w-4 h-4 shrink-0" />
      <span>{{ error }}</span>
    </div>

    <div>
      <label for="verify-code" class="label">Verification code</label>
      <input
        id="verify-code"
        v-model="code"
        inputmode="numeric"
        autocomplete="one-time-code"
        maxlength="8"
        required
        placeholder="123456"
        class="input text-lg tracking-[0.4em] font-mono text-center"
      />
    </div>

    <button
      type="submit"
      :disabled="busy || code.trim().length < 4"
      class="btn-primary btn-lg w-full"
    >
      <Loader2 v-if="busy" class="w-4 h-4 animate-spin" />
      <span v-else>Verify &amp; continue</span>
    </button>
    <button type="button" @click="$emit('cancel')" class="w-full text-xs text-fg-muted hover:text-fg">
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
