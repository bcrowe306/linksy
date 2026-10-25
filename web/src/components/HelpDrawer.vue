<template>
  <Teleport to="body">
    <Transition enter-active-class="transition-opacity duration-200" enter-from-class="opacity-0" leave-active-class="transition-opacity duration-150" leave-to-class="opacity-0">
      <div v-if="helpState.open" class="fixed inset-0 z-50 bg-black/40 backdrop-blur-sm" @click="close" />
    </Transition>
    <Transition enter-active-class="transition duration-250 ease-out" enter-from-class="translate-x-full" leave-active-class="transition duration-200 ease-in" leave-to-class="translate-x-full">
      <aside
        v-if="helpState.open"
        class="fixed inset-y-0 right-0 z-50 flex w-full max-w-lg flex-col border-l border-line bg-elevated shadow-pop"
        role="dialog"
        aria-modal="true"
        aria-label="Help and quick docs"
        data-testid="help-drawer"
        @keydown.esc="close"
      >
        <header class="flex items-center gap-2 border-b border-line px-5 py-4">
          <button v-if="current" class="btn-ghost btn-icon -ml-2" aria-label="Back to all guides" @click="helpState.slug = ''"><ArrowLeft class="h-4 w-4" /></button>
          <LifeBuoy v-else class="h-5 w-5 text-brand-500" />
          <h2 class="flex-1 truncate text-base font-semibold text-fg">{{ current ? current.title : 'Help & Quick Docs' }}</h2>
          <button class="btn-ghost btn-sm" @click="openShortcuts"><Keyboard class="h-3.5 w-3.5" /> Shortcuts</button>
          <button class="btn-ghost btn-icon -mr-2" aria-label="Close help" @click="close"><X class="h-4 w-4" /></button>
        </header>

        <!-- Reader -->
        <div v-if="current" ref="reader" class="flex-1 overflow-y-auto px-6 py-5">
          <article class="guide-prose" data-testid="guide-article" v-html="current.html" />
          <div class="mt-8 border-t border-line pt-4">
            <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-fg-subtle">More guides</p>
            <button v-for="g in related" :key="g.slug" class="block w-full rounded-lg px-2 py-1.5 text-left text-sm text-brand-500 hover:bg-subtle" @click="show(g.slug)">{{ g.title }}</button>
          </div>
        </div>

        <!-- Library -->
        <div v-else class="flex-1 overflow-y-auto">
          <div class="p-5 pb-3">
            <div class="relative">
              <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-subtle" />
              <input ref="searchBox" v-model="query" type="search" placeholder="Search guides…" class="input pl-9" aria-label="Search guides" data-testid="help-search" />
            </div>
          </div>
          <ul class="space-y-1 px-3 pb-5">
            <li v-for="g in results" :key="g.slug">
              <button class="w-full rounded-xl px-3 py-2.5 text-left hover:bg-subtle" :data-testid="`guide-${g.slug}`" @click="show(g.slug)">
                <p class="text-sm font-medium text-fg">{{ g.title }}</p>
                <p class="text-xs text-fg-muted">{{ g.description }}</p>
              </button>
            </li>
            <li v-if="!results.length" class="px-3 py-8 text-center text-sm text-fg-muted">No guides match “{{ query }}”.</li>
          </ul>
          <div class="mx-5 mb-5 rounded-xl border border-line p-4">
            <p class="text-sm font-medium text-fg">Building an integration?</p>
            <p class="mb-3 text-xs text-fg-muted">The API reference has every endpoint, with examples in cURL, Go, Python and JavaScript.</p>
            <router-link to="/api-docs" class="btn-secondary btn-sm" @click="close"><Code2 class="h-3.5 w-3.5" /> Open API reference</router-link>
          </div>
        </div>
      </aside>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { LifeBuoy, X, Search, ArrowLeft, Keyboard, Code2 } from 'lucide-vue-next'
import { helpState } from '../composables/useHelp'
import { guides, guideBySlug, searchGuides } from '../lib/guides'

const query = ref('')
const reader = ref(null)
const searchBox = ref(null)
const current = computed(() => guideBySlug(helpState.slug))
const results = computed(() => searchGuides(query.value))
const related = computed(() => guides.filter((g) => g.slug !== helpState.slug).slice(0, 4))

function close() {
  helpState.open = false
}
function show(slug) {
  helpState.slug = slug
  helpState.anchor = ''
}
function openShortcuts() {
  helpState.open = false
  helpState.shortcuts = true
}

// Scroll to a section when opened contextually, e.g. useHelp().open('qr-types', 'wifi').
watch(
  () => [helpState.open, helpState.slug, helpState.anchor],
  async () => {
    if (!helpState.open) return
    await nextTick()
    if (current.value && reader.value) {
      const target = helpState.anchor && reader.value.querySelector(`#${CSS.escape(helpState.anchor)}`)
      target ? target.scrollIntoView({ block: 'start' }) : reader.value.scrollTo(0, 0)
    } else searchBox.value?.focus()
  }
)
</script>

<style scoped>
.guide-prose :deep(h1) { @apply mb-4 text-xl font-bold tracking-tight text-fg; }
.guide-prose :deep(h2) { @apply mb-2 mt-6 scroll-mt-4 text-base font-semibold text-fg; }
.guide-prose :deep(p), .guide-prose :deep(li) { @apply text-sm leading-relaxed text-fg-muted; }
.guide-prose :deep(p) { @apply mb-3; }
.guide-prose :deep(ul), .guide-prose :deep(ol) { @apply mb-3 space-y-1 pl-5; }
.guide-prose :deep(ul) { @apply list-disc; }
.guide-prose :deep(ol) { @apply list-decimal; }
.guide-prose :deep(strong) { @apply font-semibold text-fg; }
.guide-prose :deep(code) { @apply rounded bg-subtle px-1 py-0.5 font-mono text-[12px] text-fg; }
.guide-prose :deep(pre) { @apply mb-3 overflow-x-auto rounded-xl bg-[#0c0c0e] p-3; }
.guide-prose :deep(pre code) { @apply bg-transparent p-0 text-zinc-200; }
.guide-prose :deep(blockquote) { @apply mb-3 rounded-xl border border-brand-500/20 bg-brand-500/5 px-4 py-3; }
.guide-prose :deep(kbd) { @apply kbd; }
.guide-prose :deep(table) { @apply mb-4 w-full text-left text-sm; }
.guide-prose :deep(th) { @apply border-b border-line px-2 py-1.5 text-xs font-semibold text-fg-subtle; }
.guide-prose :deep(td) { @apply border-b border-line px-2 py-1.5 text-fg-muted; }
</style>
