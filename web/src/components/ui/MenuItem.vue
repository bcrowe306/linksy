<template>
  <component
    :is="to ? 'router-link' : 'button'"
    :to="to"
    type="button"
    role="menuitem"
    class="flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-left text-sm transition-colors"
    :class="danger ? 'text-rose-500 hover:bg-rose-500/10' : 'text-fg-muted hover:bg-subtle hover:text-fg'"
  >
    <component :is="icon" v-if="icon" class="w-4 h-4 shrink-0" />
    <span class="flex-1 truncate"><slot /></span>
    <span v-if="shortcut" class="kbd">{{ shortcut }}</span>
  </component>
</template>

<script setup>
defineProps({ icon: [Object, Function], to: [String, Object], danger: Boolean, shortcut: String })
</script>
