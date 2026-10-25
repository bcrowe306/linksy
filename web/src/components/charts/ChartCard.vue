<template>
  <section class="card flex flex-col p-5">
    <header class="mb-4 flex items-start justify-between gap-3">
      <div class="min-w-0">
        <h3 class="flex items-center gap-1.5 text-sm font-semibold text-fg">{{ title }} <Tip v-if="tip" :text="tip" /></h3>
        <p v-if="subtitle" class="text-xs text-fg-muted">{{ subtitle }}</p>
      </div>
      <div class="flex items-center gap-1">
        <slot name="actions" />
        <Tip v-if="table" :text="showTable ? 'Show chart' : 'Show as table'">
          <button class="btn-ghost btn-icon" :aria-pressed="showTable" :aria-label="showTable ? 'Show chart' : 'Show as table'" @click="showTable = !showTable">
            <BarChart3 v-if="showTable" class="h-4 w-4" /><Table2 v-else class="h-4 w-4" />
          </button>
        </Tip>
      </div>
    </header>
    <div v-if="showTable && table" class="max-h-80 overflow-auto rounded-xl border border-line">
      <table class="table-base text-xs">
        <thead><tr><th v-for="c in table.columns" :key="c">{{ c }}</th></tr></thead>
        <tbody><tr v-for="(r, i) in table.rows" :key="i"><td v-for="(v, j) in r" :key="j" :class="j > 0 ? 'tabular-nums' : ''">{{ v }}</td></tr></tbody>
      </table>
    </div>
    <div v-else class="flex-1"><slot /></div>
  </section>
</template>

<script setup>
import { ref } from 'vue'
import { BarChart3, Table2 } from 'lucide-vue-next'
import Tip from '../ui/Tip.vue'

// table: { columns: string[], rows: any[][] } – the accessible twin of the chart.
defineProps({ title: String, subtitle: String, tip: String, table: Object })
const showTable = ref(false)
</script>
