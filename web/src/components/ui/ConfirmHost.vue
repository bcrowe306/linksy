<template>
  <AppModal :open="confirmState.open" size="sm" :dismissible="true" @close="settleConfirm(false)">
    <div class="flex gap-4">
      <div class="w-10 h-10 rounded-full flex items-center justify-center shrink-0"
        :class="confirmState.danger ? 'bg-rose-500/10 text-rose-500' : 'bg-brand-500/10 text-brand-500'">
        <AlertTriangle v-if="confirmState.danger" class="w-5 h-5" />
        <HelpCircle v-else class="w-5 h-5" />
      </div>
      <div class="min-w-0">
        <h2 class="text-base font-semibold text-fg">{{ confirmState.title }}</h2>
        <p class="mt-1 text-sm text-fg-muted whitespace-pre-line">{{ confirmState.message }}</p>
        <div v-if="confirmState.requireText" class="mt-4">
          <label class="label">Type <span class="font-mono text-fg">{{ confirmState.requireText }}</span> to confirm</label>
          <input v-model="typed" class="input" autocomplete="off" />
        </div>
      </div>
    </div>
    <template #footer>
      <button class="btn-secondary" @click="settleConfirm(false)">{{ confirmState.cancelText }}</button>
      <button
        :class="confirmState.danger ? 'btn-danger' : 'btn-primary'"
        :disabled="!!confirmState.requireText && typed !== confirmState.requireText"
        data-testid="confirm-accept"
        @click="settleConfirm(true)"
      >
        {{ confirmState.confirmText }}
      </button>
    </template>
  </AppModal>
</template>

<script setup>
import { ref, watch } from 'vue'
import { AlertTriangle, HelpCircle } from 'lucide-vue-next'
import AppModal from './AppModal.vue'
import { confirmState, settleConfirm } from '../../composables/useConfirm'

const typed = ref('')
watch(() => confirmState.open, () => (typed.value = ''))
</script>
