<template>
  <header class="sticky top-0 z-20 flex h-16 items-center gap-3 border-b border-line bg-canvas/80 px-4 backdrop-blur-xl sm:px-6">
    <button class="btn-ghost btn-icon lg:hidden" aria-label="Open navigation" @click="$emit('open-mobile-nav')">
      <Menu class="w-5 h-5" />
    </button>

    <WorkspaceSwitcher v-if="authStore.isAuthenticated" />

    <!-- Search -->
    <div class="relative hidden md:block flex-1 max-w-md">
      <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-subtle" />
      <input
        ref="searchInput"
        v-model="query"
        type="search"
        placeholder="Search links, QR codes, documents…"
        class="input pl-9 pr-10 py-1.5 bg-subtle/60 border-transparent shadow-none"
        aria-label="Search"
        data-testid="global-search"
        @focus="searchOpen = true"
        @keydown.down.prevent="move(1)"
        @keydown.up.prevent="move(-1)"
        @keydown.enter.prevent="go(results[activeIndex])"
        @keydown.esc="blurSearch"
      />
      <span class="kbd absolute right-2.5 top-1/2 -translate-y-1/2">/</span>

      <div v-if="searchOpen && query.trim()" class="absolute left-0 right-0 top-full mt-2 rounded-xl border border-line bg-elevated p-1.5 shadow-pop">
        <p v-if="!results.length" class="px-3 py-6 text-center text-sm text-fg-muted">No matches for “{{ query }}”</p>
        <button
          v-for="(r, i) in results"
          :key="r.key"
          class="flex w-full items-center gap-3 rounded-lg px-2.5 py-2 text-left"
          :class="i === activeIndex ? 'bg-subtle' : 'hover:bg-subtle'"
          @mousedown.prevent="go(r)"
          @mouseenter="activeIndex = i"
        >
          <component :is="r.icon" class="w-4 h-4 text-fg-subtle shrink-0" />
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm text-fg">{{ r.title }}</p>
            <p class="truncate text-xs text-fg-subtle">{{ r.subtitle }}</p>
          </div>
          <span class="badge-neutral">{{ r.kind }}</span>
        </button>
      </div>
    </div>

    <div class="ml-auto flex items-center gap-1.5">
      <!-- Theme -->
      <DropdownMenu>
        <template #trigger>
          <Tip text="Theme" placement="bottom">
            <button class="btn-ghost btn-icon" aria-label="Change theme" data-testid="theme-toggle">
              <Sun v-if="themeStore.effectiveTheme === 'light'" class="w-[18px] h-[18px]" />
              <Moon v-else class="w-[18px] h-[18px]" />
            </button>
          </Tip>
        </template>
        <MenuItem v-for="t in themes" :key="t.value" :icon="t.icon" :data-testid="`theme-${t.value}`" @click="themeStore.setTheme(t.value)">
          <span class="flex items-center justify-between gap-6">
            {{ t.label }}
            <Check v-if="themeStore.preference === t.value" class="w-3.5 h-3.5 text-brand-500" />
          </span>
        </MenuItem>
      </DropdownMenu>

      <!-- Create -->
      <DropdownMenu v-if="authStore.can('assets:edit')">
        <template #trigger>
          <button class="btn-primary btn-sm sm:btn" data-testid="create-new">
            <Plus class="w-4 h-4" /><span class="hidden sm:inline">Create</span>
          </button>
        </template>
        <MenuItem :icon="Link2" shortcut="L" @click="$emit('open-create', 'link')">Short link</MenuItem>
        <MenuItem :icon="QrCode" shortcut="Q" @click="$emit('open-create', 'qr')">QR code</MenuItem>
        <MenuItem :icon="Wand2" to="/qr-studio">Design in QR Studio</MenuItem>
        <MenuItem :icon="FileUp" shortcut="D" @click="$emit('open-create', 'doc')">Upload document</MenuItem>
      </DropdownMenu>

      <!-- User -->
      <DropdownMenu v-if="authStore.isAuthenticated">
        <template #trigger>
          <button class="ml-1 flex items-center gap-2 rounded-full p-0.5 hover:ring-4 hover:ring-subtle transition" aria-label="Account menu" data-testid="user-menu">
            <img v-if="authStore.user.avatar_url" :src="authStore.user.avatar_url" alt="" class="h-8 w-8 rounded-full object-cover" />
            <span v-else class="flex h-8 w-8 items-center justify-center rounded-full bg-gradient-to-br from-brand-500 to-violet-500 text-xs font-bold text-white">
              {{ authStore.initials }}
            </span>
          </button>
        </template>
        <div class="px-2.5 pb-2 pt-1.5 border-b border-line mb-1.5">
          <p class="truncate text-sm font-semibold text-fg">{{ authStore.user.name || authStore.user.email }}</p>
          <p class="truncate text-xs text-fg-subtle">{{ authStore.user.email }}</p>
        </div>
        <MenuItem :icon="UserCircle" to="/profile">Profile & security</MenuItem>
        <MenuItem :icon="KeyRound" to="/profile?tab=api-keys">API keys</MenuItem>
        <MenuItem :icon="Settings" to="/settings">Team settings</MenuItem>
        <div class="my-1.5 border-t border-line" />
        <MenuItem :icon="LogOut" danger data-testid="sign-out" @click="authStore.logout">Sign out</MenuItem>
      </DropdownMenu>
    </div>
  </header>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import {
  Search, Plus, Sun, Moon, Monitor, LogOut, Menu, Check, Link2, QrCode, Wand2, FileUp, UserCircle, KeyRound, Settings, FileText
} from 'lucide-vue-next'
import { useThemeStore } from '../stores/theme'
import { useAuthStore } from '../stores/auth'
import { useLynqrStore } from '../stores/lynqr'
import WorkspaceSwitcher from './WorkspaceSwitcher.vue'
import DropdownMenu from './ui/DropdownMenu.vue'
import MenuItem from './ui/MenuItem.vue'
import Tip from './ui/Tip.vue'

