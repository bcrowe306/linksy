<template>
  <!-- Auth, invitation and password-reset pages render without the app chrome -->
  <div v-if="isBareRoute" class="min-h-screen bg-canvas text-fg">
    <router-view />
  </div>

  <div v-else class="flex h-screen overflow-hidden bg-canvas text-fg">
    <Sidebar
      :collapsed="sidebarCollapsed"
      :mobile-open="mobileNavOpen"
      @open-help="isHelpOpen = true"
      @toggle-collapse="toggleCollapse"
      @close-mobile="mobileNavOpen = false"
    />

    <div class="flex min-w-0 flex-1 flex-col overflow-hidden">
      <Header @open-create="openCreate" @open-mobile-nav="mobileNavOpen = true" />
      <main class="flex-1 overflow-y-auto">
        <div class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
          <router-view @open-create="openCreate" />
        </div>
      </main>
    </div>

    <CreateModal :isOpen="isCreateOpen" :initial-tab="createTab" @close="isCreateOpen = false" @created="onAssetCreated" />
    <HelpDrawer :isOpen="isHelpOpen" @close="isHelpOpen = false" />
  </div>

  <ToastHost />
  <ConfirmHost />
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import Sidebar from './components/Sidebar.vue'
import Header from './components/Header.vue'
import CreateModal from './components/CreateModal.vue'
import HelpDrawer from './components/HelpDrawer.vue'
import ToastHost from './components/ui/ToastHost.vue'
import ConfirmHost from './components/ui/ConfirmHost.vue'
import { useLynqrStore } from './stores/lynqr'
import { useAuthStore } from './stores/auth'

const route = useRoute()
const store = useLynqrStore()
const authStore = useAuthStore()
const isCreateOpen = ref(false)
const createTab = ref('link')
const isHelpOpen = ref(false)
const mobileNavOpen = ref(false)
const sidebarCollapsed = ref(localStorage.getItem('lynqr_sidebar_collapsed') === '1')

const isBareRoute = computed(() => !!route.meta.bare)

function toggleCollapse() {
  sidebarCollapsed.value = !sidebarCollapsed.value
  localStorage.setItem('lynqr_sidebar_collapsed', sidebarCollapsed.value ? '1' : '0')
}

function openCreate(tab = 'link') {
  createTab.value = typeof tab === 'string' ? tab : 'link'
  isCreateOpen.value = true
}

function onAssetCreated() {
  store.refreshAll()
}

// Load workspace data whenever the user signs in or switches workspace.
watch(
  () => authStore.isAuthenticated && authStore.team?.id,
  (teamId) => {
    if (teamId) store.refreshAll()
    else store.reset()
  },
  { immediate: true }
)

defineExpose({ openCreate, isHelpOpen })
</script>
