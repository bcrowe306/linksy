<template>
  <AppModal :open="open" :title="title" :description="description" size="sm" @close="$emit('close')">
    <form class="space-y-3" @submit.prevent="submit">
      <div>
        <label class="label" for="reason-input">{{ label }}</label>
        <textarea id="reason-input" v-model="text" rows="3" maxlength="300" class="input" required :placeholder="placeholder" data-testid="reason-input" />
      </div>
      <div class="flex justify-end gap-2">
        <button type="button" class="btn-ghost" @click="$emit('close')">Cancel</button>
        <button class="btn-danger" :disabled="!text.trim()" data-testid="reason-submit">{{ confirmText }}</button>
      </div>
    </form>
  </AppModal>
</template>

<script setup>
import { ref, watch } from 'vue'
import AppModal from '../ui/AppModal.vue'

const props = defineProps({
  open: Boolean,
  title: { type: String, default: 'Give a reason' },
  description: { type: String, default: '' },
  label: { type: String, default: 'Reason (kept in the platform audit log)' },
  placeholder: { type: String, default: 'e.g. Phishing pages impersonating a bank' },
  confirmText: { type: String, default: 'Confirm' }
})
const emit = defineEmits(['close', 'submit'])
const text = ref('')
watch(() => props.open, (o) => o && (text.value = ''))
function submit() {
  emit('submit', text.value.trim())
}
</script>
