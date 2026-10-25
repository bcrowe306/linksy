<template>
  <div class="relative" ref="root" data-testid="workspace-switcher">
    <button
      @click="open = !open"
      class="flex items-center gap-2 px-2 py-1.5 rounded-xl hover:bg-subtle text-left transition-colors max-w-[220px]"
      :title="`Workspace: ${auth.team?.name || ''} (${auth.role || ''})`"
      aria-haspopup="listbox"
      :aria-expanded="open"
    >
      <div class="w-6 h-6 rounded-md bg-gradient-to-br from-brand-500 to-violet-500 text-white flex items-center justify-center text-[11px] font-bold shrink-0">
        {{ (auth.team?.name || '?').charAt(0).toUpperCase() }}
      </div>
      <div class="min-w-0 hidden sm:block">
        <div class="text-sm font-semibold text-fg truncate">{{ auth.team?.name || 'Workspace' }}</div>
        <div class="text-[11px] text-fg-subtle capitalize">{{ auth.role }}</div>
      </div>
      <ChevronsUpDown class="w-3.5 h-3.5 text-fg-subtle shrink-0" />
    </button>

    <div v-if="open" class="absolute left-0 mt-2 w-72 bg-elevated border border-line rounded-xl shadow-pop z-50 p-1.5 animate-scale-in origin-top-left" role="listbox">
      <div class="px-2.5 py-1.5 text-[10px] uppercase tracking-wider text-fg-subtle font-semibold">Workspaces</div>
      <button
        v-for="t in auth.teams"
        :key="t.id"
        @click="choose(t)"
        :disabled="t.status !== 'active'"
        class="w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left hover:bg-subtle disabled:opacity-50 disabled:cursor-not-allowed"
        role="option"
        :aria-selected="t.id === auth.team?.id"
      >
        <div class="w-7 h-7 rounded-md bg-subtle text-fg flex items-center justify-center text-xs font-bold shrink-0">
          {{ t.name.charAt(0).toUpperCase() }}
        </div>
        <div class="min-w-0 flex-1">
          <div class="text-sm text-fg truncate">{{ t.name }}</div>
          <div class="text-[11px] text-fg-subtle">
            <span class="capitalize">{{ t.role }}</span> · {{ t.member_count }} member{{ t.member_count === 1 ? '' : 's' }}
            <span v-if="t.status !== 'active'" class="text-amber-400"> · suspended</span>
          </div>
        </div>
        <Check v-if="t.id === auth.team?.id" class="w-4 h-4 text-brand-500" />
      </button>

      <div class="border-t border-line mt-1.5 pt-1.5">
        <form v-if="creating" @submit.prevent="create" class="p-2 space-y-2">
          <input v-model="newName" maxlength="80" autofocus placeholder="Workspace name"
            class="w-full bg-gray-800 border border-gray-700 rounded-lg px-2.5 py-1.5 text-sm text-fg focus:outline-none focus:ring-2 focus:ring-lynqr-500" />
          <p v-if="error" class="text-[11px] text-rose-400">{{ error }}</p>
          <div class="flex gap-2">
            <button type="submit" :disabled="!newName.trim() || busy" class="btn-primary btn-sm flex-1">Create</button>
            <button type="button" @click="creating = false" class="btn-ghost btn-sm">Cancel</button>
          </div>
        </form>
        <button v-else @click="creating = true" class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-sm text-fg-muted hover:bg-subtle">
          <Plus class="w-4 h-4" /> New workspace
        </button>
        <router-link v-if="auth.pendingInvitations" to="/settings" @click="open = false"
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-sm text-brand-500 hover:bg-subtle">
          <Mail class="w-4 h-4" /> {{ auth.pendingInvitations }} pending invitation{{ auth.pendingInvitations === 1 ? '' : 's' }}
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { ChevronsUpDown, Check, Plus, Mail } from 'lucide-vue-next'
import { useAuthStore } from '../stores/auth'
import { api } from '../api/client'

const auth = useAuthStore()
const open = ref(false)
const creating = ref(false)
const newName = ref('')
const busy = ref(false)
const error = ref('')
const root = ref(null)

async function choose(t) {
  open.value = false
  if (t.id !== auth.team?.id) await auth.switchTeam(t.id)
}

async function create() {
  busy.value = true
  error.value = ''
  try {
    await api.createTeam(newName.value.trim())
    await auth.refresh()
    newName.value = ''
    creating.value = false
    open.value = false
  } catch (err) {
    error.value = err.message
  } finally {
    busy.value = false
  }
}

// composedPath() still includes elements that were re-rendered away by the click itself.
function onClickOutside(e) {
  if (root.value && !e.composedPath().includes(root.value)) open.value = false
}
onMounted(() => document.addEventListener('click', onClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', onClickOutside))
</script>