defineEmits(['open-create', 'open-mobile-nav'])
const themeStore = useThemeStore()
const authStore = useAuthStore()
const store = useLynqrStore()
const router = useRouter()

const themes = [
  { value: 'dark', label: 'Dark', icon: Moon },
  { value: 'light', label: 'Light', icon: Sun },
  { value: 'system', label: 'System', icon: Monitor }
]

// ---- Global search across the workspace's assets ----
const query = ref('')
const searchOpen = ref(false)
const activeIndex = ref(0)
const searchInput = ref(null)

const results = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return []
  const match = (...fields) => fields.some((f) => (f || '').toLowerCase().includes(q))
  const links = store.links
    .filter((l) => match(l.title, l.slug, l.target_url))
    .map((l) => ({ key: `l-${l.id}`, kind: 'Link', icon: Link2, title: l.title || l.slug, subtitle: `/s/${l.slug} → ${l.target_url}`, to: `/links?focus=${l.id}` }))
  const qrs = store.qrCodes
    .filter((c) => match(c.title, c.payload, c.type))
    .map((c) => ({ key: `q-${c.id}`, kind: 'QR', icon: QrCode, title: c.title, subtitle: c.type, to: `/qr-studio/${c.id}` }))
  const docs = store.documents
    .filter((d) => match(d.filename))
    .map((d) => ({ key: `d-${d.id}`, kind: 'Doc', icon: FileText, title: d.filename, subtitle: `Version ${d.version}`, to: '/documents' }))
  return [...links, ...qrs, ...docs].slice(0, 8)
})

function move(delta) {
  if (!results.value.length) return
  activeIndex.value = (activeIndex.value + delta + results.value.length) % results.value.length
}
function go(r) {
  if (!r) return
  router.push(r.to)
  blurSearch()
}
function blurSearch() {
  query.value = ''
  searchOpen.value = false
  activeIndex.value = 0
  searchInput.value?.blur()
}

function onKey(e) {
  const typing = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName) || document.activeElement?.isContentEditable
  if (e.key === '/' && !typing) {
    e.preventDefault()
    searchInput.value?.focus()
  }
}
function onDocClick(e) {
  if (searchInput.value && !e.composedPath().includes(searchInput.value.parentElement)) searchOpen.value = false
}
onMounted(() => {
  document.addEventListener('keydown', onKey)
  document.addEventListener('click', onDocClick)
})
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKey)
  document.removeEventListener('click', onDocClick)
})
</script>
