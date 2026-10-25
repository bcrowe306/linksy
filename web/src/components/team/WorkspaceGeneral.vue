<template>
  <div class="space-y-6">
    <section class="card p-5 sm:p-6">
      <h2 class="text-sm font-semibold text-fg">Workspace</h2>
      <p class="mb-4 text-xs text-fg-muted">Links, QR codes, documents and analytics belong to the workspace, not to individual members.</p>
      <form class="flex flex-col gap-3 sm:flex-row sm:items-end" @submit.prevent="rename">
        <div class="flex-1">
          <label class="label" for="ws-name">Name</label>
          <input id="ws-name" v-model="name" maxlength="80" required class="input" :disabled="!canManageTeam" data-testid="workspace-name" />
        </div>
        <button v-if="canManageTeam" class="btn-primary" :disabled="busy || name.trim() === auth.team?.name">Save</button>
      </form>
      <dl class="mt-5 grid gap-4 text-sm sm:grid-cols-3">
        <div><dt class="text-xs text-fg-subtle">Your role</dt><dd class="mt-1"><span :class="roleMeta(auth.role).badge" class="capitalize">{{ auth.role }}</span></dd></div>
        <div><dt class="text-xs text-fg-subtle">Members</dt><dd class="mt-1 font-medium text-fg">{{ auth.team?.member_count }}</dd></div>
        <div><dt class="text-xs text-fg-subtle">Created</dt><dd class="mt-1 text-fg">{{ auth.team ? new Date(auth.team.created_at).toLocaleDateString() : '' }}</dd></div>
      </dl>
    </section>

    <section class="card border-rose-500/30 p-5 sm:p-6">
      <h2 class="flex items-center gap-2 text-sm font-semibold text-rose-600 dark:text-rose-400"><AlertTriangle class="h-4 w-4" /> Danger zone</h2>
      <div class="mt-4 divide-y divide-line">
        <div v-if="auth.role !== 'owner'" class="flex flex-col gap-3 py-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p class="text-sm font-medium text-fg">Leave workspace</p>
            <p class="text-xs text-fg-muted">You'll lose access unless someone invites you again.</p>
          </div>
          <button class="btn-danger-ghost" data-testid="leave-workspace" @click="leave">Leave</button>
        </div>
        <div v-if="auth.can('team:delete')" class="flex flex-col gap-3 py-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p class="text-sm font-medium text-fg">Delete workspace</p>
            <p class="text-xs text-fg-muted">Permanently deletes every link, QR code, document and analytics event. Printed codes stop working.</p>
          </div>
          <button class="btn-danger" data-testid="delete-workspace" @click="destroy">Delete workspace</button>
        </div>
        <p v-if="auth.role === 'owner'" class="py-3 text-xs text-fg-muted">To leave a workspace you own, first make another member the owner from the Members tab.</p>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { AlertTriangle } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useAuthStore } from '../../stores/auth'
import { useToast } from '../../composables/useToast'
import { useConfirm } from '../../composables/useConfirm'
import { roleMeta } from '../../lib/roles'

const auth = useAuthStore()
const toast = useToast()
const confirm = useConfirm()
const router = useRouter()
const name = ref(auth.team?.name || '')
const busy = ref(false)
const canManageTeam = computed(() => auth.can('team:manage'))
watch(() => auth.team?.name, (n) => (name.value = n || ''))

async function rename() {
  busy.value = true
  try {
    await api.renameTeam(name.value.trim())
    await auth.refresh()
    toast.success('Workspace renamed')
  } catch (err) {
    toast.error(err.message)
  } finally {
    busy.value = false
  }
}

async function leave() {
  const ok = await confirm({ title: `Leave ${auth.team.name}?`, message: 'You will lose access to its links, QR codes and analytics.', confirmText: 'Leave workspace', danger: true })
  if (!ok) return
  try {
    await api.leaveTeam()
    await auth.refresh()
    toast.success('You left the workspace')
    router.push('/')
  } catch (err) {
    toast.error(err.message)
  }
}

async function destroy() {
  const ws = auth.team.name
  const ok = await confirm({
    title: 'Delete this workspace permanently?',
    message: `All links, QR codes, documents and analytics in "${ws}" are deleted. Printed QR codes and shared links stop working. This cannot be undone.`,
    confirmText: 'Delete forever',
    danger: true,
    requireText: ws
  })
  if (!ok) return
  try {
    await api.deleteTeam(ws)
    await auth.refresh()
    toast.success(`${ws} was deleted`)
    router.push('/')
  } catch (err) {
    toast.error(err.message)
  }
}
</script>
