<template>
  <div ref="root" class="relative inline-flex">
    <div @click="toggle"><slot name="trigger" :open="open" /></div>
    <Transition
      enter-active-class="transition duration-100 ease-out" enter-from-class="opacity-0 scale-95"
      leave-active-class="transition duration-75 ease-in" leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="open"
        role="menu"
        class="absolute z-40 mt-2 min-w-[12rem] rounded-xl border border-line bg-elevated p-1.5 shadow-pop origin-top"
        :class="[align === 'right' ? 'right-0' : 'left-0', 'top-full']"
        @click="closeOnClick && (open = false)"
      >
        <slot :close="() => (open = false)" />
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'

defineProps({
  align: { type: String, default: 'right' },
  closeOnClick: { type: Boolean, default: true }
})
const open = ref(false)
const root = ref(null)

function toggle() {
  open.value = !open.value
}
function onDoc(e) {
  if (root.value && !e.composedPath().includes(root.value)) open.value = false
}
function onKey(e) {
  if (e.key === 'Escape') open.value = false
}
onMounted(() => {
  document.addEventListener('click', onDoc)
  document.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', onDoc)
  document.removeEventListener('keydown', onKey)
})
</script>
