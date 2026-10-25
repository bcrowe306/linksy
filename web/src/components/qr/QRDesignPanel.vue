<template>
  <div>
    <div class="mb-4 flex gap-1 overflow-x-auto rounded-xl bg-subtle p-1" role="tablist">
      <button
        v-for="t in tabs"
        :key="t.id"
        role="tab"
        :aria-selected="tab === t.id"
        class="flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition"
        :class="tab === t.id ? 'bg-surface text-fg shadow-card' : 'text-fg-muted hover:text-fg'"
        :data-testid="`design-tab-${t.id}`"
        @click="tab = t.id"
      >
        <component :is="t.icon" class="h-3.5 w-3.5" />{{ t.label }}
      </button>
    </div>

    <!-- Templates -->
    <div v-if="tab === 'templates'" class="grid grid-cols-2 gap-3 sm:grid-cols-3">
      <button
        v-for="p in STYLE_PRESETS"
        :key="p.name"
        type="button"
        class="group overflow-hidden rounded-xl border border-line text-left transition hover:border-brand-500 hover:shadow-glow"
        @click="applyPreset(p)"
      >
        <div class="h-14" :style="{ background: `linear-gradient(135deg, ${p.swatch[0]}, ${p.swatch[1]})` }" />
        <div class="px-3 py-2 text-xs font-semibold text-fg">{{ p.name }}</div>
      </button>
    </div>

    <!-- Pattern -->
    <div v-else-if="tab === 'pattern'" class="space-y-5">
      <ShapePicker label="Module shape" tip="The style of the small data squares." kind="dot" :options="opts.dot" v-model="s.dot_style" />
      <ShapePicker label="Corner frame" tip="The three large 'eyes' scanners use to find the code." kind="eye" :options="opts.eye" v-model="s.eye_shape" />
      <ShapePicker label="Corner center" kind="inner" :options="opts.inner" v-model="s.eye_inner_shape" />
    </div>

    <!-- Colors -->
    <div v-else-if="tab === 'colors'" class="space-y-4">
      <div>
        <label class="label">Fill</label>
        <div class="grid grid-cols-3 gap-1 rounded-xl bg-subtle p-1">
          <button v-for="g in ['none', 'linear', 'radial']" :key="g" type="button" class="rounded-lg py-1.5 text-xs font-medium capitalize"
            :class="s.gradient_type === g ? 'bg-surface text-fg shadow-card' : 'text-fg-muted'" @click="s.gradient_type = g">
            {{ g === 'none' ? 'Solid' : g }}
          </button>
        </div>
      </div>
      <div class="grid gap-3 sm:grid-cols-2">
        <ColorField v-if="s.gradient_type === 'none'" v-model="s.fg_color" label="Code color" />
        <template v-else>
          <ColorField v-model="s.gradient_start" label="Gradient start" />
          <ColorField v-model="s.gradient_end" label="Gradient end" />
        </template>
        <ColorField v-model="s.bg_color" label="Background" fallback="#ffffff" />
      </div>
      <div v-if="s.gradient_type === 'linear'">
        <label class="label">Angle <span class="font-mono text-fg-subtle">{{ s.gradient_angle }}°</span></label>
        <input v-model.number="s.gradient_angle" type="range" min="0" max="359" class="w-full accent-brand-500" />
      </div>
      <div class="grid gap-3 sm:grid-cols-2">
        <ColorField v-model="s.eye_color" label="Corner frame color" clearable placeholder="Same as code" tip="Leave empty to match the code color." />
        <ColorField v-model="s.eye_inner_color" label="Corner center color" clearable placeholder="Same as frame" />
      </div>
      <label class="flex items-center justify-between rounded-xl border border-line px-3.5 py-2.5 text-sm">
        <span>Transparent background <Tip text="For PNG and SVG. Make sure the surface you print on contrasts with the code." /></span>
        <Toggle v-model="s.is_transparent" />
      </label>
      <button type="button" class="btn-ghost btn-sm" @click="swapColors"><ArrowLeftRight class="h-3.5 w-3.5" /> Swap code & background</button>
    </div>

    <!-- Logo -->
    <div v-else-if="tab === 'logo'" class="space-y-4">
      <div v-if="s.logo_base64" class="flex items-center gap-4 rounded-xl border border-line p-3">
        <img :src="s.logo_base64" alt="Logo" class="h-14 w-14 rounded-lg bg-subtle object-contain p-1" />
        <div class="flex-1 text-xs text-fg-muted">Error correction is locked to High while a logo is used, so the code stays scannable.</div>
        <button type="button" class="btn-danger-ghost btn-sm" @click="s.logo_base64 = ''"><Trash2 class="h-3.5 w-3.5" /> Remove</button>
      </div>
      <label v-else class="flex cursor-pointer flex-col items-center gap-2 rounded-2xl border-2 border-dashed border-line px-6 py-8 text-center hover:border-line-strong hover:bg-subtle/40">
        <input type="file" accept="image/png,image/jpeg,image/webp,image/gif" class="sr-only" data-testid="logo-input" @change="onLogo" />
        <ImagePlus class="h-6 w-6 text-brand-500" />
        <span class="text-sm text-fg"><span class="font-semibold text-brand-500">Upload a logo</span> (PNG, JPEG or WebP, max 500 KB)</span>
        <span class="text-xs text-fg-muted">Square logos with a transparent background look best.</span>
      </label>
      <template v-if="s.logo_base64">
        <div>
          <label class="label">Logo size <span class="font-mono text-fg-subtle">{{ Math.round(s.logo_size * 100) }}%</span>
            <Tip text="Larger logos hide more of the code. Keep it under ~25% for reliable scanning." /></label>
          <input v-model.number="s.logo_size" type="range" min="0.12" max="0.3" step="0.01" class="w-full accent-brand-500" />
        </div>
        <div>
          <label class="label">Logo spacing <span class="font-mono text-fg-subtle">{{ s.logo_padding }}</span></label>
          <input v-model.number="s.logo_padding" type="range" min="0" max="30" class="w-full accent-brand-500" />
        </div>
        <div>
          <label class="label">Background badge</label>
          <div class="grid grid-cols-4 gap-1 rounded-xl bg-subtle p-1">
            <button v-for="b in ['none', 'square', 'rounded', 'circle']" :key="b" type="button" class="rounded-lg py-1.5 text-xs font-medium capitalize"
              :class="s.logo_bg_shape === b ? 'bg-surface text-fg shadow-card' : 'text-fg-muted'" @click="s.logo_bg_shape = b">{{ b }}</button>
          </div>
        </div>
      </template>
    </div>

    <!-- Frame -->
    <div v-else-if="tab === 'frame'" class="space-y-4">
      <ShapePicker label="Frame" kind="frame" :options="opts.frame" v-model="s.frame_style" />
      <template v-if="s.frame_style !== 'none'">
        <div v-if="s.frame_style !== 'border'">
          <label class="label">Call to action <Tip text="Short labels like 'SCAN ME' or 'MENU' boost scan rates. Up to 40 characters." /></label>
          <input v-model="s.label_text" maxlength="40" class="input" placeholder="SCAN ME" data-testid="frame-label" />
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <ColorField v-model="s.border_color" label="Frame color" />
          <ColorField v-if="s.frame_style !== 'border'" v-model="s.label_color" label="Label color" fallback="#ffffff" />
          <ColorField v-model="s.frame_bg_color" label="Frame background" fallback="#ffffff" />
        </div>
        <div class="grid gap-3 sm:grid-cols-3">
          <div>
            <label class="label">Thickness <span class="font-mono text-fg-subtle">{{ s.border_width }}</span></label>
            <input v-model.number="s.border_width" type="range" min="0" max="30" class="w-full accent-brand-500" />
          </div>
          <div>
            <label class="label">Corners <span class="font-mono text-fg-subtle">{{ s.border_radius }}</span></label>
            <input v-model.number="s.border_radius" type="range" min="0" max="64" class="w-full accent-brand-500" />
          </div>
          <div v-if="s.frame_style !== 'border'">
            <label class="label">Label size <span class="font-mono text-fg-subtle">{{ s.label_size }}%</span></label>
            <input v-model.number="s.label_size" type="range" min="50" max="160" step="5" class="w-full accent-brand-500" />
          </div>
        </div>
      </template>
    </div>

    <!-- Advanced -->
    <div v-else class="space-y-5">
      <div>
        <label class="label">Error correction
          <Tip text="Higher levels survive more damage and allow logos, but make the code denser. L≈7%, M≈15%, Q≈25%, H≈30% recovery." /></label>
        <div class="grid grid-cols-4 gap-1 rounded-xl bg-subtle p-1">
          <button v-for="ec in ecLevels" :key="ec.id" type="button" class="rounded-lg py-1.5 text-xs font-medium"
            :disabled="!!s.logo_base64 && ec.id !== 'H'"
            :class="[s.error_correction === ec.id ? 'bg-surface text-fg shadow-card' : 'text-fg-muted', !!s.logo_base64 && ec.id !== 'H' ? 'opacity-40' : '']"
            @click="s.error_correction = ec.id">
            {{ ec.id }} · {{ ec.pct }}
          </button>
        </div>
      </div>
      <div>
        <label class="label">Quiet zone <span class="font-mono text-fg-subtle">{{ s.margin }} modules</span>
          <Tip text="Blank space around the code. Scanners need at least 2 modules; 4 is the spec recommendation for print." /></label>
        <input v-model.number="s.margin" type="range" min="1" max="10" class="w-full accent-brand-500" />
      </div>
      <div v-if="s.frame_style === 'none'">
        <label class="label">Rounded corners <span class="font-mono text-fg-subtle">{{ s.border_radius }}</span></label>
        <input v-model.number="s.border_radius" type="range" min="0" max="64" class="w-full accent-brand-500" />
      </div>
      <button type="button" class="btn-ghost btn-sm" @click="reset"><RotateCcw class="h-3.5 w-3.5" /> Reset design</button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, defineComponent, h } from 'vue'
