<template>
  <Teleport to="body">
    <div class="fixed bottom-4 right-4 z-[60] flex flex-col gap-2 w-[calc(100vw-2rem)] max-w-sm pointer-events-none" aria-live="polite">
      <TransitionGroup
        enter-active-class="transition duration-200 ease-out" enter-from-class="opacity-0 translate-y-2 scale-95"
        leave-active-class="transition duration-150 ease-in" leave-to-class="opacity-0 translate-x-4"
      >
        <div
          v-for="t in toastState.items"
          :key="t.id"
          class="pointer-events-auto flex items-start gap-3 rounded-xl border border-line bg-elevated px-4 py-3 shadow-pop"
          role="status"
        >
          <component :is="icons[t.type]" class="w-4 h-4 mt-0.5 shrink-0" :class="colors[t.type]" />
          <div class="min-w-0 flex-1 text-sm">
            <p v-if="t.title" class="font-semibold text-fg">{{ t.title }}</p>
            <p class="text-fg-muted break-words">{{ t.message }}</p>
          </div>
          <button class="text-fg-subtle hover:text-fg" aria-label="Dismiss" @click="dismiss(t.id)">
            <X class="w-3.5 h-3.5" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup>
import { CheckCircle2, AlertCircle, Info, AlertTriangle, X } from 'lucide-vue-next'
import { toastState, dismiss } from '../../composables/useToast'

const icons = { success: CheckCircle2, error: AlertCircle, info: Info, warning: AlertTriangle }
const colors = { success: 'text-emerald-500', error: 'text-rose-500', info: 'text-brand-500', warning: 'text-amber-500' }
</script>
