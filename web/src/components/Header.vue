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
      <!-- Theme Switcher -->
      <button
        @click="themeStore.toggleTheme"
        class="p-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
        :title="themeStore.currentTheme === 'dark' ? 'Switch to Light Theme' : 'Switch to Dark Theme'"
      >
        <Sun v-if="themeStore.currentTheme === 'dark'" class="w-4 h-4" />
        <Moon v-else class="w-4 h-4" />
      </button>

      <!-- Create New Dropdown / Button -->
      <button
        @click="$emit('open-create')"
        class="bg-linksy-600 hover:bg-linksy-500 text-white px-4 py-2 rounded-lg text-sm font-semibold flex items-center gap-2 shadow-lg shadow-linksy-600/20 transition-all hover:scale-[1.02] active:scale-[0.98]"
      >
        <Plus class="w-4 h-4 stroke-[2.5]" />
        <span>Create New</span>
      </button>

      <!-- Team Badge / Profile -->
      <div class="flex items-center gap-2.5 pl-3 border-l border-gray-800">
        <div class="w-8 h-8 rounded-full bg-gradient-to-tr from-linksy-600 to-indigo-400 flex items-center justify-center font-bold text-xs text-white">
          JD
        </div>
        <div class="hidden md:flex flex-col text-left">
          <span class="text-xs font-semibold text-gray-200">Main Workspace</span>
          <span class="text-[10px] text-linksy-400 font-mono">Owner</span>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup>
import { ref } from 'vue'
import { Search, Plus, Sun, Moon } from 'lucide-vue-next'
import { useThemeStore } from '../stores/theme'

defineEmits(['open-create', 'search'])
const themeStore = useThemeStore()
const searchQuery = ref('')
</script>
