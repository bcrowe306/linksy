<template>
  <div ref="wrap" class="relative w-full" :class="{ 'opacity-60 transition-opacity': loading }">
    <svg v-if="paths.length" :viewBox="`0 0 ${W} ${H}`" class="h-auto w-full" role="img" :aria-label="`Engagement by country, ${rows.length} countries`">
      <path
        v-for="p in paths"
        :key="p.id"
        :d="p.d"
        :fill="fill(p.a2)"
        stroke="rgb(var(--surface))"
        stroke-width="0.5"
        class="cursor-default transition-opacity hover:opacity-80"
        @pointerenter="hover = p"
        @pointermove="move"
        @pointerleave="hover = null"
      />
    </svg>
    <div v-else class="skeleton aspect-[2/1] w-full" />

    <div v-if="hover" class="pointer-events-none absolute z-10 rounded-xl border border-line bg-elevated px-3 py-2 text-xs shadow-pop" :style="{ left: `${pos.x + 12}px`, top: `${pos.y + 12}px` }">
      <p class="font-semibold tabular-nums text-fg">{{ (counts[hover.a2] || 0).toLocaleString() }}</p>
      <p class="text-fg-muted">{{ flag(hover.a2) }} {{ names?.name(hover.a2) || hover.a2 }}</p>
    </div>

    <!-- Sequential legend -->
    <div class="mt-3 flex items-center gap-2 text-[11px] text-fg-subtle">
      <span>0</span>
      <span class="flex h-2 flex-1 max-w-48 overflow-hidden rounded-full">
        <span v-for="i in 6" :key="i" class="flex-1" :style="{ background: `rgb(var(--seq-${i}))` }" />
      </span>
      <span class="tabular-nums">{{ max.toLocaleString() }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { loadCountries, flag } from '../../lib/countries'

const props = defineProps({ rows: { type: Array, default: () => [] }, loading: Boolean }) // [{ name: 'US', count }]
const W = 960, H = 470
const wrap = ref(null)
const paths = ref([])
const names = ref(null)
const hover = ref(null)
const pos = ref({ x: 0, y: 0 })

const counts = computed(() => Object.fromEntries(props.rows.map((r) => [r.name, r.count])))
const max = computed(() => Math.max(1, ...props.rows.map((r) => r.count)))

// Sequential blue, light→dark; countries with no data stay on the base step.
function fill(a2) {
  const n = counts.value[a2]
  if (!n) return 'rgb(var(--seq-0))'
  const step = Math.min(6, 1 + Math.floor((Math.sqrt(n) / Math.sqrt(max.value)) * 5.999))
  return `rgb(var(--seq-${step}))`
}

function move(e) {
  const r = wrap.value.getBoundingClientRect()
  pos.value = { x: Math.min(e.clientX - r.left, r.width - 160), y: e.clientY - r.top }
}

onMounted(async () => {
  const [{ geoNaturalEarth1, geoPath }, topo, world, c] = await Promise.all([
    import('d3-geo'),
    import('topojson-client'),
    import('world-atlas/countries-110m.json'),
    loadCountries()
  ])
  names.value = c
  const fc = topo.feature(world.default || world, (world.default || world).objects.countries)
  const features = fc.features.filter((f) => f.id !== '010') // drop Antarctica
  const projection = geoNaturalEarth1().fitSize([W, H], { type: 'FeatureCollection', features })
  const path = geoPath(projection)
  paths.value = features.map((f) => ({ id: f.id, a2: c.alpha2FromNumeric(f.id), d: path(f) })).filter((p) => p.d)
})
watch(() => props.rows, () => {})
</script>