import { LayoutTemplate, Shapes, Palette, ImagePlus, Frame, SlidersHorizontal, Trash2, ArrowLeftRight, RotateCcw } from 'lucide-vue-next'
import { STYLE_PRESETS, DEFAULT_STYLE } from '../../lib/qrContent'
import { useToast } from '../../composables/useToast'
import ColorField from '../ui/ColorField.vue'
import Tip from '../ui/Tip.vue'
import Toggle from '../ui/Toggle.vue'
import ShapeIcon from './ShapeIcon.vue'

const props = defineProps({ modelValue: { type: Object, required: true } })
const s = computed(() => props.modelValue)
const toast = useToast()
const tab = ref('templates')

const tabs = [
  { id: 'templates', label: 'Templates', icon: LayoutTemplate },
  { id: 'pattern', label: 'Pattern', icon: Shapes },
  { id: 'colors', label: 'Colors', icon: Palette },
  { id: 'logo', label: 'Logo', icon: ImagePlus },
  { id: 'frame', label: 'Frame', icon: Frame },
  { id: 'advanced', label: 'Advanced', icon: SlidersHorizontal }
]
const opts = {
  dot: [['square', 'Square'], ['rounded', 'Rounded'], ['extra-rounded', 'Fluid'], ['dots', 'Dots'], ['classy', 'Classy']],
  eye: [['square', 'Square'], ['rounded', 'Rounded'], ['extra-rounded', 'Soft'], ['circle', 'Circle'], ['leaf', 'Leaf']],
  inner: [['square', 'Square'], ['rounded', 'Rounded'], ['circle', 'Circle'], ['leaf', 'Leaf']],
  frame: [['none', 'None'], ['border', 'Border'], ['banner-bottom', 'Banner'], ['banner-top', 'Top banner'], ['card', 'Card'], ['pill', 'Badge']]
}
const ecLevels = [{ id: 'L', pct: '7%' }, { id: 'M', pct: '15%' }, { id: 'Q', pct: '25%' }, { id: 'H', pct: '30%' }]

