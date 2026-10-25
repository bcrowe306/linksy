<template>
  <button type="button" class="inline-flex items-center gap-1 text-xs font-medium text-brand-500 hover:underline" :data-testid="`help-${guide}${anchor ? '-' + anchor : ''}`" @click="open(guide, anchor)">
    <BookOpen class="h-3.5 w-3.5" /><slot>Learn more</slot>
  </button>
</template>

<script setup>
import { BookOpen } from 'lucide-vue-next'
import { useHelp } from '../../composables/useHelp'

defineProps({ guide: { type: String, required: true }, anchor: { type: String, default: '' } })
const { open } = useHelp()
</script>
