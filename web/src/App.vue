<template>
  <!-- Full Screen for Auth Routes (Sign Up / Sign In) -->
  <div v-if="isAuthRoute" class="min-h-screen bg-gray-950 text-gray-100 font-sans">
    <router-view />
  </div>

  <!-- Standard Dashboard Layout -->
  <div v-else class="flex h-screen overflow-hidden bg-gray-950 text-gray-100 font-sans">
    <!-- Sidebar Navigation -->
    <Sidebar @open-help="isHelpOpen = true" />

    <!-- Main Content Area -->
    <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
      <Header
        @open-create="isCreateOpen = true"
        @search="handleSearch"
      />

      <main class="flex-1 overflow-y-auto p-6 md:p-8 bg-gray-900/40">
        <div class="max-w-7xl mx-auto">
          <router-view @open-create="isCreateOpen = true" />
        </div>
      </main>
    </div>

    <!-- Modals & Drawers -->
    <CreateModal
      :isOpen="isCreateOpen"
      @close="isCreateOpen = false"
      @created="onAssetCreated"
    />

    <HelpDrawer
      :isOpen="isHelpOpen"
      @close="isHelpOpen = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import Sidebar from './components/Sidebar.vue'
import Header from './components/Header.vue'
import CreateModal from './components/CreateModal.vue'
import HelpDrawer from './components/HelpDrawer.vue'
import { useLynqrStore } from './stores/lynqr'

const route = useRoute()
const store = useLynqrStore()
const isCreateOpen = ref(false)
const isHelpOpen = ref(false)

const isAuthRoute = computed(() => {
  return route.path === '/signup' || route.path === '/login'
})

function handleSearch(q) {
  // Filter store links or QRs
}

function onAssetCreated() {
  store.refreshAll()
}

onMounted(() => {
  store.refreshAll()
})
</script>
