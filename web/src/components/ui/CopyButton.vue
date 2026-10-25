<template>
  <Tip :text="copied ? 'Copied!' : label">
    <button type="button" class="btn-ghost btn-icon" :aria-label="label" @click.stop="copy">
      <Check v-if="copied" class="w-3.5 h-3.5 text-emerald-500" />
      <Copy v-else class="w-3.5 h-3.5" />
    </button>
  </Tip>
</template>

<script setup>
import { ref } from 'vue'
import { Copy, Check } from 'lucide-vue-next'
import Tip from './Tip.vue'
import { copyToClipboard } from '../../composables/useToast'

const props = defineProps({ value: { type: String, required: true }, label: { type: String, default: 'Copy' } })
const copied = ref(false)
async function copy() {
  if (await copyToClipboard(props.value)) {
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  }
}
</script>
