<template>
  <AppModal :open="helpState.shortcuts" title="Keyboard shortcuts" size="md" @close="helpState.shortcuts = false">
    <div class="grid gap-6 sm:grid-cols-2" data-testid="shortcuts-modal">
      <section v-for="g in SHORTCUT_GROUPS" :key="g.title">
        <h3 class="mb-2 text-xs font-semibold uppercase tracking-wider text-fg-subtle">{{ g.title }}</h3>
        <ul class="space-y-1.5">
          <li v-for="s in g.items" :key="s.label" class="flex items-center justify-between gap-3 text-sm">
            <span class="text-fg-muted">{{ s.label }}</span>
            <span class="flex gap-1"><kbd v-for="k in s.keys" :key="k" class="kbd">{{ k }}</kbd></span>
          </li>
        </ul>
      </section>
    </div>
  </AppModal>
</template>

<script setup>
import AppModal from './ui/AppModal.vue'
import { helpState } from '../composables/useHelp'
import { SHORTCUT_GROUPS } from '../composables/useShortcuts'
</script>
