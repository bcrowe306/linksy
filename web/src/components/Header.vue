<template>
  <header class="h-16 bg-gray-900 border-b border-gray-800 px-6 flex items-center justify-between z-10">
    <div class="flex items-center gap-4 flex-1 max-w-xl">
      <!-- Search Input -->
      <div class="relative w-full">
        <Search class="w-4 h-4 text-gray-500 absolute left-3 top-1/2 -translate-y-1/2" />
        <input
          v-model="searchQuery"
          @input="$emit('search', searchQuery)"
          type="text"
          placeholder="Search links, QR codes, tags... (Press '/' to focus)"
          class="w-full bg-gray-800/80 border border-gray-700/80 rounded-lg pl-9 pr-4 py-1.5 text-sm text-gray-200 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-linksy-500 focus:border-transparent transition-all"
        />
      </div>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-3">
      <!-- Theme Switcher (System / Dark / Light) -->
      <button
        @click="themeStore.cycleTheme"
        class="px-2.5 py-1.5 text-xs text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors flex items-center gap-1.5"
        :title="themeTitle"
      >
        <Monitor v-if="themeStore.preference === 'system'" class="w-4 h-4 text-linksy-400" />
        <Moon v-else-if="themeStore.preference === 'dark'" class="w-4 h-4 text-indigo-400" />
        <Sun v-else class="w-4 h-4 text-amber-400" />
        <span class="text-[11px] font-medium capitalize hidden sm:inline">
          {{ themeStore.preference === 'system' ? 'OS Theme' : themeStore.preference }}
        </span>
      </button>

      <!-- Create New Dropdown / Button -->
      <button
        @click="$emit('open-create')"
        class="bg-linksy-600 hover:bg-linksy-500 text-white px-4 py-2 rounded-lg text-sm font-semibold flex items-center gap-2 shadow-lg shadow-linksy-600/20 transition-all hover:scale-[1.02] active:scale-[0.98]"
      >
        <Plus class="w-4 h-4 stroke-[2.5]" />
        <span>Create New</span>
      </button>

      <!-- Authentication Navigation -->
      <div v-if="!authStore.isAuthenticated" class="pl-2 border-l border-gray-800 flex items-center gap-2">
        <router-link
          to="/login"
          class="px-3 py-1.5 rounded-lg text-gray-300 hover:text-white hover:bg-gray-800 text-xs font-semibold transition-colors"
        >
          Sign In
        </router-link>
        <router-link
          to="/signup"
          class="px-3 py-1.5 rounded-lg bg-linksy-600 hover:bg-linksy-500 text-white text-xs font-semibold shadow-md shadow-linksy-600/20 transition-all"
        >
          Sign Up
        </router-link>
      </div>

      <div v-else class="flex items-center gap-2.5 pl-3 border-l border-gray-800">
        <div class="w-8 h-8 rounded-full bg-gradient-to-tr from-linksy-600 to-indigo-400 flex items-center justify-center font-bold text-xs text-white">
          {{ authStore.initials }}
        </div>
        <div class="hidden md:flex flex-col text-left">
          <span class="text-xs font-semibold text-gray-200">{{ authStore.user.name }}</span>
          <span class="text-[10px] text-linksy-400 font-mono">{{ authStore.user.email }}</span>
        </div>
        <button
          @click="authStore.logout"
          title="Sign Out"
          class="p-1.5 rounded-lg text-gray-400 hover:text-rose-400 hover:bg-gray-800 transition-colors ml-1"
        >
          <LogOut class="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  </header>
</template>

<script setup>
import { ref, computed } from 'vue'
import { Search, Plus, Sun, Moon, Monitor, LogOut } from 'lucide-vue-next'
import { useThemeStore } from '../stores/theme'
import { useAuthStore } from '../stores/auth'

defineEmits(['open-create', 'search'])
const themeStore = useThemeStore()
const authStore = useAuthStore()
const searchQuery = ref('')

const themeTitle = computed(() => {
  if (themeStore.preference === 'system') {
    return `Theme: Following OS specification (${themeStore.effectiveTheme})`
  }
  return `Theme: ${themeStore.preference}`
})
</script>
