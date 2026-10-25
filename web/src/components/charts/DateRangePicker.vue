<template>
  <DropdownMenu align="left" :close-on-click="false">
    <template #trigger>
      <button class="btn-secondary" data-testid="date-range"><CalendarDays class="h-4 w-4 text-fg-subtle" />{{ currentLabel }}<ChevronDown class="h-4 w-4 text-fg-subtle" /></button>
    </template>
    <template #default="{ close }">
      <div class="w-64">
        <button
          v-for="p in presets"
          :key="p.id"
          class="flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-sm hover:bg-subtle"
          :data-testid="`range-${p.id}`"
          @click="select(p.id, close)"
        >
          <Check class="h-4 w-4 stroke-[3]" :class="modelValue.preset === p.id ? 'text-brand-500' : 'invisible'" />{{ p.label }}
        </button>
        <div class="mt-1 border-t border-line px-2.5 pb-1 pt-2.5">
          <p class="mb-2 text-xs font-medium text-fg-muted">Custom range</p>
          <div class="grid grid-cols-2 gap-2">
            <input v-model="from" type="date" class="input input-sm" :max="to || today" aria-label="From date" />
            <input v-model="to" type="date" class="input input-sm" :min="from" :max="today" aria-label="To date" />
          </div>
          <button class="btn-primary btn-sm mt-2 w-full" :disabled="!from || !to || from > to" @click="applyCustom(close)">Apply</button>
        </div>
      </div>
    </template>
  </DropdownMenu>
</template>

<script setup>
import { ref, computed } from 'vue'
import { CalendarDays, ChevronDown, Check } from 'lucide-vue-next'
import DropdownMenu from '../ui/DropdownMenu.vue'

// modelValue: { preset: 'today'|'7d'|'30d'|'90d'|'12m'|'custom', from: 'YYYY-MM-DD', to: 'YYYY-MM-DD' }
const props = defineProps({ modelValue: { type: Object, required: true } })
const emit = defineEmits(['update:modelValue'])
const presets = [
  { id: 'today', label: 'Today' },
  { id: '7d', label: 'Last 7 days' },
  { id: '30d', label: 'Last 30 days' },
  { id: '90d', label: 'Last 90 days' },
  { id: '12m', label: 'Last 12 months' }
]
const today = new Date().toISOString().slice(0, 10)
const from = ref(props.modelValue.from || '')
const to = ref(props.modelValue.to || '')

const currentLabel = computed(() => {
  if (props.modelValue.preset !== 'custom') return presets.find((p) => p.id === props.modelValue.preset)?.label || 'Last 7 days'
  const f = (d) => new Date(`${d}T00:00`).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  return `${f(props.modelValue.from)} – ${f(props.modelValue.to)}`
})

function select(id, close) {
  emit('update:modelValue', { preset: id })
  close()
}
function applyCustom(close) {
  emit('update:modelValue', { preset: 'custom', from: from.value, to: to.value })
  close()
}
</script>
