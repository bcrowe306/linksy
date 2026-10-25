<template>
  <div class="flex flex-col items-center justify-center text-center px-6 py-14">
    <div class="relative mb-4">
      <div class="absolute inset-0 rounded-2xl bg-brand-500/20 blur-xl" />
      <div class="relative w-12 h-12 rounded-2xl bg-gradient-to-br from-brand-500/15 to-violet-500/15 border border-brand-500/20 flex items-center justify-center text-brand-500">
        <component :is="icon" class="w-6 h-6" />
      </div>
    </div>
    <h3 class="text-sm font-semibold text-fg">{{ title }}</h3>
    <p v-if="description" class="mt-1 max-w-sm text-sm text-fg-muted">{{ description }}</p>
    <div v-if="$slots.default" class="mt-5 flex flex-wrap items-center justify-center gap-2"><slot /></div>
  </div>
</template>

<script setup>
defineProps({ icon: [Object, Function], title: String, description: String })
</script>
