<template>
  <div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
    <div class="min-w-0">
      <div class="flex items-center gap-2">
        <h1 class="page-title truncate">{{ title }}</h1>
        <slot name="title-extra" />
      </div>
      <p v-if="subtitle" class="page-subtitle">{{ subtitle }}</p>
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2"><slot name="actions" /></div>
  </div>
</template>

<script setup>
defineProps({ title: String, subtitle: String })
</script>
