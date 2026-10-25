<template>
  <!-- Mobile backdrop -->
  <Transition enter-active-class="transition-opacity duration-200" enter-from-class="opacity-0" leave-active-class="transition-opacity duration-150" leave-to-class="opacity-0">
    <div v-if="mobileOpen" class="fixed inset-0 z-30 bg-black/50 backdrop-blur-sm lg:hidden" @click="$emit('close-mobile')" />
  </Transition>

  <aside
    class="fixed inset-y-0 left-0 z-40 flex flex-col border-r border-line bg-surface transition-all duration-200 lg:static lg:translate-x-0"
    :class="[mobileOpen ? 'translate-x-0' : '-translate-x-full', collapsed ? 'lg:w-[72px]' : 'lg:w-64', 'w-64']"
    aria-label="Primary"
  >
    <div class="flex h-16 items-center justify-between gap-2 px-4 border-b border-line">
      <router-link to="/" class="min-w-0" @click="$emit('close-mobile')">
        <LynqrLogo :icon-only="collapsed" />
      </router-link>
      <button class="btn-ghost btn-icon lg:hidden" aria-label="Close navigation" @click="$emit('close-mobile')">
        <X class="w-4 h-4" />
      </button>
    </div>

    <nav class="flex-1 overflow-y-auto px-3 py-4 space-y-6">
      <div v-for="group in groups" :key="group.label">
        <p v-if="!collapsed" class="px-3 mb-1.5 text-[10px] font-semibold uppercase tracking-[0.12em] text-fg-subtle">{{ group.label }}</p>
        <div class="space-y-0.5">
          <Tip v-for="item in group.items" :key="item.path" :text="collapsed ? item.label : ''" placement="right" class="w-full">
            <router-link
              :to="item.path"
              class="group relative flex w-full items-center gap-3 rounded-xl px-3 py-2 text-sm font-medium transition-colors"
              :class="isActive(item) ? 'bg-brand-500/10 text-brand-600 dark:text-brand-300' : 'text-fg-muted hover:bg-subtle hover:text-fg'"
              :data-testid="`nav-${item.id}`"
              @click="$emit('close-mobile')"
            >
              <span v-if="isActive(item)" class="absolute left-0 top-1/2 -translate-y-1/2 h-5 w-0.5 rounded-r bg-brand-500" />
              <component :is="item.icon" class="w-[18px] h-[18px] shrink-0" />
              <span v-if="!collapsed" class="truncate">{{ item.label }}</span>
              <span v-if="!collapsed && item.badge" class="ml-auto rounded-full bg-subtle px-2 py-0.5 text-[10px] font-semibold text-fg-muted">{{ item.badge }}</span>
            </router-link>
          </Tip>
        </div>
      </div>
    </nav>

    <div class="border-t border-line p-3 space-y-1">
      <button
        class="flex w-full items-center gap-3 rounded-xl px-3 py-2 text-sm font-medium text-fg-muted hover:bg-subtle hover:text-fg"
        @click="$emit('open-help')"
      >
        <LifeBuoy class="w-[18px] h-[18px] shrink-0" />
        <span v-if="!collapsed" class="flex-1 text-left">Help & Quick Docs</span>
        <span v-if="!collapsed" class="kbd">?</span>
      </button>
      <button
        class="hidden lg:flex w-full items-center gap-3 rounded-xl px-3 py-2 text-sm font-medium text-fg-muted hover:bg-subtle hover:text-fg"
        :aria-label="collapsed ? 'Expand sidebar' : 'Collapse sidebar'"
        @click="$emit('toggle-collapse')"
      >
        <PanelLeftOpen v-if="collapsed" class="w-[18px] h-[18px] shrink-0" />
        <PanelLeftClose v-else class="w-[18px] h-[18px] shrink-0" />
        <span v-if="!collapsed">Collapse</span>
      </button>
    </div>
  </aside>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  LayoutDashboard, Link2, QrCode, Wand2, FileText, BarChart3, Code2, Settings, UserCircle, LifeBuoy, X,
  PanelLeftClose, PanelLeftOpen
} from 'lucide-vue-next'
import { useLynqrStore } from '../stores/lynqr'
import LynqrLogo from './ui/LynqrLogo.vue'
import Tip from './ui/Tip.vue'

defineProps({ collapsed: Boolean, mobileOpen: Boolean })
defineEmits(['open-help', 'toggle-collapse', 'close-mobile'])
const store = useLynqrStore()
const route = useRoute()

const groups = computed(() => [
  {
    label: 'Workspace',
    items: [
      { id: 'dashboard', path: '/', label: 'Dashboard', icon: LayoutDashboard, exact: true },
      { id: 'links', path: '/links', label: 'Short Links', icon: Link2, badge: store.links.length || null },
      { id: 'qr-codes', path: '/qr-codes', label: 'QR Codes', icon: QrCode, badge: store.qrCodes.length || null },
      { id: 'qr-studio', path: '/qr-studio', label: 'QR Studio', icon: Wand2 },
      { id: 'documents', path: '/documents', label: 'Documents', icon: FileText, badge: store.documents.length || null },
      { id: 'analytics', path: '/analytics', label: 'Analytics', icon: BarChart3 }
    ]
  },
  {
    label: 'Account',
    items: [
      { id: 'settings', path: '/settings', label: 'Team & Settings', icon: Settings },
      { id: 'profile', path: '/profile', label: 'Profile', icon: UserCircle },
      { id: 'api-docs', path: '/api-docs', label: 'API Reference', icon: Code2 }
    ]
  }
])

function isActive(item) {
  return item.exact ? route.path === item.path : route.path === item.path || route.path.startsWith(item.path + '/')
}
</script>
