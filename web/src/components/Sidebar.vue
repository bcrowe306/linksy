<template>
  <aside class="w-64 bg-gray-900 border-r border-gray-800 flex flex-col justify-between select-none">
    <div>
      <!-- Brand Logo -->
      <div class="h-16 flex items-center px-6 gap-3 border-b border-gray-800">
        <div class="w-8 h-8 rounded-lg bg-linksy-600 flex items-center justify-center text-white shadow-lg shadow-linksy-500/20">
          <QrCode class="w-5 h-5" />
        </div>
        <div class="flex flex-col">
          <span class="font-extrabold text-xl tracking-tight text-white flex items-center gap-1.5">
            Linksy
            <span class="text-[10px] px-1.5 py-0.5 rounded bg-linksy-500/20 text-linksy-400 font-mono">v1</span>
          </span>
        </div>
      </div>

      <!-- Navigation Links -->
      <nav class="p-3 space-y-1">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          v-slot="{ isActive }"
          class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all group"
          :class="[
            $route.path === item.path
              ? 'bg-linksy-600/15 text-linksy-400 font-semibold'
              : 'text-gray-400 hover:text-gray-100 hover:bg-gray-800/60'
          ]"
        >
          <component
            :is="item.icon"
            class="w-4 h-4 transition-transform group-hover:scale-110"
            :class="$route.path === item.path ? 'text-linksy-400' : 'text-gray-400 group-hover:text-gray-200'"
          />
          <span>{{ item.label }}</span>
          <span v-if="item.badge" class="ml-auto text-xs px-2 py-0.5 rounded-full bg-gray-800 text-gray-400">
            {{ item.badge }}
          </span>
        </router-link>
      </nav>
    </div>

    <!-- Quick Help & Documentation Card -->
    <div class="p-4 border-t border-gray-800">
      <div class="bg-gradient-to-br from-gray-800/80 to-gray-800/40 rounded-xl p-3.5 border border-gray-700/60">
        <div class="flex items-center gap-2 text-xs font-semibold text-linksy-400 mb-1">
          <BookOpen class="w-3.5 h-3.5" />
          <span>Quick Guidance</span>
        </div>
        <p class="text-[11px] text-gray-400 leading-relaxed mb-3">
          Explore interactive guides, payload types, and dynamic QR tutorials.
        </p>
        <button
          @click="$emit('open-help')"
          class="w-full py-1.5 px-3 rounded-lg bg-gray-700 hover:bg-gray-600 text-xs font-medium text-white transition-colors flex items-center justify-center gap-1.5"
        >
          <HelpCircle class="w-3.5 h-3.5" />
          Open Docs & Help
        </button>
      </div>
    </div>
  </aside>
</template>

<script setup>
import { computed } from 'vue'
import {
  LayoutDashboard,
  Link2,
  QrCode,
  Wand2,
  FileText,
  BarChart3,
  Code2,
  Settings,
  UserCircle,
  BookOpen,
  HelpCircle
} from 'lucide-vue-next'
import { useLinksyStore } from '../stores/linksy'

defineEmits(['open-help'])
const store = useLinksyStore()

const navItems = computed(() => [
  { path: '/', label: 'Dashboard', icon: LayoutDashboard },
  { path: '/links', label: 'Short Links', icon: Link2, badge: store.links.length || null },
  { path: '/qr-codes', label: 'QR Codes', icon: QrCode, badge: store.qrCodes.length || null },
  { path: '/qr-studio', label: 'QR Design Studio', icon: Wand2 },
  { path: '/documents', label: 'Documents', icon: FileText, badge: store.documents.length || null },
  { path: '/analytics', label: 'Analytics', icon: BarChart3 },
  { path: '/api-docs', label: 'API Reference', icon: Code2 },
  { path: '/profile', label: 'Account Profile', icon: UserCircle },
  { path: '/settings', label: 'Team & Settings', icon: Settings },
])
</script>