function applyPreset(p) {
  Object.assign(s.value, { ...DEFAULT_STYLE, ...p.style, logo_base64: s.value.logo_base64, logo_size: s.value.logo_size, logo_bg_shape: s.value.logo_bg_shape })
}
function reset() {
  Object.assign(s.value, { ...DEFAULT_STYLE })
}
function swapColors() {
  const fg = s.value.fg_color
  s.value.fg_color = s.value.bg_color
  s.value.bg_color = fg
}

function onLogo(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  if (file.size > 500 * 1024) {
    toast.error('Logos must be 500 KB or smaller')
    return
  }
  const reader = new FileReader()
  reader.onload = () => {
    s.value.logo_base64 = reader.result
    s.value.error_correction = 'H'
  }
  reader.readAsDataURL(file)
}

const ShapePicker = defineComponent({
  props: { label: String, tip: String, kind: String, options: Array, modelValue: String },
  emits: ['update:modelValue'],
  setup(p, { emit }) {
    return () =>
      h('div', [
        h('label', { class: 'label' }, [p.label, p.tip ? h(Tip, { text: p.tip }) : null]),
        h(
          'div',
          { class: 'grid grid-cols-3 gap-2 sm:grid-cols-6', role: 'radiogroup', 'aria-label': p.label },
          p.options.map(([id, name]) =>
            h(
              'button',
              {
                type: 'button',
                role: 'radio',
                'aria-checked': p.modelValue === id,
                title: name,
                'data-testid': `${p.kind}-${id}`,
                class: [
                  'flex flex-col items-center gap-1 rounded-xl border px-1 py-2 text-[10px] font-medium transition',
                  p.modelValue === id ? 'border-brand-500 bg-brand-500/10 text-brand-600 dark:text-brand-300' : 'border-line text-fg-muted hover:border-line-strong hover:text-fg'
                ],
                onClick: () => emit('update:modelValue', id)
              },
              [h(ShapeIcon, { kind: p.kind, shape: id }), name]
            )
          )
        )
      ])
  }
})
</script>
