<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-150 ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100"
      leave-active-class="transition duration-100 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0"
    >
      <div v-if="open" class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-0 sm:p-6" @keydown.esc="close">
        <div class="absolute inset-0 bg-black/50 backdrop-blur-sm" @click="dismissible && close()" />
        <div
          ref="panel"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="title ? titleId : undefined"
          tabindex="-1"
          class="relative w-full bg-elevated border border-line shadow-pop flex flex-col max-h-[92vh] animate-scale-in
            rounded-t-2xl sm:rounded-2xl focus:outline-none"
          :class="sizeClass"
        >
          <header v-if="title || $slots.header" class="flex items-start justify-between gap-4 px-6 pt-5 pb-4 border-b border-line">
            <slot name="header">
              <div class="min-w-0">
                <h2 :id="titleId" class="text-base font-semibold text-fg">{{ title }}</h2>
                <p v-if="description" class="mt-0.5 text-xs text-fg-muted">{{ description }}</p>
              </div>
            </slot>
            <button class="btn-ghost btn-icon -mr-2 -mt-1" aria-label="Close" @click="close">
              <X class="w-4 h-4" />
            </button>
          </header>
          <div class="overflow-y-auto px-6 py-5" :class="bodyClass"><slot /></div>
          <footer v-if="$slots.footer" class="flex items-center justify-end gap-2 px-6 py-4 border-t border-line bg-subtle/40 rounded-b-2xl">
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, ref, watch, onBeforeUnmount } from 'vue'
import { X } from 'lucide-vue-next'

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: '' },
  description: { type: String, default: '' },
  size: { type: String, default: 'md' }, // sm | md | lg | xl | 2xl
  dismissible: { type: Boolean, default: true },
  bodyClass: { type: String, default: '' }
})
const emit = defineEmits(['close', 'update:open'])
const panel = ref(null)
const titleId = `modal-title-${Math.random().toString(36).slice(2, 8)}`

const sizeClass = computed(() => ({
  sm: 'sm:max-w-sm', md: 'sm:max-w-lg', lg: 'sm:max-w-2xl', xl: 'sm:max-w-4xl', '2xl': 'sm:max-w-6xl'
})[props.size])

function close() {
  emit('update:open', false)
  emit('close')
}

function onKey(e) {
  if (e.key === 'Escape' && props.open && props.dismissible) close()
}

watch(() => props.open, async (o) => {
  if (o) {
    document.addEventListener('keydown', onKey)
    document.body.style.overflow = 'hidden'
    await nextTick()
    const focusable = panel.value?.querySelector('input:not([type=hidden]),select,textarea,[autofocus]')
    ;(focusable || panel.value)?.focus()
  } else {
    document.removeEventListener('keydown', onKey)
    document.body.style.overflow = ''
  }
})
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKey)
  document.body.style.overflow = ''
})
</script>
