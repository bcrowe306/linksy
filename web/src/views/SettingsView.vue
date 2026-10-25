<template>
  <div>
    <PageHeader title="Team & Settings" :subtitle="`Manage ${auth.team?.name || 'your workspace'}, its members and its history.`">
      <template #title-extra><HelpLink guide="teams-and-roles">Roles explained</HelpLink></template>
    </PageHeader>

    <nav class="mb-6 flex gap-1 overflow-x-auto border-b border-line" aria-label="Settings sections">
      <router-link
        v-for="t in tabs"
        :key="t.id"
        :to="{ query: { tab: t.id } }"
        class="-mb-px flex shrink-0 items-center gap-2 border-b-2 px-3 py-2.5 text-sm font-medium transition"
        :class="tab === t.id ? 'border-brand-500 text-fg' : 'border-transparent text-fg-muted hover:text-fg'"
        :data-testid="`settings-tab-${t.id}`"
      >
        <component :is="t.icon" class="h-4 w-4" />{{ t.label }}
      </router-link>
    </nav>

    <WorkspaceGeneral v-if="tab === 'general'" :key="auth.team?.id" />
    <MembersPanel v-else-if="tab === 'members'" :key="`m-${auth.team?.id}`" />
    <AuditLogPanel v-else-if="tab === 'audit'" :key="`a-${auth.team?.id}`" />
  </div>
</template>

<script setup>
import HelpLink from '../components/ui/HelpLink.vue'
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Building2, Users, ScrollText } from 'lucide-vue-next'
import { useAuthStore } from '../stores/auth'
import PageHeader from '../components/ui/PageHeader.vue'
import WorkspaceGeneral from '../components/team/WorkspaceGeneral.vue'
import MembersPanel from '../components/team/MembersPanel.vue'
import AuditLogPanel from '../components/team/AuditLogPanel.vue'

const auth = useAuthStore()
const route = useRoute()

const tabs = computed(() => [
  { id: 'members', label: 'Members', icon: Users },
  { id: 'general', label: 'General', icon: Building2 },
  ...(auth.can('audit:view') ? [{ id: 'audit', label: 'Audit log', icon: ScrollText }] : [])
])
const tab = computed(() => (tabs.value.some((t) => t.id === route.query.tab) ? route.query.tab : 'members'))
</script>
