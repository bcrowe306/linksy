<template>
  <span
    ref="anchor"
    class="inline-flex"
    @mouseenter="show" @mouseleave="hide" @focusin="show" @focusout="hide"
  >
    <slot>
      <button type="button" class="inline-flex text-fg-subtle hover:text-fg-muted focus:outline-none" :aria-label="text">
        <HelpCircle class="w-3.5 h-3.5" />
      </button>
    </slot>
  <Teleport to="body">
    <div
      v-if="visible && (text || $slots.content)"
      role="tooltip"
      class="fixed z-[70] max-w-xs rounded-lg bg-gray-50 px-2.5 py-1.5 text-xs leading-relaxed text-gray-950 shadow-pop pointer-events-none animate-fade-in"
      :style="style"
    >
      <slot name="content">{{ text }}</slot>
      <span v-if="shortcut" class="ml-1.5 opacity-60 font-mono">{{ shortcut }}</span>
    </div>
  </Teleport>
  </span>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue'
import { HelpCircle } from 'lucide-vue-next'

// Accessible hover/focus tooltip, positioned in a portal so it is never clipped.
const props = defineProps({
  text: { type: String, default: '' },
  shortcut: { type: String, default: '' },
  placement: { type: String, default: 'top' } // top | bottom | right
})
const anchor = ref(null)
const visible = ref(false)
const pos = ref({ x: 0, y: 0 })
let timer = null

const style = computed(() => {
  if (props.placement === 'right') return { left: `${pos.value.x}px`, top: `${pos.value.y}px`, transform: 'translateY(-50%)' }
  return {
    left: `${pos.value.x}px`,
    top: `${pos.value.y}px`,
    transform: props.placement === 'bottom' ? 'translateX(-50%)' : 'translate(-50%, -100%)'
  }
})

function show() {
  clearTimeout(timer)
  timer = setTimeout(async () => {
    const r = anchor.value?.getBoundingClientRect()
    if (!r) return
    if (props.placement === 'right') pos.value = { x: r.right + 8, y: r.top + r.height / 2 }
    else if (props.placement === 'bottom') pos.value = { x: r.left + r.width / 2, y: r.bottom + 8 }
    else pos.value = { x: Math.min(Math.max(r.left + r.width / 2, 120), window.innerWidth - 120), y: r.top - 8 }
    visible.value = true
    await nextTick()
  }, 250)
}
function hide() {
  clearTimeout(timer)
  visible.value = false
}
</script>
