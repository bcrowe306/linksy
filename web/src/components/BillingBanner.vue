<template>
  <div v-if="banner" class="border-b px-4 py-2.5 text-sm sm:px-6 lg:px-8" :class="banner.tone" role="status" data-testid="billing-banner">
    <div class="mx-auto flex max-w-7xl flex-wrap items-center gap-x-3 gap-y-1">
      <component :is="banner.icon" class="h-4 w-4 shrink-0" />
      <p class="flex-1 text-fg">{{ banner.text }}</p>
      <router-link v-if="banner.cta" :to="{ path: '/settings', query: { tab: 'billing' } }" class="font-medium text-brand-500 hover:underline">{{ banner.cta }} →</router-link>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { AlertTriangle, Clock, Lock } from 'lucide-vue-next'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()

const banner = computed(() => {
  const b = auth.billing
  if (!b?.enabled && !b?.suspended) return null
  const cta = b.can_manage ? 'Choose a plan' : ''
  const ask = b.can_manage ? '' : ' Ask the workspace owner to subscribe.'
  if (b.suspended) {
    return { icon: Lock, tone: 'border-rose-500/30 bg-rose-500/10', text: 'Lynqr has suspended this workspace. It is read-only and its links and QR codes are disabled. Contact support@lynqr.app.', cta: '' }
  }
  if (b.read_only) {
    return {
      icon: Lock,
      tone: 'border-rose-500/30 bg-rose-500/10',
      text: `${b.state === 'canceled' ? 'Your subscription has ended' : 'Your free trial has ended'}. This workspace is read-only and its links and QR codes are paused.${ask}`,
      cta
    }
  }
  if (b.state === 'past_due') {
    return { icon: AlertTriangle, tone: 'border-amber-500/30 bg-amber-500/10', text: `Your last payment failed. Update your payment method to keep links and QR codes working.${ask}`, cta: b.can_manage ? 'Update billing' : '' }
  }
  if (b.state === 'trial' && b.trial_days_left <= 3) {
    const left = b.trial_days_left <= 0 ? 'ends today' : `ends in ${b.trial_days_left} day${b.trial_days_left === 1 ? '' : 's'}`
    return { icon: Clock, tone: 'border-brand-500/30 bg-brand-500/10', text: `Your free trial ${left}.${ask}`, cta }
  }
  return null
})
</script>
